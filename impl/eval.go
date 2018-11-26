package impl

import (
	"bytes"
	"fmt"
	"github.com/puppetlabs/go-parser/literal"
	"sort"

	"github.com/puppetlabs/go-evaluator/errors"
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/types"
	"github.com/puppetlabs/go-issues/issue"
	"github.com/puppetlabs/go-parser/parser"
	"github.com/puppetlabs/go-parser/validator"
)

var coreTypes = map[string]eval.Type{
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
	evaluator struct {
		logger eval.Logger
	}
	systemLocation struct{}
)

func EachCoreType(fc func(t eval.Type)) {
	keys := make([]string, len(coreTypes))
	i := 0
	for key := range coreTypes {
		keys[i] = key
		i++
	}
	sort.Strings(keys)
	for _, key := range keys {
		fc(coreTypes[key])
	}
}

func (systemLocation) File() string {
	return ``
}

func (systemLocation) Line() int {
	return 0
}

func (systemLocation) Pos() int {
	return 0
}

func init() {
	eval.TopEvaluate = topEvaluate

	eval.Error = func(issueCode issue.Code, args issue.H) issue.Reported {
		return issue.NewReported(issueCode, issue.SEVERITY_ERROR, args, eval.CurrentContext().StackTop())
	}

	eval.Error2 = func(location issue.Location, issueCode issue.Code, args issue.H) issue.Reported {
		return issue.NewReported(issueCode, issue.SEVERITY_ERROR, args, location)
	}

	eval.Warning = func(issueCode issue.Code, args issue.H) issue.Reported {
		c := eval.CurrentContext()
		ri := issue.NewReported(issueCode, issue.SEVERITY_WARNING, args, c.StackTop())
		c.Logger().LogIssue(ri)
		return ri
	}
}

func NewEvaluator(logger eval.Logger) eval.Evaluator {
	return &evaluator{logger: logger}
}

func topEvaluate(ctx eval.Context, expr parser.Expression) (result eval.Value, err issue.Reported) {
	defer func() {
		if r := recover(); r != nil {
			switch r.(type) {
			case issue.Reported:
				result = eval.UNDEF
				err = r.(issue.Reported)
			case *errors.StopIteration:
				result = eval.UNDEF
				err = evalError(eval.EVAL_ILLEGAL_BREAK, r.(*errors.StopIteration).Location(), issue.NO_ARGS)
			case *errors.NextIteration:
				result = eval.UNDEF
				err = evalError(eval.EVAL_ILLEGAL_NEXT, r.(*errors.NextIteration).Location(), issue.NO_ARGS)
			case *errors.Return:
				result = eval.UNDEF
				err = evalError(eval.EVAL_ILLEGAL_RETURN, r.(*errors.Return).Location(), issue.NO_ARGS)
			case *errors.ArgumentsError:
				err = evalError(eval.EVAL_ARGUMENTS_ERROR, expr, issue.H{`expression`: expr, `message`: r.(*errors.ArgumentsError).Error()})
			default:
				panic(r)
			}
		}
	}()

	err = nil
	ctx.StackPush(expr)
	ctx.ResolveDefinitions()
	result = ctx.Evaluator().Eval(expr, ctx)
	return
}

func (e *evaluator) Eval(expr parser.Expression, c eval.Context) eval.Value {
	return BasicEval(e, expr, c)
}

func (e *evaluator) Logger() eval.Logger {
	return e.logger
}

func callFunction(e eval.Evaluator, name string, args []eval.Value, ce parser.CallExpression, c eval.Context) eval.Value {
	return call(e, `function`, name, args, ce, c)
}

func call(e eval.Evaluator, funcType eval.Namespace, name string, args []eval.Value, call parser.CallExpression, c eval.Context) (result eval.Value) {
	tn := eval.NewTypedName2(funcType, name, c.Loader().NameAuthority())
	f, ok := eval.Load(c, tn)
	if !ok {
		panic(evalError(eval.EVAL_UNKNOWN_FUNCTION, call, issue.H{`name`: tn.String()}))
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
			convertCallError(err, call, call.Arguments())
		}
	}()
	result = fn.Call(c, block, args...)
	return
}

