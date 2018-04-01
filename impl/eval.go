package impl

import (
	"bytes"
	"fmt"
	"path"

	"github.com/puppetlabs/go-evaluator/errors"
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/types"
	"github.com/puppetlabs/go-parser/issue"
	"github.com/puppetlabs/go-parser/parser"
	"github.com/puppetlabs/go-parser/validator"
)

var coreTypes = map[string]eval.PType{
	`annotation`:    types.DefaultAnnotationType(),
	`any`:           types.DefaultAnyType(),
	`array`:         types.DefaultArrayType(),
	`binary`:        types.DefaultBinaryType(),
	`boolean`:       types.DefaultBooleanType(),
	`callable`:      types.DefaultCallableType(),
	`collection`:    types.DefaultCollectionType(),
	`data`:          types.DefaultDataType(),
	`default`:       types.DefaultDefaultType(),
	`enum`:          types.DefaultEnumType(),
	`float`:         types.DefaultFloatType(),
	`hash`:          types.DefaultHashType(),
	`integer`:       types.DefaultIntegerType(),
	`iterable`:      types.DefaultIterableType(),
	`iterator`:      types.DefaultIteratorType(),
	`notundef`:      types.DefaultNotUndefType(),
	`numeric`:       types.DefaultNumericType(),
	`optional`:      types.DefaultOptionalType(),
	`object`:        types.DefaultObjectType(),
	`pattern`:       types.DefaultPatternType(),
	`regexp`:        types.DefaultRegexpType(),
	`richdata`:      types.DefaultRichDataType(),
	`runtime`:       types.DefaultRuntimeType(),
	`scalardata`:    types.DefaultScalarDataType(),
	`scalar`:        types.DefaultScalarType(),
	`semver`:        types.DefaultSemVerType(),
	`semverrange`:   types.DefaultSemVerRangeType(),
	`sensitive`:     types.DefaultSensitiveType(),
	`string`:        types.DefaultStringType(),
	`struct`:        types.DefaultStructType(),
	`timespan`:      types.DefaultTimespanType(),
	`timestamp`:     types.DefaultTimestampType(),
	`tuple`:         types.DefaultTupleType(),
	`type`:          types.DefaultTypeType(),
	`typealias`:     types.DefaultTypeAliasType(),
	`typereference`: types.DefaultTypeReferenceType(),
	`typeset`:       types.DefaultTypeSetType(),
	`undef`:         types.DefaultUndefType(),
	`unit`:          types.DefaultUnitType(),
	`uri`:           types.DefaultUriType(),
	`variant`:       types.DefaultVariantType(),
}

type (
	context struct {
		evaluator eval.Evaluator
		loader    eval.Loader
		scope     eval.Scope
		stack     []issue.Location
	}

	evaluator struct {
		self          eval.Evaluator
		logger        eval.Logger
		definitions   []*definition
		defaultLoader eval.DefiningLoader
	}

	definition struct {
		definedValue interface{}
		loader       eval.Loader
	}

	systemLocation struct{}
)

func (systemLocation) File() string {
	return `System`
}

func (systemLocation) Line() int {
	return 0
}

func (systemLocation) Pos() int {
	return 0
}

var currentContext eval.EvalContext

func init() {
	eval.CurrentContext = func() eval.EvalContext {
		return currentContext
	}

	eval.Error = func(issueCode issue.Code, args issue.H) *issue.Reported {
		var location issue.Location
		c := currentContext
		if c == nil {
			location = nil
		} else {
			location = c.StackTop()
		}
		return issue.NewReported(issueCode, issue.SEVERITY_ERROR, args, location)
	}

	eval.Error2 = func(location issue.Location, issueCode issue.Code, args issue.H) *issue.Reported {
		return issue.NewReported(issueCode, issue.SEVERITY_ERROR, args, location)
	}

	eval.Warning = func(issueCode issue.Code, args issue.H) *issue.Reported {
		var location issue.Location
		var logger eval.Logger
		c := currentContext
		if c == nil {
			location = nil
			logger = eval.NewStdLogger()
		} else {
			location = c.StackTop()
			logger = currentContext.Logger()
		}
		ri := issue.NewReported(issueCode, issue.SEVERITY_WARNING, args, location)
		logger.LogIssue(ri)
		return ri
	}
}

