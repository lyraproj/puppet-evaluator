package evaluator

import (
	"bytes"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
	"github.com/lyraproj/puppet-evaluator/errors"
	"github.com/lyraproj/puppet-evaluator/pdsl"
	"github.com/lyraproj/puppet-parser/literal"
	"github.com/lyraproj/puppet-parser/parser"
	"github.com/lyraproj/puppet-parser/validator"
)

type evaluator struct {
	pdsl.EvaluationContext
}

func init() {
	pdsl.TopEvaluate = topEvaluate
}

func NewEvaluator(ctx pdsl.EvaluationContext) pdsl.Evaluator {
	return &evaluator{ctx}
}

func topEvaluate(ctx pdsl.EvaluationContext, expr parser.Expression) px.Value {
	defer func() {
		if r := recover(); r != nil {
			switch r := r.(type) {
			case *errors.StopIteration:
				panic(evalError(pdsl.IllegalBreak, r.Location(), issue.NoArgs))
			case *errors.NextIteration:
				panic(evalError(pdsl.IllegalNext, r.Location(), issue.NoArgs))
			case *errors.Return:
				panic(evalError(pdsl.IllegalReturn, r.Location(), issue.NoArgs))
			default:
				panic(r)
			}
		}
	}()

	ctx.StackPush(expr)
	ctx.ResolveDefinitions()
	result := ctx.GetEvaluator().Eval(expr)
	ctx.StackPop()
	return result
}

func (e *evaluator) Eval(expr parser.Expression) px.Value {
	return BasicEval(e, expr)
}

func callFunction(e pdsl.Evaluator, name string, args []px.Value, ce parser.CallExpression) px.Value {
	return call(e, `function`, name, args, ce)
}

func call(e pdsl.Evaluator, funcType px.Namespace, name string, args []px.Value, call parser.CallExpression) (result px.Value) {
	tn := px.NewTypedName2(funcType, name, e.Loader().NameAuthority())
	f, ok := px.Load(e, tn)
	if !ok {
		panic(evalError(px.UnknownFunction, call, issue.H{`name`: tn.String()}))
	}

	var blk px.Lambda
	if call.Lambda() != nil {
		blk = e.Eval(call.Lambda()).(px.Lambda)
	}

	fn := f.(px.Function)

	e.StackPush(call)
	defer func() {
		e.StackPop()
	}()
	result = fn.Call(e, blk, args...)
	return
}

func evalAndExpression(e pdsl.Evaluator, expr *parser.AndExpression) px.Value {
	return types.WrapBoolean(px.IsTruthy(e.Eval(expr.Lhs())) && px.IsTruthy(e.Eval(expr.Rhs())))
}

func evalOrExpression(e pdsl.Evaluator, expr *parser.OrExpression) px.Value {
	return types.WrapBoolean(px.IsTruthy(e.Eval(expr.Lhs())) || px.IsTruthy(e.Eval(expr.Rhs())))
}

