package impl

import (
	"context"
	"fmt"

	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/types"
	"github.com/puppetlabs/go-evaluator/yamlparser"
	"github.com/puppetlabs/go-issues/issue"
	"github.com/puppetlabs/go-parser/parser"
	"github.com/puppetlabs/go-parser/validator"
	yaml "gopkg.in/yaml.v2"
)

type (
	evalCtx struct {
		context.Context
		evaluator    eval.Evaluator
		loader       eval.Loader
		stack        []issue.Location
		scope        eval.Scope
		implRegistry eval.ImplementationRegistry
		static       bool
		definitions  []interface{}
		vars         map[string]interface{}
	}
)

func NewContext(evaluator eval.Evaluator, loader eval.Loader) eval.Context {
	return WithParent(context.Background(), evaluator, loader, newImplementationRegistry())
}

func WithParent(parent context.Context, evaluator eval.Evaluator, loader eval.Loader, ir eval.ImplementationRegistry) eval.Context {
	var c *evalCtx
	ir = newParentedImplementationRegistry(ir)
	if cp, ok := parent.(evalCtx); ok {
		c = cp.clone()
		c.Context = parent
		c.evaluator = evaluator
		c.loader = loader
	} else {
		c = &evalCtx{Context: parent, evaluator: evaluator, loader: loader, stack: make([]issue.Location, 0, 8), implRegistry: ir}
	}
	return c
}

func (c *evalCtx) AddDefinitions(expr parser.Expression) {
	if prog, ok := expr.(*parser.Program); ok {
		loader := c.DefiningLoader()
		for _, d := range prog.Definitions() {
			c.define(loader, d)
		}
	}
}

func (c *evalCtx) AddTypes(types ...eval.PType) {
	l := c.DefiningLoader()
	for _, t := range types {
		l.SetEntry(eval.NewTypedName(eval.TYPE, t.Name()), eval.NewLoaderEntry(t, nil))
	}
	c.resolveTypes(types...)
}

func (c *evalCtx) Call(name string, args []eval.PValue, block eval.Lambda) eval.PValue {
	tn := eval.NewTypedName2(`function`, name, c.Loader().NameAuthority())
	if f, ok := eval.Load(c, tn); ok {
		return f.(eval.Function).Call(c, block, args...)
	}
	panic(issue.NewReported(eval.EVAL_UNKNOWN_FUNCTION, issue.SEVERITY_ERROR, issue.H{`name`: tn.String()}, c.StackTop()))
}

func (c *evalCtx) DefiningLoader() eval.DefiningLoader {
	l := c.loader
	for {
		if dl, ok := l.(eval.DefiningLoader); ok {
			return dl
		}
		if pl, ok := l.(eval.ParentedLoader); ok {
			l = pl.Parent()
			continue
		}
		panic(`No defining loader found in context`)
	}
}

func (c *evalCtx) Delete(key string) {
	if c.vars != nil {
		delete(c.vars, key)
	}
}

func (c *evalCtx) DoStatic(actor eval.Actor) {
	if c.static {
		actor()
		return
	}

	defer func() {
		c.static = false
	}()
	c.static = true
	actor()
}

func (c *evalCtx) DoWithLoader(loader eval.Loader, actor eval.Actor) {
	saveLoader := c.loader
	defer func() {
		c.loader = saveLoader
	}()
	c.loader = loader
	actor()
}

func (c *evalCtx) DoWithScope(scope eval.Scope, actor eval.Actor) {
	saveScope := c.scope
	defer func() {
		c.scope = saveScope
	}()
	c.scope = scope
	actor()
}

func (c *evalCtx) Error(location issue.Location, issueCode issue.Code, args issue.H) issue.Reported {
	if location == nil {
		location = c.StackTop()
	}
	return issue.NewReported(issueCode, issue.SEVERITY_ERROR, args, location)
}

func (c *evalCtx) Evaluate(expr parser.Expression) eval.PValue {
	return c.evaluator.Eval(expr, c)
}