func NewEvalContext(eval eval.Evaluator, loader eval.Loader, scope eval.Scope, stack []issue.Location) eval.EvalContext {
	return &context{eval, loader, scope, stack}
}

func NewEvaluator(defaultLoader eval.DefiningLoader, logger eval.Logger) eval.Evaluator {
	e := &evaluator{logger: logger, definitions: make([]*definition, 0, 16), defaultLoader: defaultLoader}
	e.self = e
	return e
}

func NewOverriddenEvaluator(defaultLoader eval.DefiningLoader, logger eval.Logger, specialization eval.Evaluator) eval.Evaluator {
	return &evaluator{self: specialization, logger: logger, definitions: make([]*definition, 0, 16), defaultLoader: defaultLoader}
}

func ResolveResolvables(loader eval.DefiningLoader, logger eval.Logger) {
	loader.ResolveResolvables(&context{NewEvaluator(loader, logger), loader, NewScope(), []issue.Location{}})
}

func (c *context) StackPush(location issue.Location) {
	c.stack = append(c.stack, location)
}

func (c *context) StackPop() {
	c.stack = c.stack[:len(c.stack)-1]
}

func (c *context) StackTop() issue.Location {
	s := len(c.stack)
	if s == 0 {
		return &systemLocation{}
	}
	return c.stack[s-1]
}

func (c *context) Stack() []issue.Location {
	return c.stack
}

func (c *context) Evaluator() eval.Evaluator {
	return c.evaluator
}

func (c *context) ParseType(typeString eval.PValue) eval.PType {
	if sv, ok := typeString.(*types.StringValue); ok {
		return c.ParseResolve(sv.String())
	}
	panic(types.NewIllegalArgumentType2(`ParseType`, 0, `String`, typeString))
}

func (c *context) Resolve(expr parser.Expression) eval.PValue {
	return c.evaluator.Eval(expr, c)
}

func (c *context) ResolveType(expr parser.Expression) eval.PType {
	resolved := c.Resolve(expr)
	if pt, ok := resolved.(eval.PType); ok {
		return pt
	}
	panic(fmt.Sprintf(`Expression "%s" does no resolve to a Type`, expr.String()))
}

func (c *context) Call(name string, args []eval.PValue, block eval.Lambda) eval.PValue {
	tn := eval.NewTypedName2(`function`, name, c.Loader().NameAuthority())
	if f, ok := eval.Load(c.Loader(), tn); ok {
		return f.(eval.Function).Call(c, block, args...)
	}
	panic(issue.NewReported(eval.EVAL_UNKNOWN_FUNCTION, issue.SEVERITY_ERROR, issue.H{`name`: tn.String()}, c.StackTop()))
}

func (c *context) Fail(message string) *issue.Reported {
	return c.Error(nil, eval.EVAL_FAILURE, issue.H{`message`: message})
}

func (c *context) Error(location issue.Location, issueCode issue.Code, args issue.H) *issue.Reported {
	if location == nil {
		location = c.StackTop()
	}
	return issue.NewReported(issueCode, issue.SEVERITY_ERROR, args, location)
}

func (c *context) Evaluate(expr parser.Expression) eval.PValue {
	return c.evaluator.Eval(expr, c)
}

func (c *context) EvaluateIn(expr parser.Expression, scope eval.Scope) eval.PValue {
	return c.evaluator.Eval(expr, c.WithScope(scope))
}

func (c *context) Logger() eval.Logger {
	return c.evaluator.Logger()
}

func (c *context) WithLoader(loader eval.Loader) eval.EvalContext {
	return &context{c.evaluator, loader, c.scope, c.stack}
}