func evalParameter(e pdsl.Evaluator, expr *parser.Parameter) px.Value {
	var pt px.Type
	if expr.Type() == nil {
		pt = types.DefaultAnyType()
	} else {
		pt = e.ResolveType(expr.Type())
	}

	var value px.Value
	if valueExpr := expr.Value(); valueExpr != nil {
		if lit, ok := literal.ToLiteral(valueExpr); ok {
			value = px.Wrap(e, lit)
		} else {
			var cf *parser.CallNamedFunctionExpression
			if cf, ok = valueExpr.(*parser.CallNamedFunctionExpression); ok {
				var qn *parser.QualifiedName
				if qn, ok = cf.Functor().(*parser.QualifiedName); ok {
					va := make([]px.Value, len(cf.Arguments()))
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
				value = newDeferredExpression(valueExpr)
			}
		}
	}
	return px.NewParameter(expr.Name(), pt, value, expr.CapturesRest())
}

func evalBlockExpression(e pdsl.Evaluator, expr *parser.BlockExpression) (result px.Value) {
	result = px.Undef
	for _, statement := range expr.Statements() {
		result = e.Eval(statement)
	}
	return result
}

func evalConcatenatedString(e pdsl.Evaluator, expr *parser.ConcatenatedString) px.Value {
	bld := bytes.NewBufferString(``)
	for _, s := range expr.Segments() {
		bld.WriteString(e.Eval(s).String())
	}
	return types.WrapString(bld.String())
}

func evalHeredocExpression(e pdsl.Evaluator, expr *parser.HeredocExpression) px.Value {
	return e.Eval(expr.Text())
}

func evalCallMethodExpression(e pdsl.Evaluator, call *parser.CallMethodExpression) px.Value {
	fc, ok := call.Functor().(*parser.NamedAccessExpression)
	if !ok {
		panic(evalError(validator.ValidateIllegalExpression, call.Functor(),
			issue.H{`expression`: call.Functor(), `feature`: `function accessor`, `container`: call}))
	}
	qn, ok := fc.Rhs().(*parser.QualifiedName)
	if !ok {
		panic(evalError(validator.ValidateIllegalExpression, call.Functor(),
			issue.H{`expression`: call.Functor(), `feature`: `function name`, `container`: call}))
	}
	receiver := unfold(e, []parser.Expression{fc.Lhs()})
	obj := receiver[0]
	var tem px.TypeWithCallableMembers
	if tem, ok = obj.PType().(px.TypeWithCallableMembers); ok {
		var mbr px.CallableMember
		if mbr, ok = tem.Member(qn.Name()); ok {
			var b px.Lambda
			if call.Lambda() != nil {
				b = e.Eval(call.Lambda()).(px.Lambda)
			}
			return mbr.Call(e, obj, b, unfold(e, call.Arguments()))
		}
	}
	return callFunction(e, qn.Name(), unfold(e, call.Arguments(), receiver...), call)
}

func evalCallNamedFunctionExpression(e pdsl.Evaluator, call *parser.CallNamedFunctionExpression) px.Value {
	fc := call.Functor()
	switch fc := fc.(type) {
	case *parser.QualifiedName:
		return callFunction(e, fc.Name(), unfold(e, call.Arguments()), call)
	case *parser.QualifiedReference:
		return callFunction(e, `new`, unfold(e, call.Arguments(), types.WrapString(fc.Name())), call)
	case *parser.AccessExpression:
		receiver := unfold(e, []parser.Expression{fc})
		return callFunction(e, `new`, unfold(e, call.Arguments(), receiver...), call)
	}
	panic(evalError(validator.ValidateIllegalExpression, call.Functor(),
		issue.H{`expression`: call.Functor(), `feature`: `function name`, `container`: call}))
}

func evalIfExpression(e pdsl.Evaluator, expr *parser.IfExpression) px.Value {
	return e.Scope().(pdsl.Scope).WithLocalScope(func() px.Value {
		if px.IsTruthy(e.Eval(expr.Test())) {
			return e.Eval(expr.Then())
		}
		return e.Eval(expr.Else())
	})
}

func evalInExpression(e pdsl.Evaluator, expr *parser.InExpression) px.Value {
	a := e.Eval(expr.Lhs())
	x := e.Eval(expr.Rhs())
	switch x := x.(type) {
	case *types.Array:
		return types.WrapBoolean(x.Any(func(b px.Value) bool {
			return doCompare(expr, `==`, a, b)
		}))
	case *types.Hash:
		return types.WrapBoolean(x.AnyPair(func(b, v px.Value) bool {
			return doCompare(expr, `==`, a, b)
		}))
	}
	return types.BooleanFalse
}

func evalUnlessExpression(e pdsl.Evaluator, expr *parser.UnlessExpression) px.Value {
	return e.Scope().(pdsl.Scope).WithLocalScope(func() px.Value {
		if !px.IsTruthy(e.Eval(expr.Test())) {
			return e.Eval(expr.Then())
		}
		return e.Eval(expr.Else())
	})
}

func evalKeyedEntry(e pdsl.Evaluator, expr *parser.KeyedEntry) px.Value {
	return types.WrapHashEntry(e.Eval(expr.Key()), e.Eval(expr.Value()))
}

func evalLambdaExpression(e pdsl.Evaluator, expr *parser.LambdaExpression) px.Value {
	return NewPuppetLambda(expr, e)
}

func evalLiteralHash(e pdsl.Evaluator, expr *parser.LiteralHash) px.Value {
	entries := expr.Entries()
	top := len(entries)
	if top == 0 {
		return px.EmptyMap
	}
	result := make([]*types.HashEntry, top)
	for idx := 0; idx < top; idx++ {
		result[idx] = e.Eval(entries[idx]).(*types.HashEntry)
	}
	return types.WrapHash(result)
}

func evalLiteralList(e pdsl.Evaluator, expr *parser.LiteralList) px.Value {
	es := expr.Elements()
	top := len(es)
	if top == 0 {
		return px.EmptyArray
	}
	result := make([]px.Value, top)
	for idx := 0; idx < top; idx++ {
		result[idx] = e.Eval(es[idx])
	}
	return types.WrapValues(result)
}

func evalLiteralBoolean(expr *parser.LiteralBoolean) px.Value {
	return types.WrapBoolean(expr.Bool())
}

func evalLiteralDefault() px.Value {
	return types.WrapDefault()
}

func evalLiteralFloat(expr *parser.LiteralFloat) px.Value {
	return types.WrapFloat(expr.Float())
}

func evalLiteralInteger(expr *parser.LiteralInteger) px.Value {
	return types.WrapInteger(expr.Int())
}

func evalLiteralString(expr *parser.LiteralString) px.Value {
	return types.WrapString(expr.StringValue())
}

func evalNotExpression(e pdsl.Evaluator, expr *parser.NotExpression) px.Value {
	return types.WrapBoolean(!px.IsTruthy(e.Eval(expr.Expr())))
}

func evalParenthesizedExpression(e pdsl.Evaluator, expr *parser.ParenthesizedExpression) px.Value {
	return e.Eval(expr.Expr())
}

func evalProgram(e pdsl.Evaluator, expr *parser.Program) px.Value {
	e.StackPush(expr)
	defer func() {
		e.StackPop()
	}()
	return e.Eval(expr.Body())
}

func evalQualifiedName(expr *parser.QualifiedName) px.Value {
	return types.WrapString(expr.Name())
}

func evalQualifiedReference(e pdsl.Evaluator, expr *parser.QualifiedReference) px.Value {
	return types.Resolve(e, expr.Name())
}

func evalRegexpExpression(expr *parser.RegexpExpression) px.Value {
	return types.WrapRegexp(expr.PatternString())
}

func evalCaseExpression(e pdsl.Evaluator, expr *parser.CaseExpression) px.Value {
	return e.Scope().(pdsl.Scope).WithLocalScope(func() px.Value {
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
					if e.Eval(cv).(px.List).Any(func(v px.Value) bool { return match(e, expr.Test(), cv, `match`, test, v) }) {
						selected = co
						break options
					}
				default:
					if match(e, expr.Test(), cv, `match`, test, e.Eval(cv)) {
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
			return px.Undef
		}
		return e.Eval(selected.Then())
	})
}

func evalSelectorExpression(e pdsl.Evaluator, expr *parser.SelectorExpression) px.Value {
	return e.Scope().(pdsl.Scope).WithLocalScope(func() px.Value {
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
				if e.Eval(me).(px.List).Any(func(v px.Value) bool { return match(e, expr.Lhs(), me, `match`, test, v) }) {
					selected = se
					break selectors
				}
			default:
				if match(e, expr.Lhs(), me, `match`, test, e.Eval(me)) {
					selected = se
					break selectors
				}
			}
		}
		if selected == nil {
			selected = theDefault
		}
		if selected == nil {
			return px.Undef
		}
		return e.Eval(selected.Value())
	})
}

