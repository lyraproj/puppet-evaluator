package impl

import (
	"context"
	"fmt"
	"sync"

	"runtime"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/threadlocal"
	"github.com/lyraproj/puppet-evaluator/types"
	dsl "github.com/lyraproj/puppet-parser/parser"
	"github.com/lyraproj/puppet-parser/validator"
)

type (
	pcoreCtx struct {
		context.Context
		loader       eval.Loader
		logger       eval.Logger
		stack        []issue.Location
		implRegistry eval.ImplementationRegistry
		vars         map[string]interface{}
	}

	evalCtx struct {
		pcoreCtx

		evaluator   eval.Evaluator
		scope       eval.Scope
		static      bool
		definitions []interface{}
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
		panic(issue.NewReported(eval.UnknownFunction, issue.SEVERITY_ERROR, issue.H{`name`: tn.String()}, c.StackTop()))
	}

	eval.AddTypes = addTypes

	eval.CurrentContext = func() eval.Context {
		if ctx, ok := threadlocal.Get(eval.PuppetContextKey); ok {
			return ctx.(eval.Context)
		}
		_, file, line, _ := runtime.Caller(1)
		panic(issue.NewReported(eval.NoCurrentContext, issue.SEVERITY_ERROR, issue.NO_ARGS, issue.NewLocation(file, line, 0)))
	}

	eval.RegisterGoFunction = func(function eval.ResolvableFunction) {
		resolvableFunctionsLock.Lock()
		resolvableFunctions = append(resolvableFunctions, function)
		resolvableFunctionsLock.Unlock()
	}

	eval.ResolveDefinitions = resolveDefinitions
	eval.ResolveResolvables = resolveResolvables
}

func NewContext(evaluatorCtor func(c eval.EvaluationContext) eval.Evaluator, loader eval.Loader, logger eval.Logger) eval.Context {
	return WithParent(context.Background(), evaluatorCtor, loader, logger, newImplementationRegistry())
}

func WithTypeParent(parent context.Context, loader eval.Loader, logger eval.Logger, ir eval.ImplementationRegistry) eval.Context {
	var c *pcoreCtx
	ir = newParentedImplementationRegistry(ir)
	if cp, ok := parent.(pcoreCtx); ok {
		c = cp.clone()
		c.Context = parent
		c.loader = loader
		c.logger = logger
	} else {
		c = &pcoreCtx{Context: parent, loader: loader, logger: logger, stack: make([]issue.Location, 0, 8), implRegistry: ir}
	}
	return c
}

func WithParent(parent context.Context, evaluatorCtor func(c eval.EvaluationContext) eval.Evaluator, loader eval.Loader, logger eval.Logger, ir eval.ImplementationRegistry) eval.Context {
	var c *evalCtx
	ir = newParentedImplementationRegistry(ir)
	if cp, ok := parent.(evalCtx); ok {
		c = cp.clone()
		c.Context = parent
		c.loader = loader
		c.logger = logger
		c.evaluator = evaluatorCtor(c)
	} else {
		c = &evalCtx{pcoreCtx: pcoreCtx{Context: parent, loader: loader, logger: logger, stack: make([]issue.Location, 0, 8), implRegistry: ir}}
		c.evaluator = evaluatorCtor(c)
	}
	return c
}

func (c *evalCtx) AddDefinitions(expr dsl.Expression) {
	if p, ok := expr.(*dsl.Program); ok {
		dl := c.DefiningLoader()
		for _, d := range p.Definitions() {
			c.define(dl, d)
		}
	}
}

func addTypes(c eval.Context, types ...eval.Type) {
	l := c.DefiningLoader()
	rts := make([]eval.ResolvableType, 0, len(types))
	for _, t := range types {
		l.SetEntry(eval.NewTypedName(eval.NsType, t.Name()), eval.NewLoaderEntry(t, nil))
		if rt, ok := t.(eval.ResolvableType); ok {
			rts = append(rts, rt)
		}
	}
	resolveTypes(c, rts...)
}

