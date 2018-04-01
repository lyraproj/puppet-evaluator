package impl

import (
	"bytes"
	"fmt"
	"io"
	"math"

	"github.com/puppetlabs/go-evaluator/errors"
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/types"
	"github.com/puppetlabs/go-parser/parser"
)

type (
	parameter struct {
		pType eval.PType
		pExpr *parser.Parameter
	}

	typeDecl struct {
		name string
		decl string
		tp   eval.PType
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
		types         []eval.PType
		blockType     eval.PType
		optionalBlock bool
		returnType    eval.PType
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
		parameters []*parameter
	}

	puppetFunction struct {
		signature  *types.CallableType
		expression *parser.FunctionDefinition
		parameters []*parameter
	}

	puppetPlan struct {
		puppetFunction
	}
)

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

func (l *lambda) Type() eval.PType {
	return l.signature
}

func (l *lambda) Signature() eval.Signature {
	return l.signature
}

func (l *goLambda) Call(c eval.EvalContext, block eval.Lambda, args ...eval.PValue) (result eval.PValue) {
	result = l.function(c, args)
	return
}

func (l *goLambdaWithBlock) Call(c eval.EvalContext, block eval.Lambda, args ...eval.PValue) (result eval.PValue) {
	result = l.function(c, args, block)
	return
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
	return &dispatchBuilder{fb: fb, types: make([]eval.PType, 0, 8), min: 0, max: 0, optionalBlock: false, blockType: nil, returnType: nil}
}

func (fb *functionBuilder) Name() string {
	return fb.name
}

func (fb *functionBuilder) Resolve(c eval.EvalContext) eval.Function {
	if len(fb.localTypeBuilder.localTypes) > 0 {
		localLoader := eval.NewParentedLoader(c.Loader())
		localEval := NewEvaluator(localLoader, c.Logger())

		b := bytes.NewBufferString(``)
		for _, td := range fb.localTypeBuilder.localTypes {
			if td.tp == nil {
				b.WriteString(`type `)
				b.WriteString(td.name)
				b.WriteString(` = `)
				b.WriteString(td.decl)
				b.WriteByte('\n')
			} else {
				localLoader.SetEntry(eval.NewTypedName(eval.TYPE, td.name), eval.NewLoaderEntry(td.tp, nil))
			}
		}

		s := b.String()
		if len(s) > 0 {
			localEval.AddDefinitions(c.ParseAndValidate(``, s, false))
		}

		localEval.ResolveDefinitions(c)
		c = NewEvalContext(localEval, localLoader, NewScope(), c.Stack())
	}
	ds := make([]eval.Lambda, len(fb.dispatchers))
	for idx, d := range fb.dispatchers {
		ds[idx] = d.createDispatch(c)
	}
	return &goFunction{fb.name, ds}
}

func (tb *localTypeBuilder) Type(name string, decl string) {
	tb.localTypes = append(tb.localTypes, &typeDecl{name, decl, nil})
}

func (tb *localTypeBuilder) Type2(name string, tp eval.PType) {
	tb.localTypes = append(tb.localTypes, &typeDecl{name, ``, tp})
}

