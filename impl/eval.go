package impl

import (
	"bytes"
	"fmt"

	"github.com/puppetlabs/go-evaluator/errors"
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/types"
	"github.com/puppetlabs/go-issues/issue"
	"github.com/puppetlabs/go-parser/parser"
	"github.com/puppetlabs/go-parser/validator"
	"strings"
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
	evaluator struct {
		self   eval.Evaluator
		logger eval.Logger
	}

	systemLocation struct{}
)

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
	eval.Error = func(c eval.Context, issueCode issue.Code, args issue.H) issue.Reported {
		var location issue.Location
		if c == nil {
			location = nil
		} else {
			location = c.StackTop()
		}
		return issue.NewReported(issueCode, issue.SEVERITY_ERROR, args, location)
	}

	eval.Error2 = func(location issue.Location, issueCode issue.Code, args issue.H) issue.Reported {
		return issue.NewReported(issueCode, issue.SEVERITY_ERROR, args, location)
	}

	eval.Warning = func(c eval.Context, issueCode issue.Code, args issue.H) issue.Reported {
		var location issue.Location
		var logger eval.Logger
		if c == nil {
			location = nil
			logger = eval.NewStdLogger()
		} else {
			location = c.StackTop()
			logger = c.Logger()
		}
		ri := issue.NewReported(issueCode, issue.SEVERITY_WARNING, args, location)
		logger.LogIssue(ri)
		return ri
	}
}

func NewEvaluator(logger eval.Logger) eval.Evaluator {
	e := &evaluator{logger: logger}
	e.self = e
	return e
}

func NewOverriddenEvaluator(logger eval.Logger, specialization eval.Evaluator) eval.Evaluator {
	return &evaluator{self: specialization, logger: logger}
}

func (e *evaluator) Evaluate(ctx eval.Context, expr parser.Expression) (result eval.PValue, err issue.Reported) {
	defer func() {
		if r := recover(); r != nil {
			switch r.(type) {
			case issue.Reported:
				result = eval.UNDEF
				err = r.(issue.Reported)
			case *errors.StopIteration:
				result = eval.UNDEF
				err = e.evalError(eval.EVAL_ILLEGAL_BREAK, r.(*errors.StopIteration).Location(), issue.NO_ARGS)
			case *errors.NextIteration:
				result = eval.UNDEF
				err = e.evalError(eval.EVAL_ILLEGAL_NEXT, r.(*errors.NextIteration).Location(), issue.NO_ARGS)
			case *errors.Return:
				result = eval.UNDEF
				err = e.evalError(eval.EVAL_ILLEGAL_RETURN, r.(*errors.Return).Location(), issue.NO_ARGS)
			case *errors.ArgumentsError:
				err = e.evalError(eval.EVAL_ARGUMENTS_ERROR, expr, issue.H{`expression`: expr, `message`: r.(*errors.ArgumentsError).Error()})
			default:
				panic(r)
			}
		}
	}()

	err = nil
	ctx.StackPush(expr)
	ctx.ResolveDefinitions()
	result = e.eval(expr, ctx)
	return
}

func (e *evaluator) Eval(expr parser.Expression, c eval.Context) eval.PValue {
	return e.internalEval(expr, c)
}

func (e *evaluator) Logger() eval.Logger {
	return e.logger
}

func (e *evaluator) CallFunction(name string, args []eval.PValue, call parser.CallExpression, c eval.Context) eval.PValue {
	return e.call(`function`, name, args, call, c)
}