func convertCallError(err interface{}, expr parser.Expression, args []parser.Expression) {
	switch err.(type) {
	case nil:
	case *errors.ArgumentsError:
		panic(evalError(eval.EVAL_ARGUMENTS_ERROR, expr, issue.H{`expression`: expr, `message`: err.(*errors.ArgumentsError).Error()}))
	case *errors.IllegalArgument:
		ia := err.(*errors.IllegalArgument)
		panic(evalError(eval.EVAL_ILLEGAL_ARGUMENT, args[ia.Index()], issue.H{`expression`: expr, `number`: ia.Index(), `message`: ia.Error()}))
	case *errors.IllegalArgumentType:
		ia := err.(*errors.IllegalArgumentType)
		panic(evalError(eval.EVAL_ILLEGAL_ARGUMENT_TYPE, args[ia.Index()],
			issue.H{`expression`: expr, `number`: ia.Index(), `expected`: ia.Expected(), `actual`: ia.Actual()}))
	case *errors.IllegalArgumentCount:
		iac := err.(*errors.IllegalArgumentCount)
		panic(evalError(eval.EVAL_ILLEGAL_ARGUMENT_COUNT, expr, issue.H{`expression`: expr, `expected`: iac.Expected(), `actual`: iac.Actual()}))
	default:
		panic(err)
	}
}

func eval_AndExpression(e eval.Evaluator, expr *parser.AndExpression, c eval.Context) eval.Value {
	return types.WrapBoolean(eval.IsTruthy(e.Eval(expr.Lhs(), c)) && eval.IsTruthy(e.Eval(expr.Rhs(), c)))
}

func eval_OrExpression(e eval.Evaluator, expr *parser.OrExpression, c eval.Context) eval.Value {
	return types.WrapBoolean(eval.IsTruthy(e.Eval(expr.Lhs(), c)) || eval.IsTruthy(e.Eval(expr.Rhs(), c)))
}

func eval_Parameter(e eval.Evaluator, expr *parser.Parameter, c eval.Context) eval.Value {
	var pt eval.Type
	if expr.Type() == nil {
		pt = types.DefaultAnyType()
	} else {
		pt = c.ResolveType(expr.Type())
	}

	var value eval.Value
	if valueExpr := expr.Value(); valueExpr != nil {
		if lit, ok := literal.ToLiteral(valueExpr); ok {
			value = eval.Wrap(c, lit)
		} else {
			if cf, ok := valueExpr.(*parser.CallNamedFunctionExpression); ok {
				if qn, ok := cf.Functor().(*parser.QualifiedName); ok {
					va := make([]eval.Value, len(cf.Arguments()))
					for i, a := range cf.Arguments() {
						va[i] = e.Eval(a, c)
					}
					value = types.NewDeferred(qn.Name(), va...)
				}
			} else if ve, ok := valueExpr.(*parser.VariableExpression); ok {
				if vn, ok := ve.Name(); ok {
					value = types.NewDeferred(`$` + vn)
				}
			}
			if value == nil {
				value = types.NewDeferredExpression(valueExpr)
			}
		}
	}
	return NewParameter(expr.Name(), pt, value, expr.CapturesRest())
}

func eval_BlockExpression(e eval.Evaluator, expr *parser.BlockExpression, c eval.Context) (result eval.Value) {
	result = eval.UNDEF
	for _, statement := range expr.Statements() {
		result = e.Eval(statement, c)
	}
	return result
}

func eval_ConcatenatedString(e eval.Evaluator, expr *parser.ConcatenatedString, c eval.Context) eval.Value {
	bld := bytes.NewBufferString(``)
	for _, s := range expr.Segments() {
		bld.WriteString(e.Eval(s, c).(*types.StringValue).String())
	}
	return types.WrapString(bld.String())
}

