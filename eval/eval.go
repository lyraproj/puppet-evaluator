package eval

import (
	"bytes"
	. "fmt"
	"path"

	. "github.com/puppetlabs/go-evaluator/errors"
	. "github.com/puppetlabs/go-evaluator/evaluator"
	. "github.com/puppetlabs/go-evaluator/types"
	. "github.com/puppetlabs/go-parser/issue"
	. "github.com/puppetlabs/go-parser/parser"
	"github.com/puppetlabs/go-parser/validator"
)

var coreTypes = map[string]PType{
	`annotation`:    DefaultAnnotationType(),
	`any`:           DefaultAnyType(),
	`array`:         DefaultArrayType(),
	`binary`:        DefaultBinaryType(),
	`boolean`:       DefaultBooleanType(),
	`callable`:      DefaultCallableType(),
	`collection`:    DefaultCollectionType(),
	`data`:          DefaultDataType(),
	`default`:       DefaultDefaultType(),
	`enum`:          DefaultEnumType(),
	`float`:         DefaultFloatType(),
	`hash`:          DefaultHashType(),
	`integer`:       DefaultIntegerType(),
	`iterable`:      DefaultIterableType(),
	`iterator`:      DefaultIteratorType(),
	`notundef`:      DefaultNotUndefType(),
	`numeric`:       DefaultNumericType(),
	`optional`:      DefaultOptionalType(),
	`object`:        DefaultObjectType(),
	`pattern`:       DefaultPatternType(),
	`regexp`:        DefaultRegexpType(),
	`richdata`:      DefaultRichDataType(),
	`runtime`:       DefaultRuntimeType(),
	`scalardata`:    DefaultScalarDataType(),
	`scalar`:        DefaultScalarType(),
	`semver`:        DefaultSemVerType(),
	`semverrange`:   DefaultSemVerRangeType(),
	`sensitive`:     DefaultSensitiveType(),
	`string`:        DefaultStringType(),
	`struct`:        DefaultStructType(),
	`timespan`:      DefaultTimespanType(),
	`timestamp`:     DefaultTimestampType(),
	`tuple`:         DefaultTupleType(),
	`type`:          DefaultTypeType(),
	`typealias`:     DefaultTypeAliasType(),
	`typereference`: DefaultTypeReferenceType(),
	`typeset`:       DefaultTypeSetType(),
	`undef`:         DefaultUndefType(),
	`unit`:          DefaultUnitType(),
	`variant`:       DefaultVariantType(),
}