func evalTextExpression(e pdsl.Evaluator, expr *parser.TextExpression) px.Value {
	return types.WrapString(e.Eval(expr.Expr()).String())
}

func evalVariableExpression(e pdsl.Evaluator, expr *parser.VariableExpression) (value px.Value) {
	name, ok := expr.Name()
	if ok {
		if value, ok = e.Scope().(pdsl.Scope).Get2(name); ok {
			return value
		}
		panic(evalError(px.UnknownVariable, expr, issue.H{`name`: name}))
	}
	idx, _ := expr.Index()
	if value, ok = e.Scope().(pdsl.Scope).RxGet(int(idx)); ok {
		return value
	}
	panic(evalError(px.UnknownVariable, expr, issue.H{`name`: idx}))
}

func evalUnfoldExpression(e pdsl.Evaluator, expr *parser.UnfoldExpression) px.Value {
	candidate := e.Eval(expr.Expr())
	switch candidate := candidate.(type) {
	case *types.UndefValue:
		return types.SingletonArray(px.Undef)
	case *types.Array:
		return candidate
	case *types.Hash:
		return types.WrapArray3(candidate)
	case px.IteratorValue:
		return candidate.(px.IteratorValue).AsArray()
	default:
		return types.SingletonArray(candidate)
	}
}