func eval_HeredocExpression(e eval.Evaluator, expr *parser.HeredocExpression, c eval.Context) eval.Value {
	return e.Eval(expr.Text(), c)
}

func eval_CallMethodExpression(e eval.Evaluator, call *parser.CallMethodExpression, c eval.Context) eval.Value {
	fc, ok := call.Functor().(*parser.NamedAccessExpression)
	if !ok {
		panic(evalError(validator.VALIDATE_ILLEGAL_EXPRESSION, call.Functor(),
			issue.H{`expression`: call.Functor(), `feature`: `function accessor`, `container`: call}))
	}
	qn, ok := fc.Rhs().(*parser.QualifiedName)
	if !ok {
		panic(evalError(validator.VALIDATE_ILLEGAL_EXPRESSION, call.Functor(),
			issue.H{`expression`: call.Functor(), `feature`: `function name`, `container`: call}))
	}
	receiver := unfold(e, []parser.Expression{fc.Lhs()}, c)
	obj := receiver[0]
	if tem, ok := obj.PType().(eval.TypeWithCallableMembers); ok {
		if mbr, ok := tem.Member(qn.Name()); ok {
			var block eval.Lambda
			if call.Lambda() != nil {
				block = e.Eval(call.Lambda(), c).(eval.Lambda)
			}
			return mbr.Call(c, obj, block, unfold(e, call.Arguments(), c))
		}
	}
	return callFunction(e, qn.Name(), unfold(e, call.Arguments(), c, receiver...), call, c)
}

func eval_CallNamedFunctionExpression(e eval.Evaluator, call *parser.CallNamedFunctionExpression, c eval.Context) eval.Value {
	fc := call.Functor()
	switch fc.(type) {
	case *parser.QualifiedName:
		return callFunction(e, fc.(*parser.QualifiedName).Name(), unfold(e, call.Arguments(), c), call, c)
	case *parser.QualifiedReference:
		return callFunction(e, `new`, unfold(e, call.Arguments(), c, types.WrapString(fc.(*parser.QualifiedReference).Name())), call, c)
	}
	panic(evalError(validator.VALIDATE_ILLEGAL_EXPRESSION, call.Functor(),
		issue.H{`expression`: call.Functor(), `feature`: `function name`, `container`: call}))
}

func eval_IfExpression(e eval.Evaluator, expr *parser.IfExpression, c eval.Context) eval.Value {
	return c.Scope().WithLocalScope(func() eval.Value {
		if eval.IsTruthy(e.Eval(expr.Test(), c)) {
			return e.Eval(expr.Then(), c)
		}
		return e.Eval(expr.Else(), c)
	})
}

func eval_InExpression(e eval.Evaluator, expr *parser.InExpression, c eval.Context) eval.Value {
	a := e.Eval(expr.Lhs(), c)
	x := e.Eval(expr.Rhs(), c)
	switch x.(type) {
	case *types.ArrayValue:
		return types.WrapBoolean(x.(*types.ArrayValue).Any(func(b eval.Value) bool {
			return doCompare(expr, `==`, a, b, c)
		}))
	case *types.HashValue:
		return types.WrapBoolean(x.(*types.HashValue).AnyPair(func(b, v eval.Value) bool {
			return doCompare(expr, `==`, a, b, c)
		}))
	}
	return types.Boolean_FALSE
}

func eval_UnlessExpression(e eval.Evaluator, expr *parser.UnlessExpression, c eval.Context) eval.Value {
	return c.Scope().WithLocalScope(func() eval.Value {
		if !eval.IsTruthy(e.Eval(expr.Test(), c)) {
			return e.Eval(expr.Then(), c)
		}
		return e.Eval(expr.Else(), c)
	})
}

func eval_KeyedEntry(e eval.Evaluator, expr *parser.KeyedEntry, c eval.Context) eval.Value {
	return types.WrapHashEntry(e.Eval(expr.Key(), c), e.Eval(expr.Value(), c))
}

func eval_LambdaExpression(e eval.Evaluator, expr *parser.LambdaExpression, c eval.Context) eval.Value {
	return NewPuppetLambda(expr, c)
}

