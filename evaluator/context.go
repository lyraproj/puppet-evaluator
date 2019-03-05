package evaluator

import (
	"fmt"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/eval"
	"github.com/lyraproj/pcore/impl"
	"github.com/lyraproj/pcore/types"
	"github.com/lyraproj/puppet-evaluator/pdsl"
	"github.com/lyraproj/puppet-parser/parser"
	"github.com/lyraproj/puppet-parser/validator"
)

type (
	evalCtx struct {
		eval.Context

		evaluator   pdsl.Evaluator
		scope       pdsl.Scope
		static      bool
		definitions []interface{}
	}
)

func NewContext(evaluatorCtor func(c pdsl.EvaluationContext) pdsl.Evaluator, loader eval.Loader, logger eval.Logger) pdsl.EvaluationContext {
	c := &evalCtx{Context: impl.NewContext(loader, logger)}
	c.evaluator = evaluatorCtor(c)
	return c
}

func WithParent(
	parent eval.Context,
	evaluatorCtor func(c pdsl.EvaluationContext) pdsl.Evaluator,
	loader eval.Loader,
	logger eval.Logger,
	ir eval.ImplementationRegistry) pdsl.EvaluationContext {
	var c *evalCtx
	if cp, ok := parent.(*evalCtx); ok {
		c = cp.clone()
		c.Context = impl.WithParent(cp.Context, loader, logger, ir)
	} else {
		c = &evalCtx{Context: impl.WithParent(parent, loader, logger, ir)}
	}
	c.evaluator = evaluatorCtor(c)
	return c
}

func (c *evalCtx) AddDefinitions(expr parser.Expression) {
	if p, ok := expr.(*parser.Program); ok {
		dl := c.DefiningLoader()
		for _, d := range p.Definitions() {
			c.define(dl, d)
		}
	}
}

func (c *evalCtx) DoStatic(doer eval.Doer) {
	if c.static {
		doer()
		return
	}

	defer func() {
		c.static = false
	}()
	c.static = true
	doer()
}

func (c *evalCtx) DoWithScope(scope pdsl.Scope, doer eval.Doer) {
	saveScope := c.scope
	defer func() {
		c.scope = saveScope
	}()
	c.scope = scope
	doer()
}

func (c *evalCtx) GetEvaluator() pdsl.Evaluator {
	return c.evaluator
}

// clone a new context from this context which is an exact copy except for the parent
// of the clone which is set to the original. It is used internally by Fork
func (c *evalCtx) clone() *evalCtx {
	clone := &evalCtx{}
	*clone = *c
	clone.Context = c
	return clone
}

func (c *evalCtx) Fork() eval.Context {
	clone := c.clone()
	clone.Context = clone.Context.Fork()
	if clone.scope != nil {
		clone.scope = NewParentedScope(clone.scope.Fork(), false)
	}
	clone.evaluator = NewEvaluator(clone)
	return clone
}

func (c *evalCtx) ParseAndValidate(filename, str string, singleExpression bool) parser.Expression {
	var parserOptions []parser.Option
	if eval.GetSetting(`workflow`, types.BooleanFalse).(eval.BooleanValue).Bool() {
		parserOptions = append(parserOptions, parser.PARSER_WORKFLOW_ENABLED)
	}
	if eval.GetSetting(`tasks`, types.BooleanFalse).(eval.BooleanValue).Bool() {
		parserOptions = append(parserOptions, parser.PARSER_TASKS_ENABLED)
	}
	expr, err := parser.CreateParser(parserOptions...).Parse(filename, str, singleExpression)
	if err != nil {
		panic(err)
	}
	checker := validator.NewChecker(validator.STRICT_ERROR)
	checker.Validate(expr)
	issues := checker.Issues()
	if len(issues) > 0 {
		severity := issue.SEVERITY_IGNORE
		for _, i := range issues {
			c.Logger().Log(eval.LogLevel(i.Severity()), types.WrapString(i.String()))
			if i.Severity() > severity {
				severity = i.Severity()
			}
		}
		if severity == issue.SEVERITY_ERROR {
			panic(c.Fail(fmt.Sprintf(`Error validating %s`, filename)))
		}
	}
	return expr
}