func evalError(code issue.Code, location issue.Location, args issue.H) issue.Reported {
	return issue.NewReported(code, issue.SeverityError, args, location)
}

// BasicEval is exported to enable the evaluator to be extended
func BasicEval(e pdsl.Evaluator, expr parser.Expression) px.Value {
	switch ex := expr.(type) {
	case *parser.AccessExpression:
		return evalAccessExpression(e, ex)
	case *parser.AndExpression:
		return evalAndExpression(e, ex)
	case *parser.ArithmeticExpression:
		return evalArithmeticExpression(e, ex)
	case *parser.ComparisonExpression:
		return evalComparisonExpression(e, ex)
	case *parser.HeredocExpression:
		return evalHeredocExpression(e, ex)
	case *parser.InExpression:
		return evalInExpression(e, ex)
	case *parser.KeyedEntry:
		return evalKeyedEntry(e, ex)
	case *parser.LiteralHash:
		return evalLiteralHash(e, ex)
	case *parser.LiteralList:
		return evalLiteralList(e, ex)
	case *parser.NotExpression:
		return evalNotExpression(e, ex)
	case *parser.OrExpression:
		return evalOrExpression(e, ex)
	case *parser.QualifiedName:
		return evalQualifiedName(ex)
	case *parser.QualifiedReference:
		return evalQualifiedReference(e, ex)
	case *parser.ParenthesizedExpression:
		return evalParenthesizedExpression(e, ex)
	case *parser.RegexpExpression:
		return evalRegexpExpression(ex)
	case *parser.LiteralBoolean:
		return evalLiteralBoolean(ex)
	case *parser.LiteralDefault:
		return evalLiteralDefault()
	case *parser.LiteralFloat:
		return evalLiteralFloat(ex)
	case *parser.LiteralInteger:
		return evalLiteralInteger(ex)
	case *parser.LiteralString:
		return evalLiteralString(ex)
	case *parser.LiteralUndef, *parser.Nop:
		return px.Undef
	case *parser.TextExpression:
		return evalTextExpression(e, ex)
	case pdsl.ParserExtension:
		return ex.Evaluate(e)
	}

	if e.Static() {
		panic(evalError(pdsl.IllegalWhenStaticExpression, expr, issue.H{`expression`: expr}))
	}

	switch ex := expr.(type) {
	case *parser.AssignmentExpression:
		return evalAssignmentExpression(e, ex)
	case *parser.BlockExpression:
		return evalBlockExpression(e, ex)
	case *parser.CallMethodExpression:
		return evalCallMethodExpression(e, ex)
	case *parser.CallNamedFunctionExpression:
		return evalCallNamedFunctionExpression(e, ex)
	case *parser.CaseExpression:
		return evalCaseExpression(e, ex)
	case *parser.ConcatenatedString:
		return evalConcatenatedString(e, ex)
	case *parser.IfExpression:
		return evalIfExpression(e, ex)
	case *parser.LambdaExpression:
		return evalLambdaExpression(e, ex)
	case *parser.MatchExpression:
		return evalMatchExpression(e, ex)
	case *parser.Parameter:
		return evalParameter(e, ex)
	case *parser.Program:
		return evalProgram(e, ex)
	case *parser.SelectorExpression:
		return evalSelectorExpression(e, ex)
	case *parser.FunctionDefinition, *parser.PlanDefinition, *parser.StepExpression, *parser.TypeAlias, *parser.TypeMapping:
		// All definitions must be processed at this time
		return px.Undef
	case *parser.UnfoldExpression:
		return evalUnfoldExpression(e, ex)
	case *parser.UnlessExpression:
		return evalUnlessExpression(e, ex)
	case *parser.VariableExpression:
		return evalVariableExpression(e, ex)
	default:
		panic(evalError(pdsl.UnhandledExpression, expr, issue.H{`expression`: expr}))
	}
}

func unfold(e pdsl.Evaluator, array []parser.Expression, initial ...px.Value) []px.Value {
	result := make([]px.Value, len(initial), len(initial)+len(array))
	copy(result, initial)
	for _, ex := range array {
		ex = unwindParenthesis(ex)
		if u, ok := ex.(*parser.UnfoldExpression); ok {
			ev := e.Eval(u.Expr())
			switch ev := ev.(type) {
			case *types.Array:
				result = ev.AppendTo(result)
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