func (db *dispatchBuilder) createDispatch(c eval.EvalContext) eval.Lambda {
	for idx, tp := range db.types {
		if trt, ok := tp.(*types.TypeReferenceType); ok {
			db.types[idx] = c.ParseResolve(trt.TypeString())
		}
	}
	if r, ok := db.blockType.(*types.TypeReferenceType); ok {
		db.blockType = c.ParseResolve(r.TypeString())
	}
	if db.optionalBlock {
		db.blockType = types.NewOptionalType(db.blockType)
	}
	if r, ok := db.returnType.(*types.TypeReferenceType); ok {
		db.returnType = c.ParseResolve(r.TypeString())
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

func (db *dispatchBuilder) Param2(tp eval.PType) {
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

func (db *dispatchBuilder) OptionalParam2(tp eval.PType) {
	db.assertNotAfterRepeated()
	db.types = append(db.types, tp)
	db.max++
}

func (db *dispatchBuilder) RepeatedParam(tp string) {
	db.RepeatedParam2(types.NewTypeReferenceType(tp))
}

func (db *dispatchBuilder) RepeatedParam2(tp eval.PType) {
	db.assertNotAfterRepeated()
	db.types = append(db.types, tp)
	db.max = math.MaxInt64
}

func (db *dispatchBuilder) RequiredRepeatedParam(tp string) {
	db.RequiredRepeatedParam2(types.NewTypeReferenceType(tp))
}

func (db *dispatchBuilder) RequiredRepeatedParam2(tp eval.PType) {
	db.assertNotAfterRepeated()
	db.types = append(db.types, tp)
	db.min++
	db.max = math.MaxInt64
}

func (db *dispatchBuilder) Block(tp string) {
	db.Block2(types.NewTypeReferenceType(tp))
}

func (db *dispatchBuilder) Block2(tp eval.PType) {
	if db.returnType != nil {
		panic(`Block specified more than once`)
	}
	db.blockType = tp
}

func (db *dispatchBuilder) OptionalBlock(tp string) {
	db.OptionalBlock2(types.NewTypeReferenceType(tp))
}

func (db *dispatchBuilder) OptionalBlock2(tp eval.PType) {
	db.Block2(tp)
	db.optionalBlock = true
}

func (db *dispatchBuilder) Returns(tp string) {
	db.Returns2(types.NewTypeReferenceType(tp))
}

func (db *dispatchBuilder) Returns2(tp eval.PType) {
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

func (f *goFunction) Call(c eval.EvalContext, block eval.Lambda, args ...eval.PValue) eval.PValue {
	argsArray := types.WrapArray(args)
	for _, d := range f.dispatchers {
		if d.Signature().CallableWith(argsArray, block) {
			return d.Call(c, block, args...)
		}
	}
	panic(errors.NewArgumentsError(f.name, eval.DescribeSignatures(signatures(f.dispatchers), types.WrapArray(args).DetailedType(), block)))
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

func (f *goFunction) Type() eval.PType {
	top := len(f.dispatchers)
	variants := make([]eval.PType, top)
	for idx := 0; idx < top; idx++ {
		variants[idx] = f.dispatchers[idx].Type()
	}
	return types.NewVariantType(variants)
}

func NewPuppetLambda(expr *parser.LambdaExpression, c eval.EvalContext) eval.Lambda {
	rps := resolveParameters(c, expr.Parameters())
	sg := createTupleType(rps)

	return &puppetLambda{types.NewCallableType(sg, resolveReturnType(c, expr.ReturnType()), nil), expr, rps}
}

func (l *puppetLambda) Call(c eval.EvalContext, block eval.Lambda, args ...eval.PValue) (v eval.PValue) {
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
	v = doCall(c, `lambda`, l.parameters, l.signature, l.expression.Body(), args)
	return
}

func (l *puppetLambda) Equals(other interface{}, guard eval.Guard) bool {
	ol, ok := other.(*puppetLambda)
	return ok && l.signature.Equals(ol.signature, guard)
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

func (l *puppetLambda) Type() eval.PType {
	return l.signature
}

func NewPuppetFunction(expr *parser.FunctionDefinition) *puppetFunction {
	return &puppetFunction{expression: expr}
}

func (f *puppetFunction) Call(c eval.EvalContext, block eval.Lambda, args ...eval.PValue) (v eval.PValue) {
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
	v = doCall(c, f.Name(), f.parameters, f.signature, f.expression.Body(), args)
	return
}

func (f *puppetFunction) Signature() eval.Signature {
	return f.signature
}

func doCall(c eval.EvalContext, name string, parameters []*parameter, signature *types.CallableType, body parser.Expression, args []eval.PValue) eval.PValue {
	return c.Scope().WithLocalScope(func(functionScope eval.Scope) (v eval.PValue) {
		na := len(args)
		np := len(parameters)
		if np > na {
			// Resolve parameter defaults in special parameter scope and assign values to function scope
			c.Scope().WithLocalScope(func(paramScope eval.Scope) eval.PValue {
				ap := make([]eval.PValue, np)
				copy(ap, args)
				for idx := na; idx < np; idx++ {
					p := parameters[idx]
					if p.pExpr.Value() == nil {
						ap[idx] = eval.UNDEF
						continue
					}
					d := c.EvaluateIn(p.pExpr.Value(), paramScope)
					if !eval.IsInstance(p.pType, d) {
						panic(errors.NewArgumentsError(name, fmt.Sprintf("expected default for parameter 1 to be %s, got %s", p.pType, d.Type())))
					}
					ap[idx] = d
				}
				args = ap
				return eval.UNDEF
			})
		}

		for idx, arg := range args {
			AssertArgument(name, idx, parameters[idx].pType, arg)
		}

		for idx, p := range parameters {
			functionScope.Set(p.pExpr.Name(), args[idx])
		}
		v = c.EvaluateIn(body, functionScope)
		if !eval.IsInstance(signature.ReturnType(), v) {
			panic(fmt.Sprintf(`Value returned from function '%s' has incorrect type. Expected %s, got %s`,
				name, signature.ReturnType().String(), eval.DetailedValueType(v).String()))
		}
		return
	})
}

func AssertArgument(name string, index int, pt eval.PType, arg eval.PValue) {
	if !eval.IsInstance(pt, arg) {
		panic(types.NewIllegalArgumentType2(name, index, pt.String(), arg))
	}
}

func (f *puppetFunction) Dispatchers() []eval.Lambda {
	return []eval.Lambda{f}
}

func (f *puppetFunction) Equals(other interface{}, guard eval.Guard) bool {
	of, ok := other.(*puppetFunction)
	return ok && f.signature.Equals(of.signature, guard)
}

func (f *puppetFunction) Name() string {
	return f.expression.Name()
}

func (f *puppetFunction) String() string {
	return eval.ToString(f)
}

func (f *puppetFunction) ToString(bld io.Writer, format eval.FormatContext, g eval.RDetect) {
	io.WriteString(bld, `function `)
	io.WriteString(bld, f.Name())
}

func (f *puppetFunction) Type() eval.PType {
	return f.signature
}

func (f *puppetFunction) Resolve(c eval.EvalContext) {
	if f.parameters != nil {
		panic(fmt.Sprintf(`Attempt to resolve already resolved function %s`, f.Name()))
	}
	f.parameters = resolveParameters(c, f.expression.Parameters())
	f.signature = types.NewCallableType(createTupleType(f.parameters), resolveReturnType(c, f.expression.ReturnType()), nil)
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

func createTupleType(params []*parameter) *types.TupleType {
	min := 0
	max := len(params)
	tps := make([]eval.PType, max)
	for idx, p := range params {
		tps[idx] = p.pType
		if p.pExpr.Value() == nil {
			min++
		}
		if p.pExpr.CapturesRest() {
			max = math.MaxInt64
		}
	}
	return types.NewTupleType(tps, types.NewIntegerType(int64(min), int64(max)))
}

func resolveReturnType(c eval.EvalContext, typeExpr parser.Expression) eval.PType {
	if typeExpr == nil {
		return types.DefaultAnyType()
	}
	return c.ResolveType(typeExpr)
}

func resolveParameters(c eval.EvalContext, eps []parser.Expression) []*parameter {
	pps := make([]*parameter, len(eps))
	for idx, ep := range eps {
		pd := ep.(*parser.Parameter)
		var pt eval.PType
		if pd.Type() == nil {
			pt = types.DefaultAnyType()
		} else {
			pt = c.ResolveType(pd.Type())
		}
		pps[idx] = &parameter{pt, pd}
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
