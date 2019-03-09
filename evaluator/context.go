package evaluator

import (
	"fmt"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/pcore"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
	"github.com/lyraproj/puppet-evaluator/pdsl"
	"github.com/lyraproj/puppet-parser/parser"
	"github.com/lyraproj/puppet-parser/validator"
)

type (
	evalCtx struct {
		px.Context

		evaluator   pdsl.Evaluator
		scope       pdsl.Scope
		static      bool
		definitions []interface{}
	}
)

func NewContext(evaluatorCtor func(c pdsl.EvaluationContext) pdsl.Evaluator, loader px.Loader, logger px.Logger) pdsl.EvaluationContext {
	c := &evalCtx{Context: pcore.NewContext(loader, logger)}
	c.evaluator = evaluatorCtor(c)
	return c
}

func WithParent(
	parent px.Context,
	evaluatorCtor func(c pdsl.EvaluationContext) pdsl.Evaluator,
	loader px.Loader,
	logger px.Logger,
	ir px.ImplementationRegistry) pdsl.EvaluationContext {
	var c *evalCtx
	if cp, ok := parent.(*evalCtx); ok {
		c = cp.clone()
		c.Context = pcore.WithParent(cp.Context, loader, logger, ir)
	} else {
		c = &evalCtx{Context: pcore.WithParent(parent, loader, logger, ir)}
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

func (c *evalCtx) DoStatic(doer px.Doer) {
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

func (c *evalCtx) DoWithScope(scope pdsl.Scope, doer px.Doer) {
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

func (c *evalCtx) Fork() px.Context {
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
	if pcore.Get(`workflow`, func() px.Value { return types.BooleanFalse }).(px.Boolean).Bool() {
		parserOptions = append(parserOptions, parser.PARSER_WORKFLOW_ENABLED)
	}
	if pcore.Get(`tasks`, func() px.Value { return types.BooleanFalse }).(px.Boolean).Bool() {
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
			c.Logger().Log(px.LogLevel(i.Severity()), types.WrapString(i.String()))
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
	ts := make([]px.ResolvableType, 0, 8)
	for _, d := range defs {
		switch d := d.(type) {
		case px.Resolvable:
			d.Resolve(c)
		case px.ResolvableType:
			ts = append(ts, d)
		}
	}
	if len(ts) > 0 {
		px.ResolveTypes(c, ts...)
	}
	return defs
}

func (c *evalCtx) ResolveType(expr parser.Expression) px.Type {
	var resolved px.Value
	c.DoStatic(func() {
		resolved = pdsl.Evaluate(c, expr)
	})
	if pt, ok := resolved.(px.Type); ok {
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

func (c *evalCtx) define(loader px.DefiningLoader, d parser.Definition) {
	var ta interface{}
	var tn px.TypedName
	switch d := d.(type) {
	case *parser.ActivityExpression:
		tn = px.NewTypedName2(px.NsActivity, d.Name(), loader.NameAuthority())
		ta = NewPuppetActivity(c, d)
	case *parser.PlanDefinition:
		tn = px.NewTypedName2(px.NsPlan, d.Name(), loader.NameAuthority())
		ta = NewPuppetPlan(d)
	case *parser.FunctionDefinition:
		tn = px.NewTypedName2(px.NsFunction, d.Name(), loader.NameAuthority())
		ta = NewPuppetFunction(d)
	default:
		ta, tn = CreateTypeDefinition(c.evaluator, d, loader.NameAuthority())
	}
	loader.SetEntry(tn, px.NewLoaderEntry(ta, d))
	if c.definitions == nil {
		c.definitions = []interface{}{ta}
	} else {
		c.definitions = append(c.definitions, ta)
	}
}

func CreateTypeDefinition(e pdsl.Evaluator, d parser.Definition, na px.URI) (interface{}, px.TypedName) {
	switch d := d.(type) {
	case *parser.TypeAlias:
		name := d.Name()
		return createTypeDefinition(e, na, name, d.Type()), px.NewTypedName2(px.NsType, name, na)
	default:
		panic(fmt.Sprintf(`Don't know how to define a %T`, d))
	}
}

func createTypeDefinition(e pdsl.Evaluator, na px.URI, name string, body parser.Expression) px.Type {
	var ta px.Type
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

func createMetaType(e pdsl.Evaluator, na px.URI, name string, parentName string, hash *parser.LiteralHash) px.Type {
	if parentName == `` {
		return objectTypeFromAST(e, name, nil, hash)
	}
	return objectTypeFromAST(e, name, types.NewTypeReferenceType(parentName), hash)
}