func (c *evalCtx) Evaluator() eval.Evaluator {
	return c.evaluator
}

func (c *evalCtx) Fork() eval.Context {
	s := make([]issue.Location, len(c.stack))
	copy(s, c.stack)
	clone := c.clone()
	clone.scope = NewParentedScope(clone.scope)
	clone.loader = eval.NewParentedLoader(clone.loader)
	clone.implRegistry = newParentedImplementationRegistry(clone.implRegistry)
	clone.stack = s

	if c.vars != nil {
		cv := make(map[string]interface{}, len(c.vars))
		for k, v := range c.vars {
			cv[k] = v
		}
		clone.vars = cv
	}
	return clone
}

func (c *evalCtx) Fail(message string) issue.Reported {
	return c.Error(nil, eval.EVAL_FAILURE, issue.H{`message`: message})
}

func (c *evalCtx) Get(key string) (interface{}, bool) {
	if c.vars != nil {
		if v, ok := c.vars[key]; ok {
			return v, true
		}
	}
	return nil, false
}

func (c *evalCtx) ImplementationRegistry() eval.ImplementationRegistry {
	return c.implRegistry
}

func (c *evalCtx) Loader() eval.Loader {
	return c.loader
}

func (c *evalCtx) Logger() eval.Logger {
	return c.evaluator.Logger()
}

func (c *evalCtx) ParseAndValidate(filename, str string, singleExpression bool) parser.Expression {
	var parserOptions []parser.Option
	if eval.GetSetting(`tasks`, types.Boolean_FALSE).(*types.BooleanValue).Bool() {
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
		for _, issue := range issues {
			c.Logger().Log(eval.LogLevel(issue.Severity()), types.WrapString(issue.String()))
			if issue.Severity() > severity {
				severity = issue.Severity()
			}
		}
		if severity == issue.SEVERITY_ERROR {
			c.Fail(fmt.Sprintf(`Error validating %s`, filename))
		}
	}
	return expr
}

func (c *evalCtx) ParseAndValidateYAML(filename string, content []byte) parser.Expression {
	ms := make(yaml.MapSlice, 0)
	err := yaml.Unmarshal(content, &ms)
	if err != nil {
		panic(eval.Error(c, eval.EVAL_PARSE_ERROR, issue.H{`language`: `YAML`, `detail`: err.Error()}))
	}
	yp := yamlparser.NewYAMLParser(c, filename, content)
	return yp.ParseYaml([]string{filename}, ms, true)
}

func (c *evalCtx) ParseType(typeString eval.PValue) eval.PType {
	if sv, ok := typeString.(*types.StringValue); ok {
		return c.ParseType2(sv.String())
	}
	panic(types.NewIllegalArgumentType2(`ParseType`, 0, `String`, typeString))
}

func (c *evalCtx) ParseType2(str string) eval.PType {
	return c.ResolveType(c.ParseAndValidate(``, str, true))
}

func (c *evalCtx) Reflector() eval.Reflector {
	return types.NewReflector(c)
}

func (c *evalCtx) ResolveDefinitions() {
	if c.definitions == nil {
		return
	}

	for len(c.definitions) > 0 {
		defs := c.definitions
		c.definitions = nil
		for _, d := range defs {
			switch d.(type) {
			case *puppetFunction:
				d.(*puppetFunction).Resolve(c)
			case eval.ResolvableType:
				c.resolveTypes(d.(eval.PType))
			}
		}
	}
}

func (c *evalCtx) ResolveResolvables() {
	c.Loader().(eval.DefiningLoader).ResolveResolvables(c)
}

func (c *evalCtx) ResolveType(expr parser.Expression) eval.PType {
	var resolved eval.PValue
	c.DoStatic(func() {
		resolved = c.Evaluate(expr)
	})
	if pt, ok := resolved.(eval.PType); ok {
		return pt
	}
	panic(fmt.Sprintf(`Expression "%s" does no resolve to a Type`, expr.String()))
}

