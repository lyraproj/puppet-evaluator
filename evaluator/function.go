package evaluator

import (
	"fmt"
	"io"
	"math"

	"github.com/lyraproj/issue/issue"

	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
	"github.com/lyraproj/pcore/utils"
	"github.com/lyraproj/puppet-evaluator/errors"
	"github.com/lyraproj/puppet-evaluator/pdsl"
	"github.com/lyraproj/puppet-parser/parser"
)

type (
	puppetLambda struct {
		signature  *types.CallableType
		expression *parser.LambdaExpression
		parameters []px.Parameter
	}

	// ParameterDefaults is implemented by functions that can have
	// default values for the parameters. Currently only applicable
	// to the Puppet DSL function.
	//
	// A default is often an instance of types.Deferred and it is the
	// callers responsibility to resolve.
	ParameterDefaults interface {
		Defaults() []px.Value
	}

	// CallNamed is implemented by functions that can be called with
	// named arguments.
	CallNamed interface {
		CallNamed(c px.Context, block px.Lambda, args px.OrderedMap) px.Value
	}

	PuppetFunction interface {
		px.Function
		Signature() px.Signature
		Expression() parser.Definition
		ReturnType() parser.Expression
		Parameters() []px.Parameter
	}

	puppetFunction struct {
		signature  *types.CallableType
		expression *parser.FunctionDefinition
		parameters []px.Parameter
	}

	puppetPlan struct {
		puppetFunction
	}
)

func NewPuppetLambda(expr *parser.LambdaExpression, c pdsl.EvaluationContext) px.Lambda {
	rps := resolveParameters(c, expr.Parameters())
	sg := createTupleType(rps)

	return &puppetLambda{types.NewCallableType(sg, resolveReturnType(c, expr.ReturnType()), nil), expr, rps}
}

func (l *puppetLambda) Call(c px.Context, block px.Lambda, args ...px.Value) (v px.Value) {
	if block != nil {
		panic(px.Error(px.IllegalArguments, issue.H{`function`: `lambda`, `message`: `nested lambdas are not supported`}))
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

func (l *puppetLambda) Equals(other interface{}, guard px.Guard) bool {
	ol, ok := other.(*puppetLambda)
	return ok && l.signature.Equals(ol.signature, guard)
}

func (l *puppetLambda) Parameters() []px.Parameter {
	return l.parameters
}

func (l *puppetLambda) Signature() px.Signature {
	return l.signature
}

func (l *puppetLambda) String() string {
	// TODO: Present lambda in a way meaningful to stack trace
	return `lambda`
}

func (l *puppetLambda) ToString(bld io.Writer, format px.FormatContext, g px.RDetect) {
	utils.WriteString(bld, `lambda`)
}

func (l *puppetLambda) PType() px.Type {
	return l.signature
}

func NewPuppetFunction(expr *parser.FunctionDefinition) *puppetFunction {
	return &puppetFunction{expression: expr}
}

func (f *puppetFunction) Call(c px.Context, block px.Lambda, args ...px.Value) (v px.Value) {
	if block != nil {
		panic(px.Error(px.IllegalArguments, issue.H{`function`: f.Name(), `message`: `Puppet functions does not yet support lambdas`}))
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

func (f *puppetFunction) Signature() px.Signature {
	return f.signature
}

func CallBlock(c pdsl.EvaluationContext, name string, parameters []px.Parameter, signature *types.CallableType, body parser.Expression, args []px.Value) px.Value {
	scope := c.Scope().(pdsl.Scope)
	return scope.WithLocalScope(func() (v px.Value) {
		na := len(args)
		np := len(parameters)
		if np > na {
			// Resolve parameter defaults in special parameter scope and assign values to function scope
			scope.WithLocalScope(func() px.Value {
				ap := make([]px.Value, np)
				copy(ap, args)
				for idx := na; idx < np; idx++ {
					p := parameters[idx]
					if !p.HasValue() {
						ap[idx] = px.Undef
						continue
					}
					d := p.Value()
					if df, ok := d.(types.Deferred); ok {
						d = df.Resolve(c, scope)
					}
					if !px.IsInstance(p.Type(), d) {
						panic(px.Error(px.IllegalArgumentType, issue.H{`function`: name, `index`: 1, `expected`: p.Type().String(), `actual`: d.PType().String()}))
					}
					ap[idx] = d
				}
				args = ap
				return px.Undef
			})
		}

		for idx, arg := range args {
			AssertArgument(name, idx, parameters[idx].Type(), arg)
		}

		for idx, p := range parameters {
			scope.Set(p.Name(), args[idx])
		}
		v = pdsl.Evaluate(c, body)
		if !px.IsInstance(signature.ReturnType(), v) {
			panic(fmt.Sprintf(`Value returned from function '%s' has incorrect type. Expected %s, got %s`,
				name, signature.ReturnType().String(), px.DetailedValueType(v).String()))
		}
		return
	})
}

func AssertArgument(name string, index int, pt px.Type, arg px.Value) {
	if !px.IsInstance(pt, arg) {
		panic(px.Error(px.IllegalArgumentType, issue.H{`function`: name, `index`: index, `expected`: pt.String(), `actual`: arg.PType().String()}))
	}
}

func (f *puppetFunction) Dispatchers() []px.Lambda {
	return []px.Lambda{f}
}

func (f *puppetFunction) Defaults() []px.Value {
	ds := make([]px.Value, len(f.parameters))
	for i, p := range f.parameters {
		ds[i] = p.Value()
	}
	return ds
}

func (f *puppetFunction) Equals(other interface{}, guard px.Guard) bool {
	of, ok := other.(*puppetFunction)
	return ok && f.signature.Equals(of.signature, guard)
}

func (f *puppetFunction) Expression() parser.Definition {
	return f.expression
}

func (f *puppetFunction) Name() string {
	return f.expression.Name()
}

func (f *puppetFunction) Parameters() []px.Parameter {
	return f.parameters
}

func (f *puppetFunction) Resolve(c px.Context) {
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
	return px.ToString(f)
}

func (f *puppetFunction) ToString(bld io.Writer, format px.FormatContext, g px.RDetect) {
	utils.WriteString(bld, `function `)
	utils.WriteString(bld, f.Name())
}

func (f *puppetFunction) PType() px.Type {
	return f.signature
}

func NewPuppetPlan(expr *parser.PlanDefinition) *puppetPlan {
	return &puppetPlan{puppetFunction{expression: &expr.FunctionDefinition}}
}

func (p *puppetPlan) ToString(bld io.Writer, format px.FormatContext, g px.RDetect) {
	utils.WriteString(bld, `plan `)
	utils.WriteString(bld, p.Name())
}

func (p *puppetPlan) String() string {
	return px.ToString(p)
}

func createTupleType(params []px.Parameter) *types.TupleType {
	min := 0
	max := len(params)
	tps := make([]px.Type, max)
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

func resolveReturnType(c pdsl.EvaluationContext, typeExpr parser.Expression) px.Type {
	if typeExpr == nil {
		return types.DefaultAnyType()
	}
	return c.ResolveType(typeExpr)
}

func resolveParameters(c pdsl.EvaluationContext, eps []parser.Expression) []px.Parameter {
	pps := make([]px.Parameter, len(eps))
	for idx, ep := range eps {
		pps[idx] = pdsl.Evaluate(c, ep).(px.Parameter)
	}
	return pps
}
