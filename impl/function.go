package impl

import (
	"bytes"
	"fmt"
	"io"
	"math"

	"github.com/lyraproj/puppet-evaluator/errors"
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/types"
	"github.com/lyraproj/puppet-parser/parser"
)

type (
	typeDecl struct {
		name string
		decl string
		tp   eval.Type
	}

	functionBuilder struct {
		name             string
		localTypeBuilder *localTypeBuilder
		dispatchers      []*dispatchBuilder
	}

	localTypeBuilder struct {
		localTypes []*typeDecl
	}

	dispatchBuilder struct {
		fb            *functionBuilder
		min           int64
		max           int64
		types         []eval.Type
		blockType     eval.Type
		optionalBlock bool
		returnType    eval.Type
		function      eval.DispatchFunction
		function2     eval.DispatchFunctionWithBlock
	}

	goFunction struct {
		name        string
		dispatchers []eval.Lambda
	}

	lambda struct {
		signature *types.CallableType
	}

	goLambda struct {
		lambda
		function eval.DispatchFunction
	}

	goLambdaWithBlock struct {
		lambda
		function eval.DispatchFunctionWithBlock
	}

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

func parametersFromSignature(s eval.Signature) []eval.Parameter {
	paramNames := s.ParameterNames()
	count := len(paramNames)
	tuple := s.ParametersType().(*types.TupleType)
	tz := tuple.Size()
	capture := -1
	if tz.Max() > int64(count) {
		capture = count - 1
	}
	paramTypes := s.ParametersType().(*types.TupleType).Types()
	ps := make([]eval.Parameter, len(paramNames))
	for i, paramName := range paramNames {
		ps[i] = NewParameter(paramName, paramTypes[i], nil, i == capture)
	}
	return ps
}

func (l *lambda) Equals(other interface{}, guard eval.Guard) bool {
	if ol, ok := other.(*lambda); ok {
		return l.signature.Equals(ol.signature, guard)
	}
	return false
}

func (l *lambda) String() string {
	return `lambda`
}

func (l *lambda) ToString(bld io.Writer, format eval.FormatContext, g eval.RDetect) {
	io.WriteString(bld, `lambda`)
}

func (l *lambda) PType() eval.Type {
	return l.signature
}

func (l *lambda) Signature() eval.Signature {
	return l.signature
}

func (l *goLambda) Call(c eval.Context, block eval.Lambda, args ...eval.Value) (result eval.Value) {
	result = l.function(c, args)
	return
}

func (l *goLambda) Parameters() []eval.Parameter {
	return parametersFromSignature(l.signature)
}

func (l *goLambdaWithBlock) Call(c eval.Context, block eval.Lambda, args ...eval.Value) (result eval.Value) {
	result = l.function(c, args, block)
	return
}

func (l *goLambdaWithBlock) Parameters() []eval.Parameter {
	return parametersFromSignature(l.signature)
}

var emptyTypeBuilder = &localTypeBuilder{[]*typeDecl{}}

func buildFunction(name string, localTypes eval.LocalTypesCreator, creators []eval.DispatchCreator) eval.ResolvableFunction {
	lt := emptyTypeBuilder
	if localTypes != nil {
		lt = &localTypeBuilder{make([]*typeDecl, 0, 8)}
		localTypes(lt)
	}

	fb := &functionBuilder{name: name, localTypeBuilder: lt, dispatchers: make([]*dispatchBuilder, len(creators))}
	dbs := fb.dispatchers
	fb.dispatchers = dbs
	for idx, creator := range creators {
		dbs[idx] = fb.newDispatchBuilder()
		creator(dbs[idx])
	}
	return fb
}

func (fb *functionBuilder) newDispatchBuilder() *dispatchBuilder {
	return &dispatchBuilder{fb: fb, types: make([]eval.Type, 0, 8), min: 0, max: 0, optionalBlock: false, blockType: nil, returnType: nil}
}

func (fb *functionBuilder) Name() string {
	return fb.name
}

func (fb *functionBuilder) Resolve(c eval.Context) eval.Function {
	ds := make([]eval.Lambda, len(fb.dispatchers))
	if len(fb.localTypeBuilder.localTypes) > 0 {
		localLoader := eval.NewParentedLoader(c.Loader())
		c.DoWithLoader(localLoader, func() {
			b := bytes.NewBufferString(``)
			for _, td := range fb.localTypeBuilder.localTypes {
				if td.tp == nil {
					b.WriteString(`type `)
					b.WriteString(td.name)
					b.WriteString(` = `)
					b.WriteString(td.decl)
					b.WriteByte('\n')
				} else {
					localLoader.SetEntry(eval.NewTypedName(eval.NsType, td.name), eval.NewLoaderEntry(td.tp, nil))
				}
			}

			s := b.String()
			if len(s) > 0 {
				c.AddDefinitions(c.ParseAndValidate(``, s, false))
			}
			c.ResolveDefinitions()
			for i, d := range fb.dispatchers {
				ds[i] = d.createDispatch(c)
			}
		})
	} else {
		for i, d := range fb.dispatchers {
			ds[i] = d.createDispatch(c)
		}
	}
	return &goFunction{fb.name, ds}
}

func (tb *localTypeBuilder) Type(name string, decl string) {
	tb.localTypes = append(tb.localTypes, &typeDecl{name, decl, nil})
}

func (tb *localTypeBuilder) Type2(name string, tp eval.Type) {
	tb.localTypes = append(tb.localTypes, &typeDecl{name, ``, tp})
}

func (db *dispatchBuilder) createDispatch(c eval.Context) eval.Lambda {
	for idx, tp := range db.types {
		if trt, ok := tp.(*types.TypeReferenceType); ok {
			db.types[idx] = c.ParseType2(trt.TypeString())
		}
	}
	if r, ok := db.blockType.(*types.TypeReferenceType); ok {
		db.blockType = c.ParseType2(r.TypeString())
	}
	if db.optionalBlock {
		db.blockType = types.NewOptionalType(db.blockType)
	}
	if r, ok := db.returnType.(*types.TypeReferenceType); ok {
		db.returnType = c.ParseType2(r.TypeString())
	}
	if db.function2 == nil {
		return &goLambda{lambda{types.NewCallableType(types.NewTupleType(db.types, types.NewIntegerType(db.min, db.max)), db.returnType, nil)}, db.function}
	}
	return &goLambdaWithBlock{lambda{types.NewCallableType(types.NewTupleType(db.types, types.NewIntegerType(db.min, db.max)), db.returnType, db.blockType)}, db.function2}
}

func (db *dispatchBuilder) Name() string {
	return db.fb.name
}

func (db *dispatchBuilder) Param(tp string) {
	db.Param2(types.NewTypeReferenceType(tp))
}

func (db *dispatchBuilder) Param2(tp eval.Type) {
	db.assertNotAfterRepeated()
	if db.min < db.max {
		panic(`Required parameters must not come after optional parameters in a dispatch`)
	}
	db.types = append(db.types, tp)
	db.min++
	db.max++
}

func (db *dispatchBuilder) OptionalParam(tp string) {
	db.OptionalParam2(types.NewTypeReferenceType(tp))
}

func (db *dispatchBuilder) OptionalParam2(tp eval.Type) {
	db.assertNotAfterRepeated()
	db.types = append(db.types, tp)
	db.max++
}

func (db *dispatchBuilder) RepeatedParam(tp string) {
	db.RepeatedParam2(types.NewTypeReferenceType(tp))
}

func (db *dispatchBuilder) RepeatedParam2(tp eval.Type) {
	db.assertNotAfterRepeated()
	db.types = append(db.types, tp)
	db.max = math.MaxInt64
}

func (db *dispatchBuilder) RequiredRepeatedParam(tp string) {
	db.RequiredRepeatedParam2(types.NewTypeReferenceType(tp))
}

func (db *dispatchBuilder) RequiredRepeatedParam2(tp eval.Type) {
	db.assertNotAfterRepeated()
	db.types = append(db.types, tp)
	db.min++
	db.max = math.MaxInt64
}

func (db *dispatchBuilder) Block(tp string) {
	db.Block2(types.NewTypeReferenceType(tp))
}

func (db *dispatchBuilder) Block2(tp eval.Type) {
	if db.returnType != nil {
		panic(`Block specified more than once`)
	}
	db.blockType = tp
}

func (db *dispatchBuilder) OptionalBlock(tp string) {
	db.OptionalBlock2(types.NewTypeReferenceType(tp))
}

func (db *dispatchBuilder) OptionalBlock2(tp eval.Type) {
	db.Block2(tp)
	db.optionalBlock = true
}

func (db *dispatchBuilder) Returns(tp string) {
	db.Returns2(types.NewTypeReferenceType(tp))
}

func (db *dispatchBuilder) Returns2(tp eval.Type) {
	if db.returnType != nil {
		panic(`Returns specified more than once`)
	}
	db.returnType = tp
}

func (db *dispatchBuilder) Function(df eval.DispatchFunction) {
	if _, ok := db.blockType.(*types.CallableType); ok {
		panic(`Dispatch requires a block. Use FunctionWithBlock`)
	}
	db.function = df
}

func (db *dispatchBuilder) Function2(df eval.DispatchFunctionWithBlock) {
	if db.blockType == nil {
		panic(`Dispatch does not expect a block. Use Function instead of FunctionWithBlock`)
	}
	db.function2 = df
}

func (db *dispatchBuilder) assertNotAfterRepeated() {
	if db.max == math.MaxInt64 {
		panic(`Repeated parameters can only occur last in a dispatch`)
	}
}

func (f *goFunction) Call(c eval.Context, block eval.Lambda, args ...eval.Value) eval.Value {
	for _, d := range f.dispatchers {
		if d.Signature().CallableWith(args, block) {
			return d.Call(c, block, args...)
		}
	}
	panic(errors.NewArgumentsError(f.name, eval.DescribeSignatures(signatures(f.dispatchers), types.WrapValues(args).DetailedType(), block)))
}

func signatures(lambdas []eval.Lambda) []eval.Signature {
	s := make([]eval.Signature, len(lambdas))
	for i, l := range lambdas {
		s[i] = l.Signature()
	}
	return s
}

func (f *goFunction) Dispatchers() []eval.Lambda {
	return f.dispatchers
}

func (f *goFunction) Name() string {
	return f.name
}

func (f *goFunction) Equals(other interface{}, g eval.Guard) bool {
	dc := len(f.dispatchers)
	if of, ok := other.(*goFunction); ok && f.name == of.name && dc == len(of.dispatchers) {
		for i := 0; i < dc; i++ {
			if !f.dispatchers[i].Equals(of.dispatchers[i], g) {
				return false
			}
		}
		return true
	}
	return false
}

func (f *goFunction) String() string {
	return fmt.Sprintf(`function %s`, f.name)
}

func (f *goFunction) ToString(bld io.Writer, format eval.FormatContext, g eval.RDetect) {
	fmt.Fprintf(bld, `function %s`, f.name)
}

func (f *goFunction) PType() eval.Type {
	top := len(f.dispatchers)
	variants := make([]eval.Type, top)
	for idx := 0; idx < top; idx++ {
		variants[idx] = f.dispatchers[idx].PType()
	}
	return types.NewVariantType(variants...)
}

func NewPuppetLambda(expr *parser.LambdaExpression, c eval.Context) eval.Lambda {
	rps := ResolveParameters(c, expr.Parameters())
	sg := CreateTupleType(rps)

	return &puppetLambda{types.NewCallableType(sg, ResolveReturnType(c, expr.ReturnType()), nil), expr, rps}
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
	v = CallBlock(c, `lambda`, l.parameters, l.signature, l.expression.Body(), args)
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
	io.WriteString(bld, `lambda`)
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
			switch err.(type) {
			case *errors.NextIteration:
				v = err.(*errors.NextIteration).Value()
			case *errors.Return:
				v = err.(*errors.Return).Value()
			default:
				panic(err)
			}
		}
	}()
	v = CallBlock(c, f.Name(), f.parameters, f.signature, f.expression.Body(), args)
	return
}

