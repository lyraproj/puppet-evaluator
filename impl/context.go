package impl

import (
	"context"
	"fmt"
	"sync"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/threadlocal"
	"github.com/lyraproj/puppet-evaluator/types"
	"github.com/lyraproj/puppet-parser/parser"
	"github.com/lyraproj/puppet-parser/validator"
	"runtime"
)

type (
	evalCtx struct {
		context.Context
		evaluator    eval.Evaluator
		loader       eval.Loader
		logger       eval.Logger
		stack        []issue.Location
		scope        eval.Scope
		implRegistry eval.ImplementationRegistry
		static       bool
		definitions  []interface{}
		vars         map[string]interface{}
	}
)

var resolvableFunctions = make([]eval.ResolvableFunction, 0, 16)
var resolvableFunctionsLock sync.Mutex

func init() {
	eval.Call = func(c eval.Context, name string, args []eval.Value, block eval.Lambda) eval.Value {
		tn := eval.NewTypedName2(`function`, name, c.Loader().NameAuthority())
		if f, ok := eval.Load(c, tn); ok {
			return f.(eval.Function).Call(c, block, args...)
		}
		panic(issue.NewReported(eval.EVAL_UNKNOWN_FUNCTION, issue.SEVERITY_ERROR, issue.H{`name`: tn.String()}, c.StackTop()))
	}

	eval.CurrentContext = func() eval.Context {
		if ctx, ok := threadlocal.Get(eval.PuppetContextKey); ok {
			return ctx.(eval.Context)
		}
		_, file, line, _ := runtime.Caller(1)
		panic(issue.NewReported(eval.EVAL_NO_CURRENT_CONTEXT, issue.SEVERITY_ERROR, issue.NO_ARGS, issue.NewLocation(file, line, 0)))
	}

	eval.RegisterGoFunction = func(function eval.ResolvableFunction) {
		resolvableFunctionsLock.Lock()
		resolvableFunctions = append(resolvableFunctions, function)
		resolvableFunctionsLock.Unlock()
	}
}

func NewContext(evaluatorCtor func(c eval.Context) eval.Evaluator, loader eval.Loader, logger eval.Logger) eval.Context {
	return WithParent(context.Background(), evaluatorCtor, loader, logger, newImplementationRegistry())
}

