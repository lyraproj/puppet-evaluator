package evaluator

import (
	"fmt"
	"io"
	"math"

	"github.com/lyraproj/pcore/errors"
	"github.com/lyraproj/pcore/eval"
	"github.com/lyraproj/pcore/types"
	"github.com/lyraproj/pcore/utils"
	"github.com/lyraproj/puppet-evaluator/pdsl"
	"github.com/lyraproj/puppet-parser/parser"
)

type (
	puppetLambda struct {
		signature  *types.CallableType
		expression *parser.LambdaExpression
		parameters []eval.Parameter
	}

	PuppetFunction interface {
		eval.Function
		Signature() eval.Signature
		Expression() parser.Definition
		ReturnType() parser.Expression
		Parameters() []eval.Parameter
	}

	puppetFunction struct {
		signature  *types.CallableType
		expression *parser.FunctionDefinition
		parameters []eval.Parameter
	}

	puppetPlan struct {
		puppetFunction
	}
)

func NewPuppetLambda(expr *parser.LambdaExpression, c pdsl.EvaluationContext) eval.Lambda {
	rps := resolveParameters(c, expr.Parameters())
	sg := createTupleType(rps)

	return &puppetLambda{types.NewCallableType(sg, resolveReturnType(c, expr.ReturnType()), nil), expr, rps}
}

func (l *puppetLambda) Call(c eval.Context, block eval.Lambda, args ...eval.Value) (v eval.Value) {
	if block != nil {
		panic(errors.NewArgumentsError(`lambda`, `nested lambdas are not supported`))
	}
	defer func() {
		if err := recover(); err != nil {
			if ni, ok := err.(*errors.NextIteration); ok {
				v = ni.Value()
			} else {
				panic(err)
			}
		}
	}()
	v = CallBlock(c.(pdsl.EvaluationContext), `lambda`, l.parameters, l.signature, l.expression.Body(), args)
	return
}

func (l *puppetLambda) Equals(other interface{}, guard eval.Guard) bool {
	ol, ok := other.(*puppetLambda)
	return ok && l.signature.Equals(ol.signature, guard)
}

func (l *puppetLambda) Parameters() []eval.Parameter {
	return l.parameters
}

func (l *puppetLambda) Signature() eval.Signature {
	return l.signature
}

func (l *puppetLambda) String() string {
	// TODO: Present lambda in a way meaningful to stack trace
	return `lambda`
}

func (l *puppetLambda) ToString(bld io.Writer, format eval.FormatContext, g eval.RDetect) {
	utils.WriteString(bld, `lambda`)
}

func (l *puppetLambda) PType() eval.Type {
	return l.signature
}

func NewPuppetFunction(expr *parser.FunctionDefinition) *puppetFunction {
	return &puppetFunction{expression: expr}
}

func (f *puppetFunction) Call(c eval.Context, block eval.Lambda, args ...eval.Value) (v eval.Value) {
	if block != nil {
		panic(errors.NewArgumentsError(f.Name(), `Puppet functions does not yet support lambdas`))
	}
	defer func() {
		if err := recover(); err != nil {
			switch err := err.(type) {
			case *errors.NextIteration:
				v = err.Value()
			case *errors.Return:
				v = err.Value()
			default:
				panic(err)
			}
		}
	}()
	v = CallBlock(c.(pdsl.EvaluationContext), f.Name(), f.parameters, f.signature, f.expression.Body(), args)
	return
}

func (f *puppetFunction) Signature() eval.Signature {
	return f.signature
}