func (c *context) WithScope(scope eval.Scope) eval.EvalContext {
	return &context{c.evaluator, c.loader, scope, c.stack}
}

func (c *context) Loader() eval.Loader {
	return c.loader
}

func (c *context) Scope() eval.Scope {
	return c.scope
}

func (c *context) ParseAndValidate(filename, str string, singleExpression bool) parser.Expression {
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

func (c *context) ParseResolve(str string) eval.PType {
	return c.ResolveType(c.ParseAndValidate(``, str, true))
}

func (e *evaluator) AddDefinitions(expr parser.Expression) {
	if prog, ok := expr.(*parser.Program); ok {
		loader := e.loaderForFile(prog.File())
		for _, d := range prog.Definitions() {
			e.define(loader, d)
		}
	}
}

func (e *evaluator) Evaluate(expr parser.Expression, scope eval.Scope, loader eval.Loader) (result eval.PValue, err *issue.Reported) {
	defer func() {
		if r := recover(); r != nil {
			switch r.(type) {
			case *issue.Reported:
				result = eval.UNDEF
				err = r.(*issue.Reported)
			case *errors.StopIteration:
				result = eval.UNDEF
				err = e.evalError(eval.EVAL_ILLEGAL_BREAK, r.(*errors.StopIteration).Location(), issue.NO_ARGS)
			case *errors.NextIteration:
				result = eval.UNDEF
				err = e.evalError(eval.EVAL_ILLEGAL_NEXT, r.(*errors.NextIteration).Location(), issue.NO_ARGS)
			case *errors.Return:
				result = eval.UNDEF
				err = e.evalError(eval.EVAL_ILLEGAL_RETURN, r.(*errors.Return).Location(), issue.NO_ARGS)
			default:
				panic(r)
			}
		}
	}()

	err = nil
	if loader == nil {
		loader = e.loaderForFile(expr.File())
	}
	ctx := &context{e.self, loader, scope, make([]issue.Location, 0, 64)}
	ctx.StackPush(expr)
	e.ResolveDefinitions(ctx)
	currentContext = ctx
	result = e.eval(expr, ctx)
	return
}

func (e *evaluator) Eval(expr parser.Expression, c eval.EvalContext) eval.PValue {
	currentContext = c
	return e.internalEval(expr, c)
}

func (e *evaluator) Logger() eval.Logger {
	return e.logger
}

func (e *evaluator) callFunction(name string, args []eval.PValue, call parser.CallExpression, c eval.EvalContext) eval.PValue {
	return e.call(`function`, name, args, call, c)
}

func (e *evaluator) call(funcType eval.Namespace, name string, args []eval.PValue, call parser.CallExpression, c eval.EvalContext) (result eval.PValue) {
	if c == nil || c.Loader() == nil {
		panic(`foo`)
	}
	tn := eval.NewTypedName2(funcType, name, c.Loader().NameAuthority())
	f, ok := eval.Load(c.Loader(), tn)
	if !ok {
		panic(e.evalError(eval.EVAL_UNKNOWN_FUNCTION, call, issue.H{`name`: tn.String()}))
	}

	var block eval.Lambda
	if call.Lambda() != nil {
		block = e.Eval(call.Lambda(), c).(eval.Lambda)
	}

	fn := f.(eval.Function)

	c.StackPush(call)
	defer func() {
		c.StackPop()
		if err := recover(); err != nil {
			e.convertCallError(err, call, call.Arguments())
		}
	}()
	result = fn.Call(c, block, args...)
	return
}

func (e *evaluator) convertCallError(err interface{}, expr parser.Expression, args []parser.Expression) {
	switch err.(type) {
	case nil:
	case *errors.ArgumentsError:
		panic(e.evalError(eval.EVAL_ARGUMENTS_ERROR, expr, issue.H{`expression`: expr, `message`: err.(*errors.ArgumentsError).Error()}))
	case *errors.IllegalArgument:
		ia := err.(*errors.IllegalArgument)
		panic(e.evalError(eval.EVAL_ILLEGAL_ARGUMENT, args[ia.Index()], issue.H{`expression`: expr, `number`: ia.Index(), `message`: ia.Error()}))
	case *errors.IllegalArgumentType:
		ia := err.(*errors.IllegalArgumentType)
		panic(e.evalError(eval.EVAL_ILLEGAL_ARGUMENT_TYPE, args[ia.Index()],
			issue.H{`expression`: expr, `number`: ia.Index(), `expected`: ia.Expected(), `actual`: ia.Actual()}))
	case *errors.IllegalArgumentCount:
		iac := err.(*errors.IllegalArgumentCount)
		panic(e.evalError(eval.EVAL_ILLEGAL_ARGUMENT_COUNT, expr, issue.H{`expression`: expr, `expected`: iac.Expected(), `actual`: iac.Actual()}))
	default:
		panic(err)
	}
}

func (e *evaluator) define(loader eval.DefiningLoader, d parser.Definition) {
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
	e.definitions = append(e.definitions, &definition{ta, loader})
}

func (e *evaluator) eval(expr parser.Expression, c eval.EvalContext) eval.PValue {
	v := e.self.Eval(expr, c)
	if iv, ok := v.(eval.IteratorValue); ok {
		// Iterators are never returned. Convert to Array
		return iv.DynamicValue().AsArray()
	}
	return v
}

func (e *evaluator) eval_AndExpression(expr *parser.AndExpression, c eval.EvalContext) eval.PValue {
	return types.WrapBoolean(eval.IsTruthy(e.eval(expr.Lhs(), c)) && eval.IsTruthy(e.eval(expr.Rhs(), c)))
}

func (e *evaluator) eval_OrExpression(expr *parser.OrExpression, c eval.EvalContext) eval.PValue {
	return types.WrapBoolean(eval.IsTruthy(e.eval(expr.Lhs(), c)) || eval.IsTruthy(e.eval(expr.Rhs(), c)))
}

func (e *evaluator) eval_BlockExpression(expr *parser.BlockExpression, c eval.EvalContext) (result eval.PValue) {
	result = eval.UNDEF
	for _, statement := range expr.Statements() {
		result = e.eval(statement, c)
	}
	return result
}

func (e *evaluator) eval_ConcatenatedString(expr *parser.ConcatenatedString, c eval.EvalContext) eval.PValue {
	bld := bytes.NewBufferString(``)
	for _, s := range expr.Segments() {
		bld.WriteString(e.eval(s, c).(*types.StringValue).String())
	}
	return types.WrapString(bld.String())
}

func (e *evaluator) eval_HeredocExpression(expr *parser.HeredocExpression, c eval.EvalContext) eval.PValue {
	return e.eval(expr.Text(), c)
}

func (e *evaluator) eval_CallMethodExpression(call *parser.CallMethodExpression, c eval.EvalContext) eval.PValue {
	fc, ok := call.Functor().(*parser.NamedAccessExpression)
	if !ok {
		panic(e.evalError(validator.VALIDATE_ILLEGAL_EXPRESSION, call.Functor(),
			issue.H{`expression`: call.Functor(), `feature`: `function accessor`, `container`: call}))
	}
	qn, ok := fc.Rhs().(*parser.QualifiedName)
	if !ok {
		panic(e.evalError(validator.VALIDATE_ILLEGAL_EXPRESSION, call.Functor(),
			issue.H{`expression`: call.Functor(), `feature`: `function name`, `container`: call}))
	}
	receiver := e.unfold([]parser.Expression{fc.Lhs()}, c)
	obj := receiver[0]
	if tem, ok := obj.Type().(eval.TypeWithCallableMembers); ok {
		if mbr, ok := tem.Member(qn.Name()); ok {
			var block eval.Lambda
			if call.Lambda() != nil {
				block = e.Eval(call.Lambda(), c).(eval.Lambda)
			}
			return mbr.Call(c, obj, block, e.unfold(call.Arguments(), c))
		}
	}
	return e.callFunction(qn.Name(), e.unfold(call.Arguments(), c, receiver...), call, c)
}

func (e *evaluator) eval_CallNamedFunctionExpression(call *parser.CallNamedFunctionExpression, c eval.EvalContext) eval.PValue {
	fc := call.Functor()
	switch fc.(type) {
	case *parser.QualifiedName:
		return e.callFunction(fc.(*parser.QualifiedName).Name(), e.unfold(call.Arguments(), c), call, c)
	case *parser.QualifiedReference:
		return e.callFunction(`new`, e.unfold(call.Arguments(), c, types.WrapString(fc.(*parser.QualifiedReference).Name())), call, c)
	default:
		panic(e.evalError(validator.VALIDATE_ILLEGAL_EXPRESSION, call.Functor(),
			issue.H{`expression`: call.Functor(), `feature`: `function name`, `container`: call}))
	}
}

func (e *evaluator) eval_IfExpression(expr *parser.IfExpression, c eval.EvalContext) eval.PValue {
	return c.Scope().WithLocalScope(func(s eval.Scope) eval.PValue {
		c = c.WithScope(s)
		if eval.IsTruthy(e.eval(expr.Test(), c)) {
			return e.eval(expr.Then(), c)
		}
		return e.eval(expr.Else(), c)
	})
}

func (e *evaluator) eval_UnlessExpression(expr *parser.UnlessExpression, c eval.EvalContext) eval.PValue {
	return c.Scope().WithLocalScope(func(s eval.Scope) eval.PValue {
		c = c.WithScope(s)
		if !eval.IsTruthy(e.eval(expr.Test(), c)) {
			return e.eval(expr.Then(), c)
		}
		return e.eval(expr.Else(), c)
	})
}

func (e *evaluator) eval_KeyedEntry(expr *parser.KeyedEntry, c eval.EvalContext) eval.PValue {
	return types.WrapHashEntry(e.eval(expr.Key(), c), e.eval(expr.Value(), c))
}

func (e *evaluator) eval_LambdaExpression(expr *parser.LambdaExpression, c eval.EvalContext) eval.PValue {
	return NewPuppetLambda(expr, c.(*context))
}

func (e *evaluator) eval_LiteralHash(expr *parser.LiteralHash, c eval.EvalContext) eval.PValue {
	entries := expr.Entries()
	top := len(entries)
	if top == 0 {
		return eval.EMPTY_MAP
	}
	result := make([]*types.HashEntry, top)
	for idx := 0; idx < top; idx++ {
		result[idx] = e.eval(entries[idx], c).(*types.HashEntry)
	}
	return types.WrapHash(result)
}

func (e *evaluator) eval_LiteralList(expr *parser.LiteralList, c eval.EvalContext) eval.PValue {
	elems := expr.Elements()
	top := len(elems)
	if top == 0 {
		return eval.EMPTY_ARRAY
	}
	result := make([]eval.PValue, top)
	for idx := 0; idx < top; idx++ {
		result[idx] = e.eval(elems[idx], c)
	}
	return types.WrapArray(result)
}

func (e *evaluator) eval_LiteralBoolean(expr *parser.LiteralBoolean) eval.PValue {
	return types.WrapBoolean(expr.Bool())
}

func (e *evaluator) eval_LiteralDefault(expr *parser.LiteralDefault) eval.PValue {
	return types.WrapDefault()
}

func (e *evaluator) eval_LiteralFloat(expr *parser.LiteralFloat) eval.PValue {
	return types.WrapFloat(expr.Float())
}

func (e *evaluator) eval_LiteralInteger(expr *parser.LiteralInteger) eval.PValue {
	return types.WrapInteger(expr.Int())
}

func (e *evaluator) eval_LiteralString(expr *parser.LiteralString) eval.PValue {
	return types.WrapString(expr.StringValue())
}

func (e *evaluator) eval_NotExpression(expr *parser.NotExpression, c eval.EvalContext) eval.PValue {
	return types.WrapBoolean(!eval.IsTruthy(e.eval(expr.Expr(), c)))
}

func (e *evaluator) eval_ParenthesizedExpression(expr *parser.ParenthesizedExpression, c eval.EvalContext) eval.PValue {
	return e.eval(expr.Expr(), c)
}

func (e *evaluator) eval_Program(expr *parser.Program, c eval.EvalContext) eval.PValue {
	c.StackPush(expr)
	defer func() {
		c.StackPop()
	}()
	return e.eval(expr.Body(), c)
}

func (e *evaluator) eval_QualifiedName(expr *parser.QualifiedName) eval.PValue {
	return types.WrapString(expr.Name())
}

func (e *evaluator) eval_QualifiedReference(expr *parser.QualifiedReference, c eval.EvalContext) eval.PValue {
	dcName := expr.DowncasedName()
	pt := coreTypes[dcName]
	if pt != nil {
		return pt
	}
	return e.loadType(expr.Name(), c.Loader())
}

func (e *evaluator) eval_RegexpExpression(expr *parser.RegexpExpression) eval.PValue {
	return types.WrapRegexp(expr.PatternString())
}

func (e *evaluator) eval_CaseExpression(expr *parser.CaseExpression, c eval.EvalContext) eval.PValue {
	return c.Scope().WithLocalScope(func(scope eval.Scope) eval.PValue {
		c = c.WithScope(scope)
		test := e.eval(expr.Test(), c)
		var the_default *parser.CaseOption
		var selected *parser.CaseOption
	options:
		for _, o := range expr.Options() {
			co := o.(*parser.CaseOption)
			for _, cv := range co.Values() {
				cv = unwindParenthesis(cv)
				switch cv.(type) {
				case *parser.LiteralDefault:
					the_default = co
				case *parser.UnfoldExpression:
					if eval.Any2(e.eval(cv, c).(eval.IndexedValue), func(v eval.PValue) bool { return match(expr.Test(), cv, `match`, scope, test, v) }) {
						selected = co
						break options
					}
				default:
					if match(expr.Test(), cv, `match`, scope, test, e.eval(cv, c)) {
						selected = co
						break options
					}
				}
			}
		}
		if selected == nil {
			selected = the_default
		}
		if selected == nil {
			return eval.UNDEF
		}
		return e.eval(selected.Then(), c)
	})
}

func (e *evaluator) eval_SelectorExpression(expr *parser.SelectorExpression, c eval.EvalContext) eval.PValue {
	return c.Scope().WithLocalScope(func(scope eval.Scope) eval.PValue {
		c = c.WithScope(scope)
		test := e.eval(expr.Lhs(), c)
		var the_default *parser.SelectorEntry
		var selected *parser.SelectorEntry
	selectors:
		for _, s := range expr.Selectors() {
			se := s.(*parser.SelectorEntry)
			me := unwindParenthesis(se.Matching())
			switch me.(type) {
			case *parser.LiteralDefault:
				the_default = se
			case *parser.UnfoldExpression:
				if eval.Any2(e.eval(me, c).(eval.IndexedValue), func(v eval.PValue) bool { return match(expr.Lhs(), me, `match`, scope, test, v) }) {
					selected = se
					break selectors
				}
			default:
				if match(expr.Lhs(), me, `match`, scope, test, e.eval(me, c)) {
					selected = se
					break selectors
				}
			}
		}
		if selected == nil {
			selected = the_default
		}
		if selected == nil {
			return eval.UNDEF
		}
		return e.eval(selected.Value(), c)
	})
}

func (e *evaluator) eval_TextExpression(expr *parser.TextExpression, c eval.EvalContext) eval.PValue {
	return types.WrapString(fmt.Sprintf(`%s`, e.eval(expr.Expr(), c).String()))
}

func (e *evaluator) eval_VariableExpression(expr *parser.VariableExpression, c eval.EvalContext) (value eval.PValue) {
	name, ok := expr.Name()
	if ok {
		if value, ok = c.Scope().Get(name); ok {
			return value
		}
		panic(e.evalError(eval.EVAL_UNKNOWN_VARIABLE, expr, issue.H{`name`: name}))
	}
	idx, _ := expr.Index()
	if value, ok = c.Scope().RxGet(int(idx)); ok {
		return value
	}
	panic(e.evalError(eval.EVAL_UNKNOWN_VARIABLE, expr, issue.H{`name`: idx}))
}

func (e *evaluator) eval_UnfoldExpression(expr *parser.UnfoldExpression, c eval.EvalContext) eval.PValue {
	candidate := e.eval(expr.Expr(), c)
	switch candidate.(type) {
	case *types.UndefValue:
		return types.SingletonArray(eval.UNDEF)
	case *types.ArrayValue:
		return candidate
	case *types.HashValue:
		return types.WrapArray3(candidate.(*types.HashValue))
	case eval.IteratorValue:
		return candidate.(eval.IteratorValue).DynamicValue().AsArray()
	default:
		return types.SingletonArray(candidate)
	}
}

func (e *evaluator) evalError(code issue.Code, location issue.Location, args issue.H) *issue.Reported {
	return issue.NewReported(code, issue.SEVERITY_ERROR, args, location)
}

func (e *evaluator) internalEval(expr parser.Expression, c eval.EvalContext) eval.PValue {
	switch expr.(type) {
	case *parser.AccessExpression:
		return e.eval_AccessExpression(expr.(*parser.AccessExpression), c)
	case *parser.AndExpression:
		return e.eval_AndExpression(expr.(*parser.AndExpression), c)
	case *parser.ArithmeticExpression:
		return e.eval_ArithmeticExpression(expr.(*parser.ArithmeticExpression), c)
	case *parser.AssignmentExpression:
		return e.eval_AssignmentExpression(expr.(*parser.AssignmentExpression), c)
	case *parser.BlockExpression:
		return e.eval_BlockExpression(expr.(*parser.BlockExpression), c)
	case *parser.CallMethodExpression:
		return e.eval_CallMethodExpression(expr.(*parser.CallMethodExpression), c)
	case *parser.CallNamedFunctionExpression:
		return e.eval_CallNamedFunctionExpression(expr.(*parser.CallNamedFunctionExpression), c)
	case *parser.CaseExpression:
		return e.eval_CaseExpression(expr.(*parser.CaseExpression), c)
	case *parser.ComparisonExpression:
		return e.eval_ComparisonExpression(expr.(*parser.ComparisonExpression), c)
	case *parser.ConcatenatedString:
		return e.eval_ConcatenatedString(expr.(*parser.ConcatenatedString), c)
	case *parser.HeredocExpression:
		return e.eval_HeredocExpression(expr.(*parser.HeredocExpression), c)
	case *parser.IfExpression:
		return e.eval_IfExpression(expr.(*parser.IfExpression), c)
	case *parser.KeyedEntry:
		return e.eval_KeyedEntry(expr.(*parser.KeyedEntry), c)
	case *parser.LambdaExpression:
		return e.eval_LambdaExpression(expr.(*parser.LambdaExpression), c)
	case *parser.LiteralHash:
		return e.eval_LiteralHash(expr.(*parser.LiteralHash), c)
	case *parser.LiteralList:
		return e.eval_LiteralList(expr.(*parser.LiteralList), c)
	case *parser.MatchExpression:
		return e.eval_MatchExpression(expr.(*parser.MatchExpression), c)
	case *parser.NotExpression:
		return e.eval_NotExpression(expr.(*parser.NotExpression), c)
	case *parser.OrExpression:
		return e.eval_OrExpression(expr.(*parser.OrExpression), c)
	case *parser.ParenthesizedExpression:
		return e.eval_ParenthesizedExpression(expr.(*parser.ParenthesizedExpression), c)
	case *parser.Program:
		return e.eval_Program(expr.(*parser.Program), c)
	case *parser.QualifiedName:
		return e.eval_QualifiedName(expr.(*parser.QualifiedName))
	case *parser.QualifiedReference:
		return e.eval_QualifiedReference(expr.(*parser.QualifiedReference), c)
	case *parser.RegexpExpression:
		return e.eval_RegexpExpression(expr.(*parser.RegexpExpression))
	case *parser.SelectorExpression:
		return e.eval_SelectorExpression(expr.(*parser.SelectorExpression), c)
	case *parser.FunctionDefinition, *parser.TypeAlias, *parser.TypeMapping:
		// All definitions must be processed at this time
		return eval.UNDEF
	case *parser.LiteralBoolean:
		return e.eval_LiteralBoolean(expr.(*parser.LiteralBoolean))
	case *parser.LiteralDefault:
		return e.eval_LiteralDefault(expr.(*parser.LiteralDefault))
	case *parser.LiteralFloat:
		return e.eval_LiteralFloat(expr.(*parser.LiteralFloat))
	case *parser.LiteralInteger:
		return e.eval_LiteralInteger(expr.(*parser.LiteralInteger))
	case *parser.LiteralString:
		return e.eval_LiteralString(expr.(*parser.LiteralString))
	case *parser.LiteralUndef, *parser.Nop:
		return eval.UNDEF
	case *parser.TextExpression:
		return e.eval_TextExpression(expr.(*parser.TextExpression), c)
	case *parser.UnfoldExpression:
		return e.eval_UnfoldExpression(expr.(*parser.UnfoldExpression), c)
	case *parser.UnlessExpression:
		return e.eval_UnlessExpression(expr.(*parser.UnlessExpression), c)
	case *parser.VariableExpression:
		return e.eval_VariableExpression(expr.(*parser.VariableExpression), c)
	default:
		panic(e.evalError(eval.EVAL_UNHANDLED_EXPRESSION, expr, issue.H{`expression`: expr}))
	}
}

func (e *evaluator) loaderForFile(fileName string) eval.DefiningLoader {
	if fileName == `` {
		return e.defaultLoader
	}
	return e.loaderForDir(path.Dir(fileName))
}

func (e *evaluator) loaderForDir(dirName string) eval.DefiningLoader {
	// TODO: Proper handling of module loaders
	return e.defaultLoader
}

func (e *evaluator) loadType(name string, loader eval.Loader) eval.PType {
	tn := eval.NewTypedName2(eval.TYPE, name, loader.NameAuthority())
	found, ok := eval.Load(loader, tn)
	if ok {
		return found.(eval.PType)
	}
	return types.NewTypeReferenceType(name)
}

func (e *evaluator) ResolveDefinitions(c eval.EvalContext) {
	for len(e.definitions) > 0 {
		scope := NewScope()
		defs := e.definitions
		e.definitions = make([]*definition, 0, 16)
		for _, d := range defs {
			tr := &context{e.self, d.loader, scope, c.Stack()}
			currentContext = tr
			switch d.definedValue.(type) {
			case *puppetFunction:
				d.definedValue.(*puppetFunction).Resolve(tr)
			case eval.ResolvableType:
				d.definedValue.(eval.ResolvableType).Resolve(tr)
			}
		}
	}
}

func (e *evaluator) unfold(array []parser.Expression, c eval.EvalContext, initial ...eval.PValue) []eval.PValue {
	result := make([]eval.PValue, len(initial), len(initial)+len(array))
	copy(result, initial)
	for _, ex := range array {
		ex = unwindParenthesis(ex)
		if u, ok := ex.(*parser.UnfoldExpression); ok {
			ev := e.eval(u.Expr(), c)
			switch ev.(type) {
			case *types.ArrayValue:
				result = ev.(*types.ArrayValue).AppendTo(result)
			default:
				result = append(result, ev)
			}
		} else {
			result = append(result, e.eval(ex, c))
		}
	}
	return result
}

func unwindParenthesis(expr parser.Expression) parser.Expression {
	if p, ok := expr.(*parser.ParenthesizedExpression); ok {
		return p.Expr()
	}
	return expr
}