func (e *evaluator) call(funcType eval.Namespace, name string, args []eval.PValue, call parser.CallExpression, c eval.Context) (result eval.PValue) {
	if c == nil || c.Loader() == nil {
		panic(`foo`)
	}
	tn := eval.NewTypedName2(funcType, name, c.Loader().NameAuthority())
	f, ok := eval.Load(c, tn)
	if !ok {
		panic(e.evalError(eval.EVAL_UNKNOWN_FUNCTION, call, issue.H{`name`: tn.String()}))
	}

	var block eval.Lambda
	if call.Lambda() != nil {
		block = e.eval(call.Lambda(), c).(eval.Lambda)
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

func (e *evaluator) eval(expr parser.Expression, c eval.Context) eval.PValue {
	return e.self.Eval(expr, c)
}

func (e *evaluator) eval_AndExpression(expr *parser.AndExpression, c eval.Context) eval.PValue {
	return types.WrapBoolean(eval.IsTruthy(e.eval(expr.Lhs(), c)) && eval.IsTruthy(e.eval(expr.Rhs(), c)))
}

func (e *evaluator) eval_OrExpression(expr *parser.OrExpression, c eval.Context) eval.PValue {
	return types.WrapBoolean(eval.IsTruthy(e.eval(expr.Lhs(), c)) || eval.IsTruthy(e.eval(expr.Rhs(), c)))
}

func (e *evaluator) eval_BlockExpression(expr *parser.BlockExpression, c eval.Context) (result eval.PValue) {
	result = eval.UNDEF
	for _, statement := range expr.Statements() {
		result = e.eval(statement, c)
	}
	return result
}

func (e *evaluator) eval_ConcatenatedString(expr *parser.ConcatenatedString, c eval.Context) eval.PValue {
	bld := bytes.NewBufferString(``)
	for _, s := range expr.Segments() {
		bld.WriteString(e.eval(s, c).(*types.StringValue).String())
	}
	return types.WrapString(bld.String())
}

func (e *evaluator) eval_HeredocExpression(expr *parser.HeredocExpression, c eval.Context) eval.PValue {
	return e.eval(expr.Text(), c)
}

func (e *evaluator) eval_CallMethodExpression(call *parser.CallMethodExpression, c eval.Context) eval.PValue {
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
				block = e.eval(call.Lambda(), c).(eval.Lambda)
			}
			return mbr.Call(c, obj, block, e.unfold(call.Arguments(), c))
		}
	}
	return e.self.CallFunction(qn.Name(), e.unfold(call.Arguments(), c, receiver...), call, c)
}

func (e *evaluator) eval_CallNamedFunctionExpression(call *parser.CallNamedFunctionExpression, c eval.Context) eval.PValue {
	fc := call.Functor()
	switch fc.(type) {
	case *parser.QualifiedName:
		return e.self.CallFunction(fc.(*parser.QualifiedName).Name(), e.unfold(call.Arguments(), c), call, c)
	case *parser.QualifiedReference:
		return e.self.CallFunction(`new`, e.unfold(call.Arguments(), c, types.WrapString(fc.(*parser.QualifiedReference).Name())), call, c)
	default:
		panic(e.evalError(validator.VALIDATE_ILLEGAL_EXPRESSION, call.Functor(),
			issue.H{`expression`: call.Functor(), `feature`: `function name`, `container`: call}))
	}
}

func (e *evaluator) eval_IfExpression(expr *parser.IfExpression, c eval.Context) eval.PValue {
	return c.Scope().WithLocalScope(func() eval.PValue {
		if eval.IsTruthy(e.eval(expr.Test(), c)) {
			return e.eval(expr.Then(), c)
		}
		return e.eval(expr.Else(), c)
	})
}

func (e *evaluator) eval_InExpression(expr *parser.InExpression, c eval.Context) eval.PValue {
	a := e.eval(expr.Lhs(), c)
	x := e.eval(expr.Rhs(), c)
	switch x.(type) {
	case *types.ArrayValue:
		return types.WrapBoolean(x.(*types.ArrayValue).Any(func(b eval.PValue) bool {
			return e.doCompare(expr, `==`, a, b, c)
		}))
	case *types.HashValue:
		return types.WrapBoolean(x.(*types.HashValue).AnyPair(func(b, v eval.PValue) bool {
			return e.doCompare(expr, `==`, a, b, c)
		}))
	case *types.StringValue:
		if c.Language() != eval.LangPuppet {
			if _, ok := a.(*types.StringValue); ok {
				return types.WrapBoolean(strings.Contains(strings.ToLower(x.String()), strings.ToLower(a.String())))
			}
		}
	}
	return types.Boolean_FALSE
}