func eval_LiteralHash(e eval.Evaluator, expr *parser.LiteralHash, c eval.Context) eval.Value {
	entries := expr.Entries()
	top := len(entries)
	if top == 0 {
		return eval.EMPTY_MAP
	}
	result := make([]*types.HashEntry, top)
	for idx := 0; idx < top; idx++ {
		result[idx] = e.Eval(entries[idx], c).(*types.HashEntry)
	}
	return types.WrapHash(result)
}

func eval_LiteralList(e eval.Evaluator, expr *parser.LiteralList, c eval.Context) eval.Value {
	elems := expr.Elements()
	top := len(elems)
	if top == 0 {
		return eval.EMPTY_ARRAY
	}
	result := make([]eval.Value, top)
	for idx := 0; idx < top; idx++ {
		result[idx] = e.Eval(elems[idx], c)
	}
	return types.WrapValues(result)
}

func eval_LiteralBoolean(e eval.Evaluator, expr *parser.LiteralBoolean) eval.Value {
	return types.WrapBoolean(expr.Bool())
}

func eval_LiteralDefault(e eval.Evaluator, expr *parser.LiteralDefault) eval.Value {
	return types.WrapDefault()
}

func eval_LiteralFloat(e eval.Evaluator, expr *parser.LiteralFloat) eval.Value {
	return types.WrapFloat(expr.Float())
}

func eval_LiteralInteger(e eval.Evaluator, expr *parser.LiteralInteger) eval.Value {
	return types.WrapInteger(expr.Int())
}

func eval_LiteralString(e eval.Evaluator, expr *parser.LiteralString) eval.Value {
	return types.WrapString(expr.StringValue())
}

func eval_NotExpression(e eval.Evaluator, expr *parser.NotExpression, c eval.Context) eval.Value {
	return types.WrapBoolean(!eval.IsTruthy(e.Eval(expr.Expr(), c)))
}

func eval_ParenthesizedExpression(e eval.Evaluator, expr *parser.ParenthesizedExpression, c eval.Context) eval.Value {
	return e.Eval(expr.Expr(), c)
}

func eval_Program(e eval.Evaluator, expr *parser.Program, c eval.Context) eval.Value {
	c.StackPush(expr)
	defer func() {
		c.StackPop()
	}()
	return e.Eval(expr.Body(), c)
}

func eval_QualifiedName(e eval.Evaluator, expr *parser.QualifiedName) eval.Value {
	return types.WrapString(expr.Name())
}

func eval_QualifiedReference(e eval.Evaluator, expr *parser.QualifiedReference, c eval.Context) eval.Value {
	dcName := expr.DowncasedName()
	pt := coreTypes[dcName]
	if pt != nil {
		return pt
	}
	return loadType(expr.Name(), c)
}

func eval_RegexpExpression(e eval.Evaluator, expr *parser.RegexpExpression) eval.Value {
	return types.WrapRegexp(expr.PatternString())
}