func (c *evalCtx) Scope() eval.Scope {
	if c.scope == nil {
		c.scope = NewScope()
	}
	return c.scope
}

func (c *evalCtx) Set(key string, value interface{}) {
	if c.vars == nil {
		c.vars = map[string]interface{}{key: value}
	} else {
		c.vars[key] = value
	}
}

func (c *evalCtx) Stack() []issue.Location {
	return c.stack
}

func (c *evalCtx) StackPop() {
	c.stack = c.stack[:len(c.stack)-1]
}

func (c *evalCtx) StackPush(location issue.Location) {
	c.stack = append(c.stack, location)
}

func (c *evalCtx) StackTop() issue.Location {
	s := len(c.stack)
	if s == 0 {
		return &systemLocation{}
	}
	return c.stack[s-1]
}

func (c *evalCtx) Static() bool {
	return c.static
}

func (c *evalCtx) WithScope(scope eval.Scope) eval.Context {
	clone := c.clone()
	clone.scope = scope
	return clone
}

// clone a new context from this context which is an exact copy. It is used
// internally by methods like Fork, WithLoader, and WithScope to prevent them
// from creating specific implementations.
func (c *evalCtx) clone() *evalCtx {
	return &evalCtx{c, c.evaluator, c.loader, c.stack, c.scope, c.implRegistry, c.static, c.definitions, c.vars}
}

func (c *evalCtx) define(loader eval.DefiningLoader, d parser.Definition) {
	var ta interface{}
	var tn eval.TypedName
	switch d.(type) {
	case *parser.PlanDefinition:
		pe := d.(*parser.PlanDefinition)
		tn = eval.NewTypedName2(eval.PLAN, pe.Name(), loader.NameAuthority())
		ta = NewPuppetPlan(pe)
	case *parser.FunctionDefinition:
		fe := d.(*parser.FunctionDefinition)
		tn = eval.NewTypedName2(eval.FUNCTION, fe.Name(), loader.NameAuthority())
		ta = NewPuppetFunction(fe)
	default:
		ta, tn = types.CreateTypeDefinition(d, loader.NameAuthority())
	}
	loader.SetEntry(tn, eval.NewLoaderEntry(ta, d))
	if c.definitions == nil {
		c.definitions = []interface{}{ta}
	} else {
		c.definitions = append(c.definitions, ta)
	}
}

func (c *evalCtx) resolveTypes(types ...eval.PType) {
	l := c.DefiningLoader()
	typeSets := make([]eval.TypeSet, 0)
	for _, t := range types {
		if rt, ok := t.(eval.ResolvableType); ok {
			rt.Resolve(c)
			if ts, ok := rt.(eval.TypeSet); ok {
				typeSets = append(typeSets, ts)
			} else if ot, ok := rt.(eval.ObjectType); ok {
				if ctor := ot.Constructor(); ctor != nil {
					l.SetEntry(eval.NewTypedName(eval.CONSTRUCTOR, t.Name()), eval.NewLoaderEntry(ctor, nil))
				}
			}
		}
	}

	for _, ts := range typeSets {
		c.resolveTypeSet(l, ts)
	}
}

func (c *evalCtx) resolveTypeSet(l eval.DefiningLoader, ts eval.TypeSet) {
	ts.Types().EachValue(func(tv eval.PValue) {
		t := tv.(eval.PType)
		if tsc, ok := t.(eval.TypeSet); ok {
			c.resolveTypeSet(l, tsc)
		}
		l.SetEntry(eval.NewTypedName(eval.TYPE, t.Name()), eval.NewLoaderEntry(t, nil))
		if ot, ok := t.(eval.ObjectType); ok {
			if ctor := ot.Constructor(); ctor != nil {
				l.SetEntry(eval.NewTypedName(eval.CONSTRUCTOR, t.Name()), eval.NewLoaderEntry(ctor, nil))
			}
		}
	})
}