type (
	context struct {
		evaluator Evaluator
		loader    Loader
		scope     Scope
		stack     []Location
	}

	evaluator struct {
		self          Evaluator
		logger        Logger
		definitions   []*definition
		defaultLoader DefiningLoader
	}

	definition struct {
		definedValue interface{}
		loader       Loader
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

var currentContext EvalContext

func init() {
	CurrentContext = func() EvalContext {
		return currentContext
	}

	Error = func(issueCode IssueCode, args H) *ReportedIssue {
		var location Location
		c := currentContext
		if c == nil {
			location = nil
		} else {
			location = c.StackTop()
		}
		return NewReportedIssue(issueCode, SEVERITY_ERROR, args, location)
	}

	Error2 = func(location Location, issueCode IssueCode, args H) *ReportedIssue {
		return NewReportedIssue(issueCode, SEVERITY_ERROR, args, location)
	}

	Warning = func(issueCode IssueCode, args H) *ReportedIssue {
		var location Location
		var logger Logger
		c := currentContext
		if c == nil {
			location = nil
			logger = NewStdLogger()
		} else {
			location = c.StackTop()
			logger = currentContext.Logger()
		}
		ri := NewReportedIssue(issueCode, SEVERITY_WARNING, args, location)
		logger.LogIssue(ri)
		return ri
	}
}

func NewEvalContext(eval Evaluator, loader Loader, scope Scope, stack []Location) EvalContext {
	return &context{eval, loader, scope, stack}
}

func NewEvaluator(defaultLoader DefiningLoader, logger Logger) Evaluator {
	e := &evaluator{logger: logger, definitions: make([]*definition, 0, 16), defaultLoader: defaultLoader}
	e.self = e
	return e
}

func NewOverriddenEvaluator(defaultLoader DefiningLoader, logger Logger, specialization Evaluator) Evaluator {
	return &evaluator{self: specialization, logger: logger, definitions: make([]*definition, 0, 16), defaultLoader: defaultLoader}
}

func ResolveResolvables(loader DefiningLoader, logger Logger) {
	loader.ResolveResolvables(&context{NewEvaluator(loader, logger), loader, NewScope(), []Location{}})
}

func (c *context) StackPush(location Location) {
	c.stack = append(c.stack, location)
}

func (c *context) StackPop() {
	c.stack = c.stack[:len(c.stack)-1]
}

func (c *context) StackTop() Location {
	s := len(c.stack)
	if s == 0 {
		return &systemLocation{}
	}
	return c.stack[s-1]
}

func (c *context) Stack() []Location {
	return c.stack
}

func (c *context) Evaluator() Evaluator {
	return c.evaluator
}

func (c *context) ParseType(typeString PValue) PType {
	if sv, ok := typeString.(*StringValue); ok {
		return c.ParseResolve(sv.String())
	}
	panic(NewIllegalArgumentType2(`ParseType`, 0, `String`, typeString))
}

func (c *context) Resolve(expr Expression) PValue {
	return c.evaluator.Eval(expr, c)
}

func (c *context) ResolveType(expr Expression) PType {
	resolved := c.Resolve(expr)
	if pt, ok := resolved.(PType); ok {
		return pt
	}
	panic(Sprintf(`Expression "%s" does no resolve to a Type`, expr.String()))
}

func (c *context) Call(name string, args []PValue, block Lambda) PValue {
	tn := NewTypedName2(`function`, name, c.Loader().NameAuthority())
	if f, ok := Load(c.Loader(), tn); ok {
		return f.(Function).Call(c, block, args...)
	}
	panic(NewReportedIssue(EVAL_UNKNOWN_FUNCTION, SEVERITY_ERROR, H{`name`: tn.String()}, c.StackTop()))
}

func (c *context) Fail(message string) *ReportedIssue {
	return c.Error(nil, EVAL_FAILURE, H{`message`: message})
}

func (c *context) Error(location Location, issueCode IssueCode, args H) *ReportedIssue {
	if location == nil {
		location = c.StackTop()
	}
	return NewReportedIssue(issueCode, SEVERITY_ERROR, args, location)
}

func (c *context) Evaluate(expr Expression) PValue {
	return c.evaluator.Eval(expr, c)
}

func (c *context) EvaluateIn(expr Expression, scope Scope) PValue {
	return c.evaluator.Eval(expr, c.WithScope(scope))
}

func (c *context) Logger() Logger {
	return c.evaluator.Logger()
}

func (c *context) WithScope(scope Scope) EvalContext {
	return &context{c.evaluator, c.loader, scope, c.stack}
}

func (c *context) Loader() Loader {
	return c.loader
}

func (c *context) Scope() Scope {
	return c.scope
}

func (c *context) ParseAndValidate(filename, str string, singleExpression bool) Expression {
	var parserOptions []ParserOption
	if GetSetting(`tasks`, Boolean_FALSE).(*BooleanValue).Bool() {
		parserOptions = append(parserOptions, PARSER_TASKS_ENABLED)
	}
	expr, err := CreateParser(parserOptions...).Parse(filename, str, singleExpression)
	if err != nil {
		panic(err)
	}
	checker := validator.NewChecker(validator.STRICT_ERROR)
	checker.Validate(expr)
	issues := checker.Issues()
	if len(issues) > 0 {
		severity := SEVERITY_IGNORE
		for _, issue := range issues {
			c.Logger().Log(LogLevel(issue.Severity()), WrapString(issue.String()))
			if issue.Severity() > severity {
				severity = issue.Severity()
			}
		}
		if severity == SEVERITY_ERROR {
			c.Fail(Sprintf(`Error validating %s`, filename))
		}
	}
	return expr
}

func (c *context) ParseResolve(str string) PType {
	return c.ResolveType(c.ParseAndValidate(``, str, true))
}

func (e *evaluator) AddDefinitions(expr Expression) {
	if prog, ok := expr.(*Program); ok {
		loader := e.loaderForFile(prog.File())
		for _, d := range prog.Definitions() {
			e.define(loader, d)
		}
	}
}

func (e *evaluator) Evaluate(expr Expression, scope Scope, loader Loader) (result PValue, err *ReportedIssue) {
	defer func() {
		if r := recover(); r != nil {
			switch r.(type) {
			case *ReportedIssue:
				result = UNDEF
				err = r.(*ReportedIssue)
			case *StopIteration:
				result = UNDEF
				err = e.evalError(EVAL_ILLEGAL_BREAK, r.(*StopIteration).Location(), NO_ARGS)
			case *NextIteration:
				result = UNDEF
				err = e.evalError(EVAL_ILLEGAL_NEXT, r.(*NextIteration).Location(), NO_ARGS)
			case *Return:
				result = UNDEF
				err = e.evalError(EVAL_ILLEGAL_RETURN, r.(*Return).Location(), NO_ARGS)
			default:
				panic(r)
			}
		}
	}()

	err = nil
	if loader == nil {
		loader = e.loaderForFile(expr.File())
	}
	ctx := &context{e.self, loader, scope, make([]Location, 0, 64)}
	ctx.StackPush(expr)
	e.ResolveDefinitions(ctx)
	currentContext = ctx
	result = e.eval(expr, ctx)
	return
}

func (e *evaluator) Eval(expr Expression, c EvalContext) PValue {
	currentContext = c
	return e.internalEval(expr, c)
}

func (e *evaluator) Logger() Logger {
	return e.logger
}

func (e *evaluator) callFunction(name string, args []PValue, call CallExpression, c EvalContext) PValue {
	return e.call(`function`, name, args, call, c)
}

func (e *evaluator) call(funcType Namespace, name string, args []PValue, call CallExpression, c EvalContext) (result PValue) {
	if c == nil || c.Loader() == nil {
		panic(`foo`)
	}
	tn := NewTypedName2(funcType, name, c.Loader().NameAuthority())
	f, ok := Load(c.Loader(), tn)
	if !ok {
		panic(e.evalError(EVAL_UNKNOWN_FUNCTION, call, H{`name`: tn.String()}))
	}

	var block Lambda
	if call.Lambda() != nil {
		block = e.Eval(call.Lambda(), c).(Lambda)
	}

	fn := f.(Function)

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

func (e *evaluator) convertCallError(err interface{}, expr Expression, args []Expression) {
	switch err.(type) {
	case nil:
	case *ArgumentsError:
		panic(e.evalError(EVAL_ARGUMENTS_ERROR, expr, H{`expression`: expr, `message`: err.(*ArgumentsError).Error()}))
	case *IllegalArgument:
		ia := err.(*IllegalArgument)
		panic(e.evalError(EVAL_ILLEGAL_ARGUMENT, args[ia.Index()], H{`expression`: expr, `number`: ia.Index(), `message`: ia.Error()}))
	case *IllegalArgumentType:
		ia := err.(*IllegalArgumentType)
		panic(e.evalError(EVAL_ILLEGAL_ARGUMENT_TYPE, args[ia.Index()],
			H{`expression`: expr, `number`: ia.Index(), `expected`: ia.Expected(), `actual`: ia.Actual()}))
	case *IllegalArgumentCount:
		iac := err.(*IllegalArgumentCount)
		panic(e.evalError(EVAL_ILLEGAL_ARGUMENT_COUNT, expr, H{`expression`: expr, `expected`: iac.Expected(), `actual`: iac.Actual()}))
	default:
		panic(err)
	}
}

func (e *evaluator) define(loader DefiningLoader, d Definition) {
	var ta interface{}
	var tn TypedName
	switch d.(type) {
	case *PlanDefinition:
		pe := d.(*PlanDefinition)
		tn = NewTypedName2(PLAN, pe.Name(), loader.NameAuthority())
		ta = NewPuppetPlan(pe)
	case *FunctionDefinition:
		fe := d.(*FunctionDefinition)
		tn = NewTypedName2(FUNCTION, fe.Name(), loader.NameAuthority())
		ta = NewPuppetFunction(fe)
	default:
		ta, tn = CreateTypeDefinition(d, loader.NameAuthority())
	}
	loader.SetEntry(tn, NewLoaderEntry(ta, d.File()))
	e.definitions = append(e.definitions, &definition{ta, loader})
}

func (e *evaluator) eval(expr Expression, c EvalContext) PValue {
	v := e.self.Eval(expr, c)
	if iv, ok := v.(IteratorValue); ok {
		// Iterators are never returned. Convert to Array
		return iv.DynamicValue().AsArray()
	}
	return v
}

func (e *evaluator) eval_AndExpression(expr *AndExpression, c EvalContext) PValue {
	return WrapBoolean(IsTruthy(e.eval(expr.Lhs(), c)) && IsTruthy(e.eval(expr.Rhs(), c)))
}

func (e *evaluator) eval_OrExpression(expr *OrExpression, c EvalContext) PValue {
	return WrapBoolean(IsTruthy(e.eval(expr.Lhs(), c)) || IsTruthy(e.eval(expr.Rhs(), c)))
}

func (e *evaluator) eval_BlockExpression(expr *BlockExpression, c EvalContext) (result PValue) {
	result = UNDEF
	for _, statement := range expr.Statements() {
		result = e.eval(statement, c)
	}
	return result
}

func (e *evaluator) eval_ConcatenatedString(expr *ConcatenatedString, c EvalContext) PValue {
	bld := bytes.NewBufferString(``)
	for _, s := range expr.Segments() {
		bld.WriteString(e.eval(s, c).(*StringValue).String())
	}
	return WrapString(bld.String())
}

func (e *evaluator) eval_HeredocExpression(expr *HeredocExpression, c EvalContext) PValue {
	return e.eval(expr.Text(), c)
}

func (e *evaluator) eval_CallMethodExpression(call *CallMethodExpression, c EvalContext) PValue {
	fc, ok := call.Functor().(*NamedAccessExpression)
	if !ok {
		panic(e.evalError(validator.VALIDATE_ILLEGAL_EXPRESSION, call.Functor(),
			H{`expression`: call.Functor(), `feature`: `function accessor`, `container`: call}))
	}
	qn, ok := fc.Rhs().(*QualifiedName)
	if !ok {
		panic(e.evalError(validator.VALIDATE_ILLEGAL_EXPRESSION, call.Functor(),
			H{`expression`: call.Functor(), `feature`: `function name`, `container`: call}))
	}
	receiver := e.unfold([]Expression{fc.Lhs()}, c)
	obj := receiver[0]
	if tem, ok := obj.Type().(TypeWithCallableMembers); ok {
		if mbr, ok := tem.Member(qn.Name()); ok {
			var block Lambda
			if call.Lambda() != nil {
				block = e.Eval(call.Lambda(), c).(Lambda)
			}
			return mbr.Call(c, obj, block, e.unfold(call.Arguments(), c))
		}
	}
	return e.callFunction(qn.Name(), e.unfold(call.Arguments(), c, receiver...), call, c)
}

func (e *evaluator) eval_CallNamedFunctionExpression(call *CallNamedFunctionExpression, c EvalContext) PValue {
	fc := call.Functor()
	switch fc.(type) {
	case *QualifiedName:
		return e.callFunction(fc.(*QualifiedName).Name(), e.unfold(call.Arguments(), c), call, c)
	case *QualifiedReference:
		return e.callFunction(`new`, e.unfold(call.Arguments(), c, WrapString(fc.(*QualifiedReference).Name())), call, c)
	default:
		panic(e.evalError(validator.VALIDATE_ILLEGAL_EXPRESSION, call.Functor(),
			H{`expression`: call.Functor(), `feature`: `function name`, `container`: call}))
	}
}

func (e *evaluator) eval_IfExpression(expr *IfExpression, c EvalContext) PValue {
	return c.Scope().WithLocalScope(func(s Scope) PValue {
		c = c.WithScope(s)
		if IsTruthy(e.eval(expr.Test(), c)) {
			return e.eval(expr.Then(), c)
		}
		return e.eval(expr.Else(), c)
	})
}

func (e *evaluator) eval_UnlessExpression(expr *UnlessExpression, c EvalContext) PValue {
	return c.Scope().WithLocalScope(func(s Scope) PValue {
		c = c.WithScope(s)
		if !IsTruthy(e.eval(expr.Test(), c)) {
			return e.eval(expr.Then(), c)
		}
		return e.eval(expr.Else(), c)
	})
}

func (e *evaluator) eval_KeyedEntry(expr *KeyedEntry, c EvalContext) PValue {
	return WrapHashEntry(e.eval(expr.Key(), c), e.eval(expr.Value(), c))
}

func (e *evaluator) eval_LambdaExpression(expr *LambdaExpression, c EvalContext) PValue {
	return NewPuppetLambda(expr, c.(*context))
}

func (e *evaluator) eval_LiteralHash(expr *LiteralHash, c EvalContext) PValue {
	entries := expr.Entries()
	top := len(entries)
	if top == 0 {
		return EMPTY_MAP
	}
	result := make([]*HashEntry, top)
	for idx := 0; idx < top; idx++ {
		result[idx] = e.eval(entries[idx], c).(*HashEntry)
	}
	return WrapHash(result)
}

func (e *evaluator) eval_LiteralList(expr *LiteralList, c EvalContext) PValue {
	elems := expr.Elements()
	top := len(elems)
	if top == 0 {
		return EMPTY_ARRAY
	}
	result := make([]PValue, top)
	for idx := 0; idx < top; idx++ {
		result[idx] = e.eval(elems[idx], c)
	}
	return WrapArray(result)
}

func (e *evaluator) eval_LiteralBoolean(expr *LiteralBoolean) PValue {
	return WrapBoolean(expr.Bool())
}

func (e *evaluator) eval_LiteralDefault(expr *LiteralDefault) PValue {
	return WrapDefault()
}

func (e *evaluator) eval_LiteralFloat(expr *LiteralFloat) PValue {
	return WrapFloat(expr.Float())
}

func (e *evaluator) eval_LiteralInteger(expr *LiteralInteger) PValue {
	return WrapInteger(expr.Int())
}

func (e *evaluator) eval_LiteralString(expr *LiteralString) PValue {
	return WrapString(expr.StringValue())
}

func (e *evaluator) eval_NotExpression(expr *NotExpression, c EvalContext) PValue {
	return WrapBoolean(!IsTruthy(e.eval(expr.Expr(), c)))
}

func (e *evaluator) eval_ParenthesizedExpression(expr *ParenthesizedExpression, c EvalContext) PValue {
	return e.eval(expr.Expr(), c)
}

func (e *evaluator) eval_Program(expr *Program, c EvalContext) PValue {
	c.StackPush(expr)
	defer func() {
		c.StackPop()
	}()
	return e.eval(expr.Body(), c)
}

func (e *evaluator) eval_QualifiedName(expr *QualifiedName) PValue {
	return WrapString(expr.Name())
}

func (e *evaluator) eval_QualifiedReference(expr *QualifiedReference, c EvalContext) PValue {
	dcName := expr.DowncasedName()
	pt := coreTypes[dcName]
	if pt != nil {
		return pt
	}
	return e.loadType(expr.Name(), c.Loader())
}

func (e *evaluator) eval_RegexpExpression(expr *RegexpExpression) PValue {
	return WrapRegexp(expr.PatternString())
}

func (e *evaluator) eval_CaseExpression(expr *CaseExpression, c EvalContext) PValue {
	return c.Scope().WithLocalScope(func(scope Scope) PValue {
		c = c.WithScope(scope)
		test := e.eval(expr.Test(), c)
		var the_default *CaseOption
		var selected *CaseOption
	options:
		for _, o := range expr.Options() {
			co := o.(*CaseOption)
			for _, cv := range co.Values() {
				cv = unwindParenthesis(cv)
				switch cv.(type) {
				case *LiteralDefault:
					the_default = co
				case *UnfoldExpression:
					if Any2(e.eval(cv, c).(IndexedValue), func(v PValue) bool { return match(expr.Test(), cv, `match`, scope, test, v) }) {
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
			return UNDEF
		}
		return e.eval(selected.Then(), c)
	})
}

func (e *evaluator) eval_SelectorExpression(expr *SelectorExpression, c EvalContext) PValue {
	return c.Scope().WithLocalScope(func(scope Scope) PValue {
		c = c.WithScope(scope)
		test := e.eval(expr.Lhs(), c)
		var the_default *SelectorEntry
		var selected *SelectorEntry
	selectors:
		for _, s := range expr.Selectors() {
			se := s.(*SelectorEntry)
			me := unwindParenthesis(se.Matching())
			switch me.(type) {
			case *LiteralDefault:
				the_default = se
			case *UnfoldExpression:
				if Any2(e.eval(me, c).(IndexedValue), func(v PValue) bool { return match(expr.Lhs(), me, `match`, scope, test, v) }) {
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
			return UNDEF
		}
		return e.eval(selected.Value(), c)
	})
}

func (e *evaluator) eval_TextExpression(expr *TextExpression, c EvalContext) PValue {
	return WrapString(Sprintf(`%s`, e.eval(expr.Expr(), c).String()))
}

func (e *evaluator) eval_VariableExpression(expr *VariableExpression, c EvalContext) (value PValue) {
	name, ok := expr.Name()
	if ok {
		if value, ok = c.Scope().Get(name); ok {
			return value
		}
		panic(e.evalError(EVAL_UNKNOWN_VARIABLE, expr, H{`name`: name}))
	}
	idx, _ := expr.Index()
	if value, ok = c.Scope().RxGet(int(idx)); ok {
		return value
	}
	panic(e.evalError(EVAL_UNKNOWN_VARIABLE, expr, H{`name`: idx}))
}

func (e *evaluator) eval_UnfoldExpression(expr *UnfoldExpression, c EvalContext) PValue {
	candidate := e.eval(expr.Expr(), c)
	switch candidate.(type) {
	case *UndefValue:
		return WrapArray([]PValue{UNDEF})
	case *ArrayValue:
		return candidate
	case *HashValue:
		return WrapArray(candidate.(*HashValue).Elements())
	case IteratorValue:
		return candidate.(IteratorValue).DynamicValue().AsArray()
	default:
		return WrapArray([]PValue{candidate})
	}
}

func (e *evaluator) evalError(code IssueCode, location Location, args H) *ReportedIssue {
	return NewReportedIssue(code, SEVERITY_ERROR, args, location)
}

func (e *evaluator) internalEval(expr Expression, c EvalContext) PValue {
	switch expr.(type) {
	case *AccessExpression:
		return e.eval_AccessExpression(expr.(*AccessExpression), c)
	case *AndExpression:
		return e.eval_AndExpression(expr.(*AndExpression), c)
	case *ArithmeticExpression:
		return e.eval_ArithmeticExpression(expr.(*ArithmeticExpression), c)
	case *AssignmentExpression:
		return e.eval_AssignmentExpression(expr.(*AssignmentExpression), c)
	case *BlockExpression:
		return e.eval_BlockExpression(expr.(*BlockExpression), c)
	case *CallMethodExpression:
		return e.eval_CallMethodExpression(expr.(*CallMethodExpression), c)
	case *CallNamedFunctionExpression:
		return e.eval_CallNamedFunctionExpression(expr.(*CallNamedFunctionExpression), c)
	case *CaseExpression:
		return e.eval_CaseExpression(expr.(*CaseExpression), c)
	case *ComparisonExpression:
		return e.eval_ComparisonExpression(expr.(*ComparisonExpression), c)
	case *ConcatenatedString:
		return e.eval_ConcatenatedString(expr.(*ConcatenatedString), c)
	case *HeredocExpression:
		return e.eval_HeredocExpression(expr.(*HeredocExpression), c)
	case *IfExpression:
		return e.eval_IfExpression(expr.(*IfExpression), c)
	case *KeyedEntry:
		return e.eval_KeyedEntry(expr.(*KeyedEntry), c)
	case *LambdaExpression:
		return e.eval_LambdaExpression(expr.(*LambdaExpression), c)
	case *LiteralHash:
		return e.eval_LiteralHash(expr.(*LiteralHash), c)
	case *LiteralList:
		return e.eval_LiteralList(expr.(*LiteralList), c)
	case *MatchExpression:
		return e.eval_MatchExpression(expr.(*MatchExpression), c)
	case *NotExpression:
		return e.eval_NotExpression(expr.(*NotExpression), c)
	case *OrExpression:
		return e.eval_OrExpression(expr.(*OrExpression), c)
	case *ParenthesizedExpression:
		return e.eval_ParenthesizedExpression(expr.(*ParenthesizedExpression), c)
	case *Program:
		return e.eval_Program(expr.(*Program), c)
	case *QualifiedName:
		return e.eval_QualifiedName(expr.(*QualifiedName))
	case *QualifiedReference:
		return e.eval_QualifiedReference(expr.(*QualifiedReference), c)
	case *RegexpExpression:
		return e.eval_RegexpExpression(expr.(*RegexpExpression))
	case *SelectorExpression:
		return e.eval_SelectorExpression(expr.(*SelectorExpression), c)
	case *FunctionDefinition, *TypeAlias, *TypeMapping:
		// All definitions must be processed at this time
		return UNDEF
	case *LiteralBoolean:
		return e.eval_LiteralBoolean(expr.(*LiteralBoolean))
	case *LiteralDefault:
		return e.eval_LiteralDefault(expr.(*LiteralDefault))
	case *LiteralFloat:
		return e.eval_LiteralFloat(expr.(*LiteralFloat))
	case *LiteralInteger:
		return e.eval_LiteralInteger(expr.(*LiteralInteger))
	case *LiteralString:
		return e.eval_LiteralString(expr.(*LiteralString))
	case *LiteralUndef, *Nop:
		return UNDEF
	case *TextExpression:
		return e.eval_TextExpression(expr.(*TextExpression), c)
	case *UnfoldExpression:
		return e.eval_UnfoldExpression(expr.(*UnfoldExpression), c)
	case *UnlessExpression:
		return e.eval_UnlessExpression(expr.(*UnlessExpression), c)
	case *VariableExpression:
		return e.eval_VariableExpression(expr.(*VariableExpression), c)
	default:
		panic(e.evalError(EVAL_UNHANDLED_EXPRESSION, expr, H{`expression`: expr}))
	}
}

func (e *evaluator) loaderForFile(fileName string) DefiningLoader {
	if fileName == `` {
		return e.defaultLoader
	}
	return e.loaderForDir(path.Dir(fileName))
}

func (e *evaluator) loaderForDir(dirName string) DefiningLoader {
	// TODO: Proper handling of module loaders
	return e.defaultLoader
}

func (e *evaluator) loadType(name string, loader Loader) PType {
	tn := NewTypedName2(TYPE, name, loader.NameAuthority())
	found, ok := Load(loader, tn)
	if ok {
		return found.(PType)
	}
	return NewTypeReferenceType(name)
}

func (e *evaluator) ResolveDefinitions(c EvalContext) {
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
			case ResolvableType:
				d.definedValue.(ResolvableType).Resolve(tr)
			}
		}
	}
}

func (e *evaluator) unfold(array []Expression, c EvalContext, initial ...PValue) []PValue {
	result := make([]PValue, len(initial), len(initial)+len(array))
	copy(result, initial)
	for _, ex := range array {
		ex = unwindParenthesis(ex)
		if u, ok := ex.(*UnfoldExpression); ok {
			ev := e.eval(u.Expr(), c)
			switch ev.(type) {
			case *ArrayValue:
				result = append(result, ev.(*ArrayValue).Elements()...)
			default:
				result = append(result, ev)
			}
		} else {
			result = append(result, e.eval(ex, c))
		}
	}
	return result
}

func unwindParenthesis(expr Expression) Expression {
	if p, ok := expr.(*ParenthesizedExpression); ok {
		return p.Expr()
	}
	return expr
}