func eval_CaseExpression(e eval.Evaluator, expr *parser.CaseExpression, c eval.Context) eval.Value {
	return c.Scope().WithLocalScope(func() eval.Value {
		test := e.Eval(expr.Test(), c)
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
					if eval.Any2(e.Eval(cv, c).(eval.List), func(v eval.Value) bool { return match(c, expr.Test(), cv, `match`, true, test, v) }) {
						selected = co
						break options
					}
				default:
					if match(c, expr.Test(), cv, `match`, true, test, e.Eval(cv, c)) {
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
		return e.Eval(selected.Then(), c)
	})
}

func eval_SelectorExpression(e eval.Evaluator, expr *parser.SelectorExpression, c eval.Context) eval.Value {
	return c.Scope().WithLocalScope(func() eval.Value {
		test := e.Eval(expr.Lhs(), c)
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
				if eval.Any2(e.Eval(me, c).(eval.List), func(v eval.Value) bool { return match(c, expr.Lhs(), me, `match`, true, test, v) }) {
					selected = se
					break selectors
				}
			default:
				if match(c, expr.Lhs(), me, `match`, true, test, e.Eval(me, c)) {
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
		return e.Eval(selected.Value(), c)
	})
}

func eval_TextExpression(e eval.Evaluator, expr *parser.TextExpression, c eval.Context) eval.Value {
	return types.WrapString(fmt.Sprintf(`%s`, e.Eval(expr.Expr(), c).String()))
}

func eval_VariableExpression(e eval.Evaluator, expr *parser.VariableExpression, c eval.Context) (value eval.Value) {
	name, ok := expr.Name()
	if ok {
		if value, ok = c.Scope().Get(name); ok {
			return value
		}
		panic(evalError(eval.EVAL_UNKNOWN_VARIABLE, expr, issue.H{`name`: name}))
	}
	idx, _ := expr.Index()
	if value, ok = c.Scope().RxGet(int(idx)); ok {
		return value
	}
	panic(evalError(eval.EVAL_UNKNOWN_VARIABLE, expr, issue.H{`name`: idx}))
}

func eval_UnfoldExpression(e eval.Evaluator, expr *parser.UnfoldExpression, c eval.Context) eval.Value {
	candidate := e.Eval(expr.Expr(), c)
	switch candidate.(type) {
	case *types.UndefValue:
		return types.SingletonArray(eval.UNDEF)
	case *types.ArrayValue:
		return candidate
	case *types.HashValue:
		return types.WrapArray3(candidate.(*types.HashValue))
	case eval.IteratorValue:
		return candidate.(eval.IteratorValue).AsArray()
	default:
		return types.SingletonArray(candidate)
	}
}

func evalError(code issue.Code, location issue.Location, args issue.H) issue.Reported {
	return issue.NewReported(code, issue.SEVERITY_ERROR, args, location)
}

func BasicEval(e eval.Evaluator, expr parser.Expression, c eval.Context) eval.Value {
	switch expr.(type) {
	case *parser.AccessExpression:
		return eval_AccessExpression(e, expr.(*parser.AccessExpression), c)
	case *parser.AndExpression:
		return eval_AndExpression(e, expr.(*parser.AndExpression), c)
	case *parser.ArithmeticExpression:
		return eval_ArithmeticExpression(e, expr.(*parser.ArithmeticExpression), c)
	case *parser.ComparisonExpression:
		return eval_ComparisonExpression(e, expr.(*parser.ComparisonExpression), c)
	case *parser.HeredocExpression:
		return eval_HeredocExpression(e, expr.(*parser.HeredocExpression), c)
	case *parser.InExpression:
		return eval_InExpression(e, expr.(*parser.InExpression), c)
	case *parser.KeyedEntry:
		return eval_KeyedEntry(e, expr.(*parser.KeyedEntry), c)
	case *parser.LiteralHash:
		return eval_LiteralHash(e, expr.(*parser.LiteralHash), c)
	case *parser.LiteralList:
		return eval_LiteralList(e, expr.(*parser.LiteralList), c)
	case *parser.NotExpression:
		return eval_NotExpression(e, expr.(*parser.NotExpression), c)
	case *parser.OrExpression:
		return eval_OrExpression(e, expr.(*parser.OrExpression), c)
	case *parser.QualifiedName:
		return eval_QualifiedName(e, expr.(*parser.QualifiedName))
	case *parser.QualifiedReference:
		return eval_QualifiedReference(e, expr.(*parser.QualifiedReference), c)
	case *parser.ParenthesizedExpression:
		return eval_ParenthesizedExpression(e, expr.(*parser.ParenthesizedExpression), c)
	case *parser.RegexpExpression:
		return eval_RegexpExpression(e, expr.(*parser.RegexpExpression))
	case *parser.LiteralBoolean:
		return eval_LiteralBoolean(e, expr.(*parser.LiteralBoolean))
	case *parser.LiteralDefault:
		return eval_LiteralDefault(e, expr.(*parser.LiteralDefault))
	case *parser.LiteralFloat:
		return eval_LiteralFloat(e, expr.(*parser.LiteralFloat))
	case *parser.LiteralInteger:
		return eval_LiteralInteger(e, expr.(*parser.LiteralInteger))
	case *parser.LiteralString:
		return eval_LiteralString(e, expr.(*parser.LiteralString))
	case *parser.LiteralUndef, *parser.Nop:
		return eval.UNDEF
	case *parser.TextExpression:
		return eval_TextExpression(e, expr.(*parser.TextExpression), c)
	case eval.ParserExtension:
		return expr.(eval.ParserExtension).Evaluate(e, c)
	}

	if c.Static() {
		panic(evalError(eval.EVAL_ILLEGAL_WHEN_STATIC_EXPRESSION, expr, issue.H{`expression`: expr}))
	}

	switch expr.(type) {
	case *parser.AssignmentExpression:
		return eval_AssignmentExpression(e, expr.(*parser.AssignmentExpression), c)
	case *parser.BlockExpression:
		return eval_BlockExpression(e, expr.(*parser.BlockExpression), c)
	case *parser.CallMethodExpression:
		return eval_CallMethodExpression(e, expr.(*parser.CallMethodExpression), c)
	case *parser.CallNamedFunctionExpression:
		return eval_CallNamedFunctionExpression(e, expr.(*parser.CallNamedFunctionExpression), c)
	case *parser.CaseExpression:
		return eval_CaseExpression(e, expr.(*parser.CaseExpression), c)
	case *parser.ConcatenatedString:
		return eval_ConcatenatedString(e, expr.(*parser.ConcatenatedString), c)
	case *parser.IfExpression:
		return eval_IfExpression(e, expr.(*parser.IfExpression), c)
	case *parser.LambdaExpression:
		return eval_LambdaExpression(e, expr.(*parser.LambdaExpression), c)
	case *parser.MatchExpression:
		return eval_MatchExpression(e, expr.(*parser.MatchExpression), c)
	case *parser.Parameter:
		return eval_Parameter(e, expr.(*parser.Parameter), c)
	case *parser.Program:
		return eval_Program(e, expr.(*parser.Program), c)
	case *parser.SelectorExpression:
		return eval_SelectorExpression(e, expr.(*parser.SelectorExpression), c)
	case *parser.FunctionDefinition, *parser.PlanDefinition, *parser.ActivityExpression, *parser.TypeAlias, *parser.TypeMapping:
		// All definitions must be processed at this time
		return eval.UNDEF
	case *parser.UnfoldExpression:
		return eval_UnfoldExpression(e, expr.(*parser.UnfoldExpression), c)
	case *parser.UnlessExpression:
		return eval_UnlessExpression(e, expr.(*parser.UnlessExpression), c)
	case *parser.VariableExpression:
		return eval_VariableExpression(e, expr.(*parser.VariableExpression), c)
	default:
		panic(evalError(eval.EVAL_UNHANDLED_EXPRESSION, expr, issue.H{`expression`: expr}))
	}
}

func loadType(name string, c eval.Context) eval.Type {
	tn := eval.NewTypedName2(eval.NsType, name, c.Loader().NameAuthority())
	found, ok := eval.Load(c, tn)
	if ok {
		return found.(eval.Type)
	}
	return types.NewTypeReferenceType(name)
}

func unfold(e eval.Evaluator, array []parser.Expression, c eval.Context, initial ...eval.Value) []eval.Value {
	result := make([]eval.Value, len(initial), len(initial)+len(array))
	copy(result, initial)
	for _, ex := range array {
		ex = unwindParenthesis(ex)
		if u, ok := ex.(*parser.UnfoldExpression); ok {
			ev := e.Eval(u.Expr(), c)
			switch ev.(type) {
			case *types.ArrayValue:
				result = ev.(*types.ArrayValue).AppendTo(result)
			default:
				result = append(result, ev)
			}
		} else {
			result = append(result, e.Eval(ex, c))
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