func (f *puppetFunction) Signature() eval.Signature {
	return f.signature
}

func CallBlock(c eval.Context, name string, parameters []eval.Parameter, signature *types.CallableType, body parser.Expression, args []eval.Value) eval.Value {
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
						ap[idx] = eval.UNDEF
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
				return eval.UNDEF
			})
		}

		for idx, arg := range args {
			AssertArgument(name, idx, parameters[idx].Type(), arg)
		}

		scope := c.Scope()
		for idx, p := range parameters {
			scope.Set(p.Name(), args[idx])
		}
		v = eval.Evaluate(c, body)
		if !eval.IsInstance(signature.ReturnType(), v) {
			panic(fmt.Sprintf(`Value returned from function '%s' has incorrect type. Expected %s, got %s`,
				name, signature.ReturnType().String(), eval.DetailedValueType(v).String()))
		}
		return
	})
}

func AssertArgument(name string, index int, pt eval.Type, arg eval.Value) {
	if !eval.IsInstance(pt, arg) {
		panic(types.NewIllegalArgumentType2(name, index, pt.String(), arg))
	}
}

func (f *puppetFunction) Dispatchers() []eval.Lambda {
	return []eval.Lambda{f}
}

func (f *puppetFunction) Defaults() []eval.Value {
	dflts := make([]eval.Value, len(f.parameters))
	for i, p := range f.parameters {
		dflts[i] = p.Value()
	}
	return dflts
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
	f.parameters = ResolveParameters(c, f.expression.Parameters())
	f.signature = types.NewCallableType(CreateTupleType(f.parameters), ResolveReturnType(c, f.expression.ReturnType()), nil)
}

