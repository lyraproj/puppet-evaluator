package impl

import (
	"bytes"
	"fmt"
	"github.com/lyraproj/puppet-parser/literal"
	"sort"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/puppet-evaluator/errors"
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/types"
	"github.com/lyraproj/puppet-parser/parser"
	"github.com/lyraproj/puppet-parser/validator"
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
	`init`:          types.DefaultInitType(),
	`integer`:       types.DefaultIntegerType(),
	`iterable`:      types.DefaultIterableType(),
	`iterator`:      types.DefaultIteratorType(),
	`like`:          types.DefaultLikeType(),
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
		eval.Context
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

func NewEvaluator(ctx eval.Context) eval.Evaluator {
	return &evaluator{ctx}
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
	result = ctx.GetEvaluator().Eval(expr)
	return
}

func (e *evaluator) Eval(expr parser.Expression) eval.Value {
	return BasicEval(e, expr)
}

func callFunction(e eval.Evaluator, name string, args []eval.Value, ce parser.CallExpression) eval.Value {
	return call(e, `function`, name, args, ce)
}

func call(e eval.Evaluator, funcType eval.Namespace, name string, args []eval.Value, call parser.CallExpression) (result eval.Value) {
	tn := eval.NewTypedName2(funcType, name, e.Loader().NameAuthority())
	f, ok := eval.Load(e, tn)
	if !ok {
		panic(evalError(eval.EVAL_UNKNOWN_FUNCTION, call, issue.H{`name`: tn.String()}))
	}

	var blk eval.Lambda
	if call.Lambda() != nil {
		blk = e.Eval(call.Lambda()).(eval.Lambda)
	}

	fn := f.(eval.Function)

	e.StackPush(call)
	defer func() {
		e.StackPop()
		if err := recover(); err != nil {
			convertCallError(err, call, call.Arguments())
		}
	}()
	result = fn.Call(e, blk, args...)
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

func evalAndExpression(e eval.Evaluator, expr *parser.AndExpression) eval.Value {
	return types.WrapBoolean(eval.IsTruthy(e.Eval(expr.Lhs())) && eval.IsTruthy(e.Eval(expr.Rhs())))
}

func evalOrExpression(e eval.Evaluator, expr *parser.OrExpression) eval.Value {
	return types.WrapBoolean(eval.IsTruthy(e.Eval(expr.Lhs())) || eval.IsTruthy(e.Eval(expr.Rhs())))
}

func evalParameter(e eval.Evaluator, expr *parser.Parameter) eval.Value {
	var pt eval.Type
	if expr.Type() == nil {
		pt = types.DefaultAnyType()
	} else {
		pt = e.ResolveType(expr.Type())
	}

	var value eval.Value
	if valueExpr := expr.Value(); valueExpr != nil {
		if lit, ok := literal.ToLiteral(valueExpr); ok {
			value = eval.Wrap(e, lit)
		} else {
			var cf *parser.CallNamedFunctionExpression
			if cf, ok = valueExpr.(*parser.CallNamedFunctionExpression); ok {
				var qn *parser.QualifiedName
				if qn, ok = cf.Functor().(*parser.QualifiedName); ok {
					va := make([]eval.Value, len(cf.Arguments()))
					for i, a := range cf.Arguments() {
						va[i] = e.Eval(a)
					}
					value = types.NewDeferred(qn.Name(), va...)
				}
			} else {
				var ve *parser.VariableExpression
				if ve, ok = valueExpr.(*parser.VariableExpression); ok {
					var vn string
					if vn, ok = ve.Name(); ok {
						value = types.NewDeferred(`$` + vn)
					}
				}
			}
			if value == nil {
				value = types.NewDeferredExpression(valueExpr)
			}
		}
	}
	return NewParameter(expr.Name(), pt, value, expr.CapturesRest())
}

func evalBlockExpression(e eval.Evaluator, expr *parser.BlockExpression) (result eval.Value) {
	result = eval.UNDEF
	for _, statement := range expr.Statements() {
		result = e.Eval(statement)
	}
	return result
}

func evalConcatenatedString(e eval.Evaluator, expr *parser.ConcatenatedString) eval.Value {
	bld := bytes.NewBufferString(``)
	for _, s := range expr.Segments() {
		bld.WriteString(e.Eval(s).String())
	}
	return types.WrapString(bld.String())
}

func evalHeredocExpression(e eval.Evaluator, expr *parser.HeredocExpression) eval.Value {
	return e.Eval(expr.Text())
}

func evalCallMethodExpression(e eval.Evaluator, call *parser.CallMethodExpression) eval.Value {
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
	receiver := unfold(e, []parser.Expression{fc.Lhs()})
	obj := receiver[0]
	var tem eval.TypeWithCallableMembers
	if tem, ok = obj.PType().(eval.TypeWithCallableMembers); ok {
		var mbr eval.CallableMember
		if mbr, ok = tem.Member(qn.Name()); ok {
			var b eval.Lambda
			if call.Lambda() != nil {
				b = e.Eval(call.Lambda()).(eval.Lambda)
			}
			return mbr.Call(e, obj, b, unfold(e, call.Arguments()))
		}
	}
	return callFunction(e, qn.Name(), unfold(e, call.Arguments(), receiver...), call)
}

func evalCallNamedFunctionExpression(e eval.Evaluator, call *parser.CallNamedFunctionExpression) eval.Value {
	fc := call.Functor()
	switch fc.(type) {
	case *parser.QualifiedName:
		return callFunction(e, fc.(*parser.QualifiedName).Name(), unfold(e, call.Arguments()), call)
	case *parser.QualifiedReference:
		return callFunction(e, `new`, unfold(e, call.Arguments(), types.WrapString(fc.(*parser.QualifiedReference).Name())), call)
	case *parser.AccessExpression:
		receiver := unfold(e, []parser.Expression{fc})
		return callFunction(e, `new`, unfold(e, call.Arguments(), receiver...), call)
	}
	panic(evalError(validator.VALIDATE_ILLEGAL_EXPRESSION, call.Functor(),
		issue.H{`expression`: call.Functor(), `feature`: `function name`, `container`: call}))
}

func evalIfExpression(e eval.Evaluator, expr *parser.IfExpression) eval.Value {
	return e.Scope().WithLocalScope(func() eval.Value {
		if eval.IsTruthy(e.Eval(expr.Test())) {
			return e.Eval(expr.Then())
		}
		return e.Eval(expr.Else())
	})
}

func evalInExpression(e eval.Evaluator, expr *parser.InExpression) eval.Value {
	a := e.Eval(expr.Lhs())
	x := e.Eval(expr.Rhs())
	switch x.(type) {
	case *types.ArrayValue:
		return types.WrapBoolean(x.(*types.ArrayValue).Any(func(b eval.Value) bool {
			return doCompare(expr, `==`, a, b)
		}))
	case *types.HashValue:
		return types.WrapBoolean(x.(*types.HashValue).AnyPair(func(b, v eval.Value) bool {
			return doCompare(expr, `==`, a, b)
		}))
	}
	return types.Boolean_FALSE
}

func evalUnlessExpression(e eval.Evaluator, expr *parser.UnlessExpression) eval.Value {
	return e.Scope().WithLocalScope(func() eval.Value {
		if !eval.IsTruthy(e.Eval(expr.Test())) {
			return e.Eval(expr.Then())
		}
		return e.Eval(expr.Else())
	})
}

func evalKeyedEntry(e eval.Evaluator, expr *parser.KeyedEntry) eval.Value {
	return types.WrapHashEntry(e.Eval(expr.Key()), e.Eval(expr.Value()))
}

func evalLambdaExpression(e eval.Evaluator, expr *parser.LambdaExpression) eval.Value {
	return NewPuppetLambda(expr, e)
}

func evalLiteralHash(e eval.Evaluator, expr *parser.LiteralHash) eval.Value {
	entries := expr.Entries()
	top := len(entries)
	if top == 0 {
		return eval.EMPTY_MAP
	}
	result := make([]*types.HashEntry, top)
	for idx := 0; idx < top; idx++ {
		result[idx] = e.Eval(entries[idx]).(*types.HashEntry)
	}
	return types.WrapHash(result)
}

func evalLiteralList(e eval.Evaluator, expr *parser.LiteralList) eval.Value {
	elems := expr.Elements()
	top := len(elems)
	if top == 0 {
		return eval.EMPTY_ARRAY
	}
	result := make([]eval.Value, top)
	for idx := 0; idx < top; idx++ {
		result[idx] = e.Eval(elems[idx])
	}
	return types.WrapValues(result)
}

func evalLiteralBoolean(expr *parser.LiteralBoolean) eval.Value {
	return types.WrapBoolean(expr.Bool())
}

func evalLiteralDefault() eval.Value {
	return types.WrapDefault()
}

func evalLiteralFloat(expr *parser.LiteralFloat) eval.Value {
	return types.WrapFloat(expr.Float())
}

func evalLiteralInteger(expr *parser.LiteralInteger) eval.Value {
	return types.WrapInteger(expr.Int())
}

func evalLiteralString(expr *parser.LiteralString) eval.Value {
	return types.WrapString(expr.StringValue())
}

func evalNotExpression(e eval.Evaluator, expr *parser.NotExpression) eval.Value {
	return types.WrapBoolean(!eval.IsTruthy(e.Eval(expr.Expr())))
}

func evalParenthesizedExpression(e eval.Evaluator, expr *parser.ParenthesizedExpression) eval.Value {
	return e.Eval(expr.Expr())
}

func evalProgram(e eval.Evaluator, expr *parser.Program) eval.Value {
	e.StackPush(expr)
	defer func() {
		e.StackPop()
	}()
	return e.Eval(expr.Body())
}

func evalQualifiedName(expr *parser.QualifiedName) eval.Value {
	return types.WrapString(expr.Name())
}

func evalQualifiedReference(e eval.Evaluator, expr *parser.QualifiedReference) eval.Value {
	dcName := expr.DowncasedName()
	pt := coreTypes[dcName]
	if pt != nil {
		return pt
	}
	return loadType(expr.Name(), e)
}

func evalRegexpExpression(expr *parser.RegexpExpression) eval.Value {
	return types.WrapRegexp(expr.PatternString())
}

func evalCaseExpression(e eval.Evaluator, expr *parser.CaseExpression) eval.Value {
	return e.Scope().WithLocalScope(func() eval.Value {
		test := e.Eval(expr.Test())
		var theDefault *parser.CaseOption
		var selected *parser.CaseOption
	options:
		for _, o := range expr.Options() {
			co := o.(*parser.CaseOption)
			for _, cv := range co.Values() {
				cv = unwindParenthesis(cv)
				switch cv.(type) {
				case *parser.LiteralDefault:
					theDefault = co
				case *parser.UnfoldExpression:
					if eval.Any2(e.Eval(cv).(eval.List), func(v eval.Value) bool { return match(e, expr.Test(), cv, `match`, true, test, v) }) {
						selected = co
						break options
					}
				default:
					if match(e, expr.Test(), cv, `match`, true, test, e.Eval(cv)) {
						selected = co
						break options
					}
				}
			}
		}
		if selected == nil {
			selected = theDefault
		}
		if selected == nil {
			return eval.UNDEF
		}
		return e.Eval(selected.Then())
	})
}

func evalSelectorExpression(e eval.Evaluator, expr *parser.SelectorExpression) eval.Value {
	return e.Scope().WithLocalScope(func() eval.Value {
		test := e.Eval(expr.Lhs())
		var theDefault *parser.SelectorEntry
		var selected *parser.SelectorEntry
	selectors:
		for _, s := range expr.Selectors() {
			se := s.(*parser.SelectorEntry)
			me := unwindParenthesis(se.Matching())
			switch me.(type) {
			case *parser.LiteralDefault:
				theDefault = se
			case *parser.UnfoldExpression:
				if eval.Any2(e.Eval(me).(eval.List), func(v eval.Value) bool { return match(e, expr.Lhs(), me, `match`, true, test, v) }) {
					selected = se
					break selectors
				}
			default:
				if match(e, expr.Lhs(), me, `match`, true, test, e.Eval(me)) {
					selected = se
					break selectors
				}
			}
		}
		if selected == nil {
			selected = theDefault
		}
		if selected == nil {
			return eval.UNDEF
		}
		return e.Eval(selected.Value())
	})
}

func evalTextExpression(e eval.Evaluator, expr *parser.TextExpression) eval.Value {
	return types.WrapString(fmt.Sprintf(`%s`, e.Eval(expr.Expr()).String()))
}

func evalVariableExpression(e eval.Evaluator, expr *parser.VariableExpression) (value eval.Value) {
	name, ok := expr.Name()
	if ok {
		if value, ok = e.Scope().Get(name); ok {
			return value
		}
		panic(evalError(eval.EVAL_UNKNOWN_VARIABLE, expr, issue.H{`name`: name}))
	}
	idx, _ := expr.Index()
	if value, ok = e.Scope().RxGet(int(idx)); ok {
		return value
	}
	panic(evalError(eval.EVAL_UNKNOWN_VARIABLE, expr, issue.H{`name`: idx}))
}

func evalUnfoldExpression(e eval.Evaluator, expr *parser.UnfoldExpression) eval.Value {
	candidate := e.Eval(expr.Expr())
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

// BasicEval is exported to enable the evaluator to be extended
func BasicEval(e eval.Evaluator, expr parser.Expression) eval.Value {
	switch expr.(type) {
	case *parser.AccessExpression:
		return evalAccessExpression(e, expr.(*parser.AccessExpression))
	case *parser.AndExpression:
		return evalAndExpression(e, expr.(*parser.AndExpression))
	case *parser.ArithmeticExpression:
		return evalArithmeticExpression(e, expr.(*parser.ArithmeticExpression))
	case *parser.ComparisonExpression:
		return evalComparisonExpression(e, expr.(*parser.ComparisonExpression))
	case *parser.HeredocExpression:
		return evalHeredocExpression(e, expr.(*parser.HeredocExpression))
	case *parser.InExpression:
		return evalInExpression(e, expr.(*parser.InExpression))
	case *parser.KeyedEntry:
		return evalKeyedEntry(e, expr.(*parser.KeyedEntry))
	case *parser.LiteralHash:
		return evalLiteralHash(e, expr.(*parser.LiteralHash))
	case *parser.LiteralList:
		return evalLiteralList(e, expr.(*parser.LiteralList))
	case *parser.NotExpression:
		return evalNotExpression(e, expr.(*parser.NotExpression))
	case *parser.OrExpression:
		return evalOrExpression(e, expr.(*parser.OrExpression))
	case *parser.QualifiedName:
		return evalQualifiedName(expr.(*parser.QualifiedName))
	case *parser.QualifiedReference:
		return evalQualifiedReference(e, expr.(*parser.QualifiedReference))
	case *parser.ParenthesizedExpression:
		return evalParenthesizedExpression(e, expr.(*parser.ParenthesizedExpression))
	case *parser.RegexpExpression:
		return evalRegexpExpression(expr.(*parser.RegexpExpression))
	case *parser.LiteralBoolean:
		return evalLiteralBoolean(expr.(*parser.LiteralBoolean))
	case *parser.LiteralDefault:
		return evalLiteralDefault()
	case *parser.LiteralFloat:
		return evalLiteralFloat(expr.(*parser.LiteralFloat))
	case *parser.LiteralInteger:
		return evalLiteralInteger(expr.(*parser.LiteralInteger))
	case *parser.LiteralString:
		return evalLiteralString(expr.(*parser.LiteralString))
	case *parser.LiteralUndef, *parser.Nop:
		return eval.UNDEF
	case *parser.TextExpression:
		return evalTextExpression(e, expr.(*parser.TextExpression))
	case eval.ParserExtension:
		return expr.(eval.ParserExtension).Evaluate(e)
	}

	if e.Static() {
		panic(evalError(eval.EVAL_ILLEGAL_WHEN_STATIC_EXPRESSION, expr, issue.H{`expression`: expr}))
	}

	switch expr.(type) {
	case *parser.AssignmentExpression:
		return evalAssignmentExpression(e, expr.(*parser.AssignmentExpression))
	case *parser.BlockExpression:
		return evalBlockExpression(e, expr.(*parser.BlockExpression))
	case *parser.CallMethodExpression:
		return evalCallMethodExpression(e, expr.(*parser.CallMethodExpression))
	case *parser.CallNamedFunctionExpression:
		return evalCallNamedFunctionExpression(e, expr.(*parser.CallNamedFunctionExpression))
	case *parser.CaseExpression:
		return evalCaseExpression(e, expr.(*parser.CaseExpression))
	case *parser.ConcatenatedString:
		return evalConcatenatedString(e, expr.(*parser.ConcatenatedString))
	case *parser.IfExpression:
		return evalIfExpression(e, expr.(*parser.IfExpression))
	case *parser.LambdaExpression:
		return evalLambdaExpression(e, expr.(*parser.LambdaExpression))
	case *parser.MatchExpression:
		return evalMatchExpression(e, expr.(*parser.MatchExpression))
	case *parser.Parameter:
		return evalParameter(e, expr.(*parser.Parameter))
	case *parser.Program:
		return evalProgram(e, expr.(*parser.Program))
	case *parser.SelectorExpression:
		return evalSelectorExpression(e, expr.(*parser.SelectorExpression))
	case *parser.FunctionDefinition, *parser.PlanDefinition, *parser.ActivityExpression, *parser.TypeAlias, *parser.TypeMapping:
		// All definitions must be processed at this time
		return eval.UNDEF
	case *parser.UnfoldExpression:
		return evalUnfoldExpression(e, expr.(*parser.UnfoldExpression))
	case *parser.UnlessExpression:
		return evalUnlessExpression(e, expr.(*parser.UnlessExpression))
	case *parser.VariableExpression:
		return evalVariableExpression(e, expr.(*parser.VariableExpression))
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

func unfold(e eval.Evaluator, array []parser.Expression, initial ...eval.Value) []eval.Value {
	result := make([]eval.Value, len(initial), len(initial)+len(array))
	copy(result, initial)
	for _, ex := range array {
		ex = unwindParenthesis(ex)
		if u, ok := ex.(*parser.UnfoldExpression); ok {
			ev := e.Eval(u.Expr())
			switch ev.(type) {
			case *types.ArrayValue:
				result = ev.(*types.ArrayValue).AppendTo(result)
			default:
				result = append(result, ev)
			}
		} else {
			result = append(result, e.Eval(ex))
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