func (c *pcoreCtx) DefiningLoader() eval.DefiningLoader {
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

func (c *pcoreCtx) Delete(key string) {
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

func (c *pcoreCtx) DoWithLoader(loader eval.Loader, doer eval.Doer) {
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

func (c *pcoreCtx) Error(location issue.Location, issueCode issue.Code, args issue.H) issue.Reported {
	if location == nil {
		location = c.StackTop()
	}
	return issue.NewReported(issueCode, issue.SEVERITY_ERROR, args, location)
}

func (c *evalCtx) GetEvaluator() eval.Evaluator {
	return c.evaluator
}

func (c *pcoreCtx) Fork() eval.Context {
	s := make([]issue.Location, len(c.stack))
	copy(s, c.stack)
	clone := c.clone()
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

// clone a new context from this context which is an exact copy except for the parent
// of the clone which is set to the original. It is used internally by Fork
func (c *evalCtx) clone() *evalCtx {
	clone := &evalCtx{}
	*clone = *c
	clone.Context = c
	return clone
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

func (c *pcoreCtx) Fail(message string) issue.Reported {
	return c.Error(nil, eval.Failure, issue.H{`message`: message})
}

func (c *pcoreCtx) Get(key string) (interface{}, bool) {
	if c.vars != nil {
		if v, ok := c.vars[key]; ok {
			return v, true
		}
	}
	return nil, false
}

func (c *pcoreCtx) ImplementationRegistry() eval.ImplementationRegistry {
	return c.implRegistry
}

func (c *pcoreCtx) Loader() eval.Loader {
	return c.loader
}

func (c *pcoreCtx) Logger() eval.Logger {
	return c.logger
}

func (c *evalCtx) ParseAndValidate(filename, str string, singleExpression bool) dsl.Expression {
	var parserOptions []dsl.Option
	if eval.GetSetting(`workflow`, types.BooleanFalse).(eval.BooleanValue).Bool() {
		parserOptions = append(parserOptions, dsl.PARSER_WORKFLOW_ENABLED)
	}
	if eval.GetSetting(`tasks`, types.BooleanFalse).(eval.BooleanValue).Bool() {
		parserOptions = append(parserOptions, dsl.PARSER_TASKS_ENABLED)
	}
	expr, err := dsl.CreateParser(parserOptions...).Parse(filename, str, singleExpression)
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

func (c *pcoreCtx) ParseType(typeString eval.Value) eval.Type {
	if sv, ok := typeString.(eval.StringValue); ok {
		return c.ParseType2(sv.String())
	}
	panic(types.NewIllegalArgumentType(`ParseType`, 0, `String`, typeString))
}

func (c *pcoreCtx) ParseType2(str string) eval.Type {
	t, err := types.Parse(str)
	if err != nil {
		panic(err)
	}
	if pt, ok := t.(eval.ResolvableType); ok {
		return pt.Resolve(c)
	}
	panic(fmt.Errorf(`expression "%s" does no resolve to a Type`, str))
}

func (c *pcoreCtx) Reflector() eval.Reflector {
	return types.NewReflector(c)
}

func resolveDefinitions(c eval.Context) []interface{} {
	ec, ok := c.(*evalCtx)
	if !ok || len(ec.definitions) == 0 {
		return []interface{}{}
	}

	defs := ec.definitions
	ec.definitions = nil
	ts := make([]eval.ResolvableType, 0, 8)
	for _, d := range defs {
		switch d := d.(type) {
		case eval.Resolvable:
			d.Resolve(ec)
		case eval.ResolvableType:
			ts = append(ts, d)
		}
	}
	if len(ts) > 0 {
		resolveTypes(c, ts...)
	}
	return defs
}

func resolveResolvables(c eval.Context) {
	l := c.Loader().(eval.DefiningLoader)
	ts := types.PopDeclaredTypes()
	for _, rt := range ts {
		l.SetEntry(eval.NewTypedName(eval.NsType, rt.Name()), eval.NewLoaderEntry(rt, nil))
	}

	for _, mp := range types.PopDeclaredMappings() {
		c.ImplementationRegistry().RegisterType(mp.T, mp.R)
	}

	resolveTypes(c, ts...)

	ctors := types.PopDeclaredConstructors()
	for _, ct := range ctors {
		rf := eval.BuildFunction(ct.Name, ct.LocalTypes, ct.Creators)
		l.SetEntry(eval.NewTypedName(eval.NsConstructor, rf.Name()), eval.NewLoaderEntry(rf.Resolve(c), nil))
	}

	fs := popDeclaredGoFunctions()
	for _, rf := range fs {
		l.SetEntry(eval.NewTypedName(eval.NsFunction, rf.Name()), eval.NewLoaderEntry(rf.Resolve(c), nil))
	}
}

func (c *evalCtx) ResolveType(expr dsl.Expression) eval.Type {
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

func (c *pcoreCtx) Set(key string, value interface{}) {
	if c.vars == nil {
		c.vars = map[string]interface{}{key: value}
	} else {
		c.vars[key] = value
	}
}

func (c *pcoreCtx) SetLoader(loader eval.Loader) {
	c.loader = loader
}

func (c *pcoreCtx) Stack() []issue.Location {
	return c.stack
}

func (c *pcoreCtx) StackPop() {
	c.stack = c.stack[:len(c.stack)-1]
}

func (c *pcoreCtx) StackPush(location issue.Location) {
	c.stack = append(c.stack, location)
}

func (c *pcoreCtx) StackTop() issue.Location {
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
func (c *pcoreCtx) clone() *pcoreCtx {
	clone := &pcoreCtx{}
	*clone = *c
	clone.Context = c
	return clone
}

func (c *evalCtx) define(loader eval.DefiningLoader, d dsl.Definition) {
	var ta interface{}
	var tn eval.TypedName
	switch d := d.(type) {
	case *dsl.ActivityExpression:
		tn = eval.NewTypedName2(eval.NsActivity, d.Name(), loader.NameAuthority())
		ta = NewPuppetActivity(c, d)
	case *dsl.PlanDefinition:
		tn = eval.NewTypedName2(eval.NsPlan, d.Name(), loader.NameAuthority())
		ta = NewPuppetPlan(d)
	case *dsl.FunctionDefinition:
		tn = eval.NewTypedName2(eval.NsFunction, d.Name(), loader.NameAuthority())
		ta = NewPuppetFunction(d)
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

func resolveTypes(c eval.Context, types ...eval.ResolvableType) {
	l := c.DefiningLoader()
	typeSets := make([]eval.TypeSet, 0)
	allAnnotated := make([]eval.Annotatable, 0, len(types))
	for _, rt := range types {
		t := rt.Resolve(c)
		if ts, ok := t.(eval.TypeSet); ok {
			typeSets = append(typeSets, ts)
		} else {
			var ot eval.ObjectType
			if ot, ok = t.(eval.ObjectType); ok {
				if ctor := ot.Constructor(c); ctor != nil {
					l.SetEntry(eval.NewTypedName(eval.NsConstructor, t.Name()), eval.NewLoaderEntry(ctor, nil))
				}
			}
		}
		if a, ok := t.(eval.Annotatable); ok {
			allAnnotated = append(allAnnotated, a)
		}
	}

	for _, ts := range typeSets {
		allAnnotated = resolveTypeSet(c, l, ts, allAnnotated)
	}

	// Validate type annotations
	for _, a := range allAnnotated {
		a.Annotations(c).EachValue(func(v eval.Value) {
			v.(eval.Annotation).Validate(c, a)
		})
	}
}

func resolveTypeSet(c eval.Context, l eval.DefiningLoader, ts eval.TypeSet, allAnnotated []eval.Annotatable) []eval.Annotatable {
	ts.Types().EachValue(func(tv eval.Value) {
		t := tv.(eval.Type)
		if tsc, ok := t.(eval.TypeSet); ok {
			allAnnotated = resolveTypeSet(c, l, tsc, allAnnotated)
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

func popDeclaredGoFunctions() (fs []eval.ResolvableFunction) {
	resolvableFunctionsLock.Lock()
	fs = resolvableFunctions
	if len(fs) > 0 {
		resolvableFunctions = make([]eval.ResolvableFunction, 0, 16)
	}
	resolvableFunctionsLock.Unlock()
	return
}