func (e *evaluator) eval_UnlessExpression(expr *parser.UnlessExpression, c eval.Context) eval.PValue {
	return c.Scope().WithLocalScope(func() eval.PValue {
		if !eval.IsTruthy(e.eval(expr.Test(), c)) {
			return e.eval(expr.Then(), c)
		}
		return e.eval(expr.Else(), c)
	})
}

func (e *evaluator) eval_KeyedEntry(expr *parser.KeyedEntry, c eval.Context) eval.PValue {
	return types.WrapHashEntry(e.eval(expr.Key(), c), e.eval(expr.Value(), c))
}

func (e *evaluator) eval_LambdaExpression(expr *parser.LambdaExpression, c eval.Context) eval.PValue {
	return NewPuppetLambda(expr, c)
}

func (e *evaluator) eval_LiteralHash(expr *parser.LiteralHash, c eval.Context) eval.PValue {
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

func (e *evaluator) eval_LiteralList(expr *parser.LiteralList, c eval.Context) eval.PValue {
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

func (e *evaluator) eval_NotExpression(expr *parser.NotExpression, c eval.Context) eval.PValue {
	return types.WrapBoolean(!eval.IsTruthy(e.eval(expr.Expr(), c)))
}

func (e *evaluator) eval_ParenthesizedExpression(expr *parser.ParenthesizedExpression, c eval.Context) eval.PValue {
	return e.eval(expr.Expr(), c)
}

func (e *evaluator) eval_Program(expr *parser.Program, c eval.Context) eval.PValue {
	c.StackPush(expr)
	defer func() {
		c.StackPop()
	}()
	return e.eval(expr.Body(), c)
}

func (e *evaluator) eval_QualifiedName(expr *parser.QualifiedName) eval.PValue {
	return types.WrapString(expr.Name())
}

func (e *evaluator) eval_QualifiedReference(expr *parser.QualifiedReference, c eval.Context) eval.PValue {
	dcName := expr.DowncasedName()
	pt := coreTypes[dcName]
	if pt != nil {
		return pt
	}
	return e.loadType(expr.Name(), c)
}

func (e *evaluator) eval_RegexpExpression(expr *parser.RegexpExpression) eval.PValue {
	return types.WrapRegexp(expr.PatternString())
}

func (e *evaluator) eval_CaseExpression(expr *parser.CaseExpression, c eval.Context) eval.PValue {
	return c.Scope().WithLocalScope(func() eval.PValue {
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
					if eval.Any2(e.eval(cv, c).(eval.IndexedValue), func(v eval.PValue) bool { return match(c, expr.Test(), cv, `match`, true, test, v) }) {
						selected = co
						break options
					}
				default:
					if match(c, expr.Test(), cv, `match`, true, test, e.eval(cv, c)) {
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

func (e *evaluator) eval_SelectorExpression(expr *parser.SelectorExpression, c eval.Context) eval.PValue {
	return c.Scope().WithLocalScope(func() eval.PValue {
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
				if eval.Any2(e.eval(me, c).(eval.IndexedValue), func(v eval.PValue) bool { return match(c, expr.Lhs(), me, `match`, true, test, v) }) {
					selected = se
					break selectors
				}
			default:
				if match(c, expr.Lhs(), me, `match`, true, test, e.eval(me, c)) {
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

func (e *evaluator) eval_TextExpression(expr *parser.TextExpression, c eval.Context) eval.PValue {
	return types.WrapString(fmt.Sprintf(`%s`, e.eval(expr.Expr(), c).String()))
}

func (e *evaluator) eval_VariableExpression(expr *parser.VariableExpression, c eval.Context) (value eval.PValue) {
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

func (e *evaluator) eval_UnfoldExpression(expr *parser.UnfoldExpression, c eval.Context) eval.PValue {
	candidate := e.eval(expr.Expr(), c)
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

func (e *evaluator) evalError(code issue.Code, location issue.Location, args issue.H) issue.Reported {
	return issue.NewReported(code, issue.SEVERITY_ERROR, args, location)
}

func (e *evaluator) internalEval(expr parser.Expression, c eval.Context) eval.PValue {
	switch expr.(type) {
	case *parser.AccessExpression:
		return e.eval_AccessExpression(expr.(*parser.AccessExpression), c)
	case *parser.AndExpression:
		return e.eval_AndExpression(expr.(*parser.AndExpression), c)
	case *parser.ArithmeticExpression:
		return e.eval_ArithmeticExpression(expr.(*parser.ArithmeticExpression), c)
	case *parser.ComparisonExpression:
		return e.eval_ComparisonExpression(expr.(*parser.ComparisonExpression), c)
	case *parser.HeredocExpression:
		return e.eval_HeredocExpression(expr.(*parser.HeredocExpression), c)
	case *parser.InExpression:
		return e.eval_InExpression(expr.(*parser.InExpression), c)
	case *parser.KeyedEntry:
		return e.eval_KeyedEntry(expr.(*parser.KeyedEntry), c)
	case *parser.LiteralHash:
		return e.eval_LiteralHash(expr.(*parser.LiteralHash), c)
	case *parser.LiteralList:
		return e.eval_LiteralList(expr.(*parser.LiteralList), c)
	case *parser.NotExpression:
		return e.eval_NotExpression(expr.(*parser.NotExpression), c)
	case *parser.OrExpression:
		return e.eval_OrExpression(expr.(*parser.OrExpression), c)
	case *parser.QualifiedName:
		return e.eval_QualifiedName(expr.(*parser.QualifiedName))
	case *parser.QualifiedReference:
		return e.eval_QualifiedReference(expr.(*parser.QualifiedReference), c)
	case *parser.ParenthesizedExpression:
		return e.eval_ParenthesizedExpression(expr.(*parser.ParenthesizedExpression), c)
	case *parser.RegexpExpression:
		return e.eval_RegexpExpression(expr.(*parser.RegexpExpression))
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
	case eval.ParserExtension:
		return expr.(eval.ParserExtension).Evaluate(e, c)
	}

	if c.Static() {
		panic(e.evalError(eval.EVAL_ILLEGAL_WHEN_STATIC_EXPRESSION, expr, issue.H{`expression`: expr}))
	}

	switch expr.(type) {
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
	case *parser.ConcatenatedString:
		return e.eval_ConcatenatedString(expr.(*parser.ConcatenatedString), c)
	case *parser.IfExpression:
		return e.eval_IfExpression(expr.(*parser.IfExpression), c)
	case *parser.LambdaExpression:
		return e.eval_LambdaExpression(expr.(*parser.LambdaExpression), c)
	case *parser.MatchExpression:
		return e.eval_MatchExpression(expr.(*parser.MatchExpression), c)
	case *parser.Program:
		return e.eval_Program(expr.(*parser.Program), c)
	case *parser.SelectorExpression:
		return e.eval_SelectorExpression(expr.(*parser.SelectorExpression), c)
	case *parser.FunctionDefinition, *parser.TypeAlias, *parser.TypeMapping:
		// All definitions must be processed at this time
		return eval.UNDEF
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

func (e *evaluator) loadType(name string, c eval.Context) eval.PType {
	tn := eval.NewTypedName2(eval.TYPE, name, c.Loader().NameAuthority())
	found, ok := eval.Load(c, tn)
	if ok {
		return found.(eval.PType)
	}
	return types.NewTypeReferenceType(name)
}

func (e *evaluator) unfold(array []parser.Expression, c eval.Context, initial ...eval.PValue) []eval.PValue {
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