func CallBlock(c pdsl.EvaluationContext, name string, parameters []eval.Parameter, signature *types.CallableType, body parser.Expression, args []eval.Value) eval.Value {
	return c.Scope().WithLocalScope(func() (v eval.Value) {
		na := len(args)
		np := len(parameters)
		if np > na {
			// Resolve parameter defaults in special parameter scope and assign values to function scope
			c.Scope().WithLocalScope(func() eval.Value {
				ap := make([]eval.Value, np)
				copy(ap, args)
				for idx := na; idx < np; idx++ {
					p := parameters[idx]
					if !p.HasValue() {
						ap[idx] = eval.Undef
						continue
					}
					d := p.Value()
					if df, ok := d.(types.Deferred); ok {
						d = df.Resolve(c)
					}
					if !eval.IsInstance(p.Type(), d) {
						panic(errors.NewArgumentsError(name, fmt.Sprintf("expected default for parameter 1 to be %s, got %s", p.Type(), d.PType())))
					}
					ap[idx] = d
				}
				args = ap
				return eval.Undef
			})
		}

		for idx, arg := range args {
			AssertArgument(name, idx, parameters[idx].Type(), arg)
		}

		scope := c.Scope()
		for idx, p := range parameters {
			scope.Set(p.Name(), args[idx])
		}
		v = pdsl.Evaluate(c, body)
		if !eval.IsInstance(signature.ReturnType(), v) {
			panic(fmt.Sprintf(`Value returned from function '%s' has incorrect type. Expected %s, got %s`,
				name, signature.ReturnType().String(), eval.DetailedValueType(v).String()))
		}
		return
	})
}

func AssertArgument(name string, index int, pt eval.Type, arg eval.Value) {
	if !eval.IsInstance(pt, arg) {
		panic(types.NewIllegalArgumentType(name, index, pt.String(), arg))
	}
}

func (f *puppetFunction) Dispatchers() []eval.Lambda {
	return []eval.Lambda{f}
}

func (f *puppetFunction) Defaults() []eval.Value {
	ds := make([]eval.Value, len(f.parameters))
	for i, p := range f.parameters {
		ds[i] = p.Value()
	}
	return ds
}

func (f *puppetFunction) Equals(other interface{}, guard eval.Guard) bool {
	of, ok := other.(*puppetFunction)
	return ok && f.signature.Equals(of.signature, guard)
}

func (f *puppetFunction) Expression() parser.Definition {
	return f.expression
}

func (f *puppetFunction) Name() string {
	return f.expression.Name()
}

func (f *puppetFunction) Parameters() []eval.Parameter {
	return f.parameters
}

func (f *puppetFunction) Resolve(c eval.Context) {
	if f.parameters != nil {
		panic(fmt.Sprintf(`Attempt to resolve already resolved function %s`, f.Name()))
	}
	ec := c.(pdsl.EvaluationContext)
	f.parameters = resolveParameters(ec, f.expression.Parameters())
	f.signature = types.NewCallableType(createTupleType(f.parameters), resolveReturnType(ec, f.expression.ReturnType()), nil)
}

func (f *puppetFunction) ReturnType() parser.Expression {
	return f.expression.ReturnType()
}

func (f *puppetFunction) String() string {
	return eval.ToString(f)
}

func (f *puppetFunction) ToString(bld io.Writer, format eval.FormatContext, g eval.RDetect) {
	utils.WriteString(bld, `function `)
	utils.WriteString(bld, f.Name())
}

func (f *puppetFunction) PType() eval.Type {
	return f.signature
}

func NewPuppetPlan(expr *parser.PlanDefinition) *puppetPlan {
	return &puppetPlan{puppetFunction{expression: &expr.FunctionDefinition}}
}

func (p *puppetPlan) ToString(bld io.Writer, format eval.FormatContext, g eval.RDetect) {
	utils.WriteString(bld, `plan `)
	utils.WriteString(bld, p.Name())
}

func (p *puppetPlan) String() string {
	return eval.ToString(p)
}

func createTupleType(params []eval.Parameter) *types.TupleType {
	min := 0
	max := len(params)
	tps := make([]eval.Type, max)
	for idx, p := range params {
		tps[idx] = p.Type()
		if !p.HasValue() {
			min++
		}
		if p.CapturesRest() {
			max = math.MaxInt64
		}
	}
	return types.NewTupleType(tps, types.NewIntegerType(int64(min), int64(max)))
}

func resolveReturnType(c pdsl.EvaluationContext, typeExpr parser.Expression) eval.Type {
	if typeExpr == nil {
		return types.DefaultAnyType()
	}
	return c.ResolveType(typeExpr)
}

func resolveParameters(c pdsl.EvaluationContext, eps []parser.Expression) []eval.Parameter {
	pps := make([]eval.Parameter, len(eps))
	for idx, ep := range eps {
		pps[idx] = pdsl.Evaluate(c, ep).(eval.Parameter)
	}
	return pps
}