func (f *puppetFunction) ReturnType() parser.Expression {
	return f.expression.ReturnType()
}

func (f *puppetFunction) String() string {
	return eval.ToString(f)
}

func (f *puppetFunction) ToString(bld io.Writer, format eval.FormatContext, g eval.RDetect) {
	io.WriteString(bld, `function `)
	io.WriteString(bld, f.Name())
}

func (f *puppetFunction) PType() eval.Type {
	return f.signature
}

func NewPuppetPlan(expr *parser.PlanDefinition) *puppetPlan {
	return &puppetPlan{puppetFunction{expression: &expr.FunctionDefinition}}
}

func (p *puppetPlan) ToString(bld io.Writer, format eval.FormatContext, g eval.RDetect) {
	io.WriteString(bld, `plan `)
	io.WriteString(bld, p.Name())
}

func (p *puppetPlan) String() string {
	return eval.ToString(p)
}

func CreateTupleType(params []eval.Parameter) *types.TupleType {
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

func ResolveReturnType(c eval.Context, typeExpr parser.Expression) eval.Type {
	if typeExpr == nil {
		return types.DefaultAnyType()
	}
	return c.ResolveType(typeExpr)
}

func ResolveParameters(c eval.Context, eps []parser.Expression) []eval.Parameter {
	pps := make([]eval.Parameter, len(eps))
	for idx, ep := range eps {
		pps[idx] = eval.Evaluate(c, ep).(eval.Parameter)
	}
	return pps
}

func init() {
	eval.BuildFunction = buildFunction

	eval.NewGoFunction = func(name string, creators ...eval.DispatchCreator) {
		eval.RegisterGoFunction(buildFunction(name, nil, creators))
	}

	eval.NewGoFunction2 = func(name string, localTypes eval.LocalTypesCreator, creators ...eval.DispatchCreator) {
		eval.RegisterGoFunction(buildFunction(name, localTypes, creators))
	}

	eval.MakeGoAllocator = func(allocFunc eval.DispatchFunction) eval.Lambda {
		return &goLambda{lambda{types.NewCallableType(types.EmptyTupleType(), nil, nil)}, allocFunc}
	}

	eval.MakeGoConstructor = func(typeName string, creators ...eval.DispatchCreator) eval.ResolvableFunction {
		return buildFunction(typeName, nil, creators)
	}

	eval.MakeGoConstructor2 = func(typeName string, localTypes eval.LocalTypesCreator, creators ...eval.DispatchCreator) eval.ResolvableFunction {
		return buildFunction(typeName, localTypes, creators)
	}
}