func (c *evalCtx) ResolveDefinitions() []interface{} {
	if len(c.definitions) == 0 {
		return []interface{}{}
	}

	defs := c.definitions
	c.definitions = nil
	ts := make([]eval.ResolvableType, 0, 8)
	for _, d := range defs {
		switch d := d.(type) {
		case eval.Resolvable:
			d.Resolve(c)
		case eval.ResolvableType:
			ts = append(ts, d)
		}
	}
	if len(ts) > 0 {
		impl.ResolveTypes(c, ts...)
	}
	return defs
}

func (c *evalCtx) ResolveType(expr parser.Expression) eval.Type {
	var resolved eval.Value
	c.DoStatic(func() {
		resolved = pdsl.Evaluate(c, expr)
	})
	if pt, ok := resolved.(eval.Type); ok {
		return pt
	}
	panic(fmt.Sprintf(`Expression "%s" does no resolve to a Type`, expr.String()))
}

func (c *evalCtx) Scope() pdsl.Scope {
	if c.scope == nil {
		c.scope = NewScope(false)
	}
	return c.scope
}

func (c *evalCtx) Static() bool {
	return c.static
}

func (c *evalCtx) define(loader eval.DefiningLoader, d parser.Definition) {
	var ta interface{}
	var tn eval.TypedName
	switch d := d.(type) {
	case *parser.ActivityExpression:
		tn = eval.NewTypedName2(eval.NsActivity, d.Name(), loader.NameAuthority())
		ta = NewPuppetActivity(c, d)
	case *parser.PlanDefinition:
		tn = eval.NewTypedName2(eval.NsPlan, d.Name(), loader.NameAuthority())
		ta = NewPuppetPlan(d)
	case *parser.FunctionDefinition:
		tn = eval.NewTypedName2(eval.NsFunction, d.Name(), loader.NameAuthority())
		ta = NewPuppetFunction(d)
	default:
		ta, tn = CreateTypeDefinition(c.evaluator, d, loader.NameAuthority())
	}
	loader.SetEntry(tn, eval.NewLoaderEntry(ta, d))
	if c.definitions == nil {
		c.definitions = []interface{}{ta}
	} else {
		c.definitions = append(c.definitions, ta)
	}
}

func CreateTypeDefinition(e pdsl.Evaluator, d parser.Definition, na eval.URI) (interface{}, eval.TypedName) {
	switch d := d.(type) {
	case *parser.TypeAlias:
		name := d.Name()
		return createTypeDefinition(e, na, name, d.Type()), eval.NewTypedName2(eval.NsType, name, na)
	default:
		panic(fmt.Sprintf(`Don't know how to define a %T`, d))
	}
}

func createTypeDefinition(e pdsl.Evaluator, na eval.URI, name string, body parser.Expression) eval.Type {
	var ta eval.Type
	switch body := body.(type) {
	case *parser.QualifiedReference:
		ta = types.NewTypeAliasType(name, types.NewDeferredType(body.Name()), nil)
	case *parser.AccessExpression:
		if lq, ok := body.Operand().(*parser.QualifiedReference); ok {
			tn := lq.Name()
			ta = nil
			if len(body.Keys()) == 1 {
				if hash, ok := body.Keys()[0].(*parser.LiteralHash); ok {
					if tn == `Object` {
						ta = createMetaType(e, na, name, extractParentName(hash), hash)
					} else if tn == `TypeSet` {
						ta = typeSetTypeFromAST(e, na, name, hash)
					}
				}
			}
			if ta == nil {
				ta = types.NewTypeAliasType(name, types.NewDeferredType(tn, deferToArray(e, body.Keys())...), nil)
			}
		}
	case *parser.LiteralHash:
		ta = createMetaType(e, na, name, extractParentName(body), body)
	}
	if ta == nil {
		panic(fmt.Sprintf(`cannot create object from a %T`, body))
	}
	return ta
}

func extractParentName(hash *parser.LiteralHash) string {
	for _, he := range hash.Entries() {
		ke := he.(*parser.KeyedEntry)
		if k, ok := ke.Key().(*parser.LiteralString); ok && k.StringValue() == `parent` {
			if pr, ok := ke.Value().(*parser.QualifiedReference); ok {
				return pr.Name()
			}
		}
	}
	return ``
}

func createMetaType(e pdsl.Evaluator, na eval.URI, name string, parentName string, hash *parser.LiteralHash) eval.Type {
	if parentName == `` {
		return objectTypeFromAST(e, name, nil, hash)
	}
	return objectTypeFromAST(e, name, types.NewTypeReferenceType(parentName), hash)
}