func WithParent(parent context.Context, evaluatorCtor func(c eval.Context) eval.Evaluator, loader eval.Loader, logger eval.Logger, ir eval.ImplementationRegistry) eval.Context {
	var c *evalCtx
	ir = newParentedImplementationRegistry(ir)
	if cp, ok := parent.(evalCtx); ok {
		c = cp.clone()
		c.Context = parent
		c.loader = loader
		c.logger = logger
		c.evaluator = evaluatorCtor(c)
	} else {
		c = &evalCtx{Context: parent, loader: loader, logger: logger, stack: make([]issue.Location, 0, 8), implRegistry: ir}
		c.evaluator = evaluatorCtor(c)
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

func (c *evalCtx) AddTypes(types ...eval.Type) {
	l := c.DefiningLoader()
	for _, t := range types {
		l.SetEntry(eval.NewTypedName(eval.NsType, t.Name()), eval.NewLoaderEntry(t, nil))
	}
	c.resolveTypes(types...)
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

func (c *evalCtx) DoWithLoader(loader eval.Loader, doer eval.Doer) {
	saveLoader := c.loader
	defer func() {
		c.loader = saveLoader
	}()
	c.loader = loader
	doer()
}

func (c *evalCtx) DoWithScope(scope eval.Scope, doer eval.Doer) {
	saveScope := c.scope
	defer func() {
		c.scope = saveScope
	}()
	c.scope = scope
	doer()
}

func (c *evalCtx) Error(location issue.Location, issueCode issue.Code, args issue.H) issue.Reported {
	if location == nil {
		location = c.StackTop()
	}
	return issue.NewReported(issueCode, issue.SEVERITY_ERROR, args, location)
}

func (c *evalCtx) GetEvaluator() eval.Evaluator {
	return c.evaluator
}

func (c *evalCtx) Fork() eval.Context {
	s := make([]issue.Location, len(c.stack))
	copy(s, c.stack)
	clone := c.clone()
	if clone.scope != nil {
		clone.scope = NewParentedScope(clone.scope.Fork(), false)
	}
	clone.loader = eval.NewParentedLoader(clone.loader)
	clone.implRegistry = newParentedImplementationRegistry(clone.implRegistry)
	clone.evaluator = NewEvaluator(clone)
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
	return c.logger
}

func (c *evalCtx) ParseAndValidate(filename, str string, singleExpression bool) parser.Expression {
	var parserOptions []parser.Option
	if eval.GetSetting(`workflow`, types.Boolean_FALSE).(*types.BooleanValue).Bool() {
		parserOptions = append(parserOptions, parser.PARSER_WORKFLOW_ENABLED)
	}
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

func (c *evalCtx) ParseType(typeString eval.Value) eval.Type {
	if sv, ok := typeString.(*types.StringValue); ok {
		return c.ParseType2(sv.String())
	}
	panic(types.NewIllegalArgumentType2(`ParseType`, 0, `String`, typeString))
}

func (c *evalCtx) ParseType2(str string) eval.Type {
	return c.ResolveType(c.ParseAndValidate(``, str, true))
}

func (c *evalCtx) Reflector() eval.Reflector {
	return types.NewReflector(c)
}

func (c *evalCtx) ResolveDefinitions() []interface{} {
	if c.definitions == nil || len(c.definitions) == 0 {
		return []interface{}{}
	}

	defs := c.definitions
	c.definitions = nil
	ts := make([]eval.Type, 0, 8)
	for _, d := range defs {
		switch d.(type) {
		case eval.Resolvable:
			d.(eval.Resolvable).Resolve(c)
		case eval.ResolvableType:
			ts = append(ts, d.(eval.Type))
		}
	}
	if len(ts) > 0 {
		c.resolveTypes(ts...)
	}
	return defs
}

func (c *evalCtx) ResolveResolvables() {
	l := c.Loader().(eval.DefiningLoader)
	ts := types.PopDeclaredTypes()
	for _, rt := range ts {
		l.SetEntry(eval.NewTypedName(eval.NsType, rt.Name()), eval.NewLoaderEntry(rt, nil))
	}
	c.resolveTypes(ts...)

	ctors := types.PopDeclaredConstructors()
	for _, ct := range ctors {
		rf := eval.BuildFunction(ct.Name, ct.LocalTypes, ct.Creators)
		l.SetEntry(eval.NewTypedName(eval.NsConstructor, rf.Name()), eval.NewLoaderEntry(rf.Resolve(c), nil))
	}

	funcs := popDeclaredGoFunctions()
	for _, rf := range funcs {
		l.SetEntry(eval.NewTypedName(eval.NsFunction, rf.Name()), eval.NewLoaderEntry(rf.Resolve(c), nil))
	}
}

func (c *evalCtx) ResolveType(expr parser.Expression) eval.Type {
	var resolved eval.Value
	c.DoStatic(func() {
		resolved = eval.Evaluate(c, expr)
	})
	if pt, ok := resolved.(eval.Type); ok {
		return pt
	}
	panic(fmt.Sprintf(`Expression "%s" does no resolve to a Type`, expr.String()))
}

func (c *evalCtx) Scope() eval.Scope {
	if c.scope == nil {
		c.scope = NewScope(false)
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

// clone a new context from this context which is an exact copy except for the parent
// of the clone which is set to the original. It is used internally by Fork
func (c *evalCtx) clone() *evalCtx {
	clone := &evalCtx{}
	*clone = *c
	clone.Context = c
	return clone
}

func (c *evalCtx) define(loader eval.DefiningLoader, d parser.Definition) {
	var ta interface{}
	var tn eval.TypedName
	switch d.(type) {
	case *parser.ActivityExpression:
		wf := d.(*parser.ActivityExpression)
		tn = eval.NewTypedName2(eval.NsActivity, wf.Name(), loader.NameAuthority())
		ta = NewPuppetActivity(c, wf)
	case *parser.PlanDefinition:
		pe := d.(*parser.PlanDefinition)
		tn = eval.NewTypedName2(eval.NsPlan, pe.Name(), loader.NameAuthority())
		ta = NewPuppetPlan(pe)
	case *parser.FunctionDefinition:
		fe := d.(*parser.FunctionDefinition)
		tn = eval.NewTypedName2(eval.NsFunction, fe.Name(), loader.NameAuthority())
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

func (c *evalCtx) resolveTypes(types ...eval.Type) {
	l := c.DefiningLoader()
	typeSets := make([]eval.TypeSet, 0)
	allAnnotated := make([]eval.Annotatable, 0, len(types))
	for _, t := range types {
		if rt, ok := t.(eval.ResolvableType); ok {
			rt.Resolve(c)
			var ts eval.TypeSet
			if ts, ok = rt.(eval.TypeSet); ok {
				typeSets = append(typeSets, ts)
			} else {
				var ot eval.ObjectType
				if ot, ok = rt.(eval.ObjectType); ok {
					if ctor := ot.Constructor(c); ctor != nil {
						l.SetEntry(eval.NewTypedName(eval.NsConstructor, t.Name()), eval.NewLoaderEntry(ctor, nil))
					}
				}
			}
		}
		if a, ok := t.(eval.Annotatable); ok {
			allAnnotated = append(allAnnotated, a)
		}
	}

	for _, ts := range typeSets {
		allAnnotated = c.resolveTypeSet(l, ts, allAnnotated)
	}

	// Validate type annotations
	for _, a := range allAnnotated {
		a.Annotations(c).EachValue(func(v eval.Value) {
			v.(eval.Annotation).Validate(c, a)
		})
	}
}

func (c *evalCtx) resolveTypeSet(l eval.DefiningLoader, ts eval.TypeSet, allAnnotated []eval.Annotatable) []eval.Annotatable {
	ts.Types().EachValue(func(tv eval.Value) {
		t := tv.(eval.Type)
		if tsc, ok := t.(eval.TypeSet); ok {
			allAnnotated = c.resolveTypeSet(l, tsc, allAnnotated)
		}
		// Types already known to the loader might have been added to a TypeSet. When that
		// happens, we don't want them added again.
		tn := eval.NewTypedName(eval.NsType, t.Name())
		le := l.LoadEntry(c, tn)
		if le == nil || le.Value() == nil {
			if a, ok := t.(eval.Annotatable); ok {
				allAnnotated = append(allAnnotated, a)
			}
			l.SetEntry(tn, eval.NewLoaderEntry(t, nil))
			if ot, ok := t.(eval.ObjectType); ok {
				if ctor := ot.Constructor(c); ctor != nil {
					l.SetEntry(eval.NewTypedName(eval.NsConstructor, t.Name()), eval.NewLoaderEntry(ctor, nil))
				}
			}
		}
	})
	return allAnnotated
}

func popDeclaredGoFunctions() (funcs []eval.ResolvableFunction) {
	resolvableFunctionsLock.Lock()
	funcs = resolvableFunctions
	if len(funcs) > 0 {
		resolvableFunctions = make([]eval.ResolvableFunction, 0, 16)
	}
	resolvableFunctionsLock.Unlock()
	return
}
