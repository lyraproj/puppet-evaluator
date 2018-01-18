package eval

import (
	"bytes"
	"fmt"
	. "io"
	"math"

	. "github.com/puppetlabs/go-evaluator/errors"
	. "github.com/puppetlabs/go-evaluator/evaluator"
	. "github.com/puppetlabs/go-evaluator/types"
	. "github.com/puppetlabs/go-parser/parser"
)

type (
	parameter struct {
		pType PType
		pExpr *Parameter
	}

	typeDecl struct {
		name string
		decl string
		tp   PType
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
		types         []PType
		blockType     PType
		optionalBlock bool
		returnType    PType
		function      DispatchFunction
		function2     DispatchFunctionWithBlock
	}

	goFunction struct {
		name        string
		dispatchers []Lambda
	}

	lambda struct {
		signature *CallableType
	}

	goLambda struct {
		lambda
		function DispatchFunction
	}

	goLambdaWithBlock struct {
		lambda
		function DispatchFunctionWithBlock
	}

	puppetLambda struct {
		signature  *CallableType
		expression *LambdaExpression
		parameters []*parameter
	}

	puppetFunction struct {
		signature  *CallableType
		expression *FunctionDefinition
		parameters []*parameter
	}
)

func (l *lambda) Equals(other interface{}, guard Guard) bool {
	if ol, ok := other.(*lambda); ok {
		return l.signature.Equals(ol.signature, guard)
	}
	return false
}

func (l *lambda) String() string {
	return `lambda`
}

func (l *lambda) ToString(bld Writer, format FormatContext, g RDetect) {
	WriteString(bld, `lambda`)
}

func (l *lambda) Type() PType {
	return l.signature
}

func (l *lambda) Signature() Signature {
	return l.signature
}

func (l *goLambda) Call(c EvalContext, block Lambda, args ...PValue) (result PValue) {
	result = l.function(c, args)
	return
}

func (l *goLambdaWithBlock) Call(c EvalContext, block Lambda, args ...PValue) (result PValue) {
	result = l.function(c, args, block)
	return
}

var emptyTypeBuilder = &localTypeBuilder{[]*typeDecl{}}

func buildFunction(name string, localTypes LocalTypesCreator, creators []DispatchCreator) *functionBuilder {
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
	return &dispatchBuilder{fb: fb, types: make([]PType, 0, 8), min: 0, max: 0, optionalBlock: false, blockType: nil, returnType: nil}
}

func (fb *functionBuilder) Name() string {
	return fb.name
}

func (fb *functionBuilder) Resolve(c EvalContext) Function {
	if len(fb.localTypeBuilder.localTypes) > 0 {
		localLoader := NewParentedLoader(c.Loader())
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
				localLoader.SetEntry(NewTypedName(TYPE, td.name), NewLoaderEntry(td.tp, ``))
			}
		}

		s := b.String()
		if len(s) > 0 {
			p, err := CreateParser().Parse(``, s, false, false)
			if err != nil {
				panic(err)
			}
			localEval.AddDefinitions(p)
		}

		localEval.ResolveDefinitions(c)
		c = NewEvalContext(localEval, localLoader, NewScope(), c.Stack())
	}
	ds := make([]Lambda, len(fb.dispatchers))
	for idx, d := range fb.dispatchers {
		ds[idx] = d.createDispatch(c.(TypeResolver))
	}
	return &goFunction{fb.name, ds}
}

func (tb *localTypeBuilder) Type(name string, decl string) {
	tb.localTypes = append(tb.localTypes, &typeDecl{name, decl, nil})
}

func (tb *localTypeBuilder) Type2(name string, tp PType) {
	tb.localTypes = append(tb.localTypes, &typeDecl{name, ``, tp})
}

func (db *dispatchBuilder) createDispatch(tr TypeResolver) Lambda {
	for idx, tp := range db.types {
		if trt, ok := tp.(*TypeReferenceType); ok {
			db.types[idx] = tr.ParseResolve(trt.TypeString())
		}
	}
	if r, ok := db.blockType.(*TypeReferenceType); ok {
		db.blockType = tr.ParseResolve(r.TypeString())
	}
	if db.optionalBlock {
		db.blockType = NewOptionalType(db.blockType)
	}
	if r, ok := db.returnType.(*TypeReferenceType); ok {
		db.returnType = tr.ParseResolve(r.TypeString())
	}
	if db.function2 == nil {
		return &goLambda{lambda{NewCallableType(NewTupleType(db.types, NewIntegerType(db.min, db.max)), db.returnType, nil)}, db.function}
	}
	return &goLambdaWithBlock{lambda{NewCallableType(NewTupleType(db.types, NewIntegerType(db.min, db.max)), db.returnType, db.blockType)}, db.function2}
}

func (db *dispatchBuilder) Name() string {
	return db.fb.name
}

func (db *dispatchBuilder) Param(tp string) {
	db.Param2(NewTypeReferenceType(tp))
}

func (db *dispatchBuilder) Param2(tp PType) {
	db.assertNotAfterRepeated()
	if db.min < db.max {
		panic(`Required parameters must not come after optional parameters in a dispatch`)
	}
	db.types = append(db.types, tp)
	db.min++
	db.max++
}

func (db *dispatchBuilder) OptionalParam(tp string) {
	db.OptionalParam2(NewTypeReferenceType(tp))
}

func (db *dispatchBuilder) OptionalParam2(tp PType) {
	db.assertNotAfterRepeated()
	db.types = append(db.types, tp)
	db.max++
}

func (db *dispatchBuilder) RepeatedParam(tp string) {
	db.RepeatedParam2(NewTypeReferenceType(tp))
}

func (db *dispatchBuilder) RepeatedParam2(tp PType) {
	db.assertNotAfterRepeated()
	db.types = append(db.types, tp)
	db.max = math.MaxInt64
}

func (db *dispatchBuilder) RequiredRepeatedParam(tp string) {
	db.RequiredRepeatedParam2(NewTypeReferenceType(tp))
}

func (db *dispatchBuilder) RequiredRepeatedParam2(tp PType) {
	db.assertNotAfterRepeated()
	db.types = append(db.types, tp)
	db.min++
	db.max = math.MaxInt64
}

func (db *dispatchBuilder) Block(tp string) {
	db.Block2(NewTypeReferenceType(tp))
}

func (db *dispatchBuilder) Block2(tp PType) {
	if db.returnType != nil {
		panic(`Block specified more than once`)
	}
	db.blockType = tp
}

func (db *dispatchBuilder) OptionalBlock(tp string) {
	db.OptionalBlock2(NewTypeReferenceType(tp))
}

func (db *dispatchBuilder) OptionalBlock2(tp PType) {
	db.Block2(tp)
	db.optionalBlock = true
}

func (db *dispatchBuilder) Returns(tp string) {
	db.Returns2(NewTypeReferenceType(tp))
}

func (db *dispatchBuilder) Returns2(tp PType) {
	if db.returnType != nil {
		panic(`Returns specified more than once`)
	}
	db.returnType = tp
}

func (db *dispatchBuilder) Function(df DispatchFunction) {
	if _, ok := db.blockType.(*CallableType); ok {
		panic(`Dispatch requires a block. Use FunctionWithBlock`)
	}
	db.function = df
}

func (db *dispatchBuilder) Function2(df DispatchFunctionWithBlock) {
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

func (f *goFunction) Call(c EvalContext, block Lambda, args ...PValue) (result PValue) {
	for _, d := range f.dispatchers {
		if d.Signature().CallableWith(args, block) {
			result = d.Call(c, block, args...)
			return
		}
	}
	panic(NewArgumentsError(f.name, DescribeSignatures(signatures(f.dispatchers), WrapArray(args).DetailedType(), block)))
}

func signatures(lambdas []Lambda) []Signature {
	s := make([]Signature, len(lambdas))
	for i, l := range lambdas {
		s[i] = l.Signature()
	}
	return s
}

func (f *goFunction) Dispatchers() []Lambda {
	return f.dispatchers
}

func (f *goFunction) Name() string {
	return f.name
}

func (f *goFunction) Equals(other interface{}, g Guard) bool {
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

func (f *goFunction) ToString(bld Writer, format FormatContext, g RDetect) {
	fmt.Fprintf(bld, `function %s`, f.name)
}

func (f *goFunction) Type() PType {
	top := len(f.dispatchers)
	variants := make([]PType, top)
	for idx := 0; idx < top; idx++ {
		variants[idx] = f.dispatchers[idx].Type()
	}
	return NewVariantType(variants)
}

func NewPuppetLambda(expr *LambdaExpression, tr TypeResolver) Lambda {
	rps := resolveParameters(tr, expr.Parameters())
	sg := createTupleType(rps)

	return &puppetLambda{NewCallableType(sg, resolveReturnType(tr, expr.ReturnType()), nil), expr, rps}
}

func (l *puppetLambda) Call(c EvalContext, block Lambda, args ...PValue) (v PValue) {
	if block != nil {
		panic(NewArgumentsError(`lambda`, `nested lambdas are not supported`))
	}
	defer func() {
		if err := recover(); err != nil {
			if ni, ok := err.(*NextIteration); ok {
				v = ni.Value()
			} else {
				panic(err)
			}
		}
	}()
	v = doCall(c, `lambda`, l.parameters, l.signature, l.expression.Body(), args)
	return
}

func (l *puppetLambda) Equals(other interface{}, guard Guard) bool {
	ol, ok := other.(*puppetLambda)
	return ok && l.signature.Equals(ol.signature, guard)
}

func (l *puppetLambda) Signature() Signature {
	return l.signature
}

func (l *puppetLambda) String() string {
	// TODO: Present lambda in a way meaningful to stack trace
	return `lambda`
}

func (l *puppetLambda) ToString(bld Writer, format FormatContext, g RDetect) {
	WriteString(bld, `lambda`)
}

func (l *puppetLambda) Type() PType {
	return l.signature
}

func NewPuppetFunction(expr *FunctionDefinition) *puppetFunction {
	return &puppetFunction{expression: expr}
}

func (f *puppetFunction) Call(c EvalContext, block Lambda, args ...PValue) (v PValue) {
	if block != nil {
		panic(NewArgumentsError(f.Name(), `Puppet functions does not yet support lambdas`))
	}
	defer func() {
		if err := recover(); err != nil {
			switch err.(type) {
			case *NextIteration:
				v = err.(*NextIteration).Value()
			case *Return:
				v = err.(*Return).Value()
			default:
				panic(err)
			}
		}
	}()
	v = doCall(c, f.Name(), f.parameters, f.signature, f.expression.Body(), args)
	return
}

func (f *puppetFunction) Signature() Signature {
	return f.signature
}

func doCall(c EvalContext, name string, parameters []*parameter, signature *CallableType, body Expression, args []PValue) PValue {
	return c.Scope().WithLocalScope(func(functionScope Scope) (v PValue) {
		na := len(args)
		np := len(parameters)
		if np > na {
			// Resolve parameter defaults in special parameter scope and assign values to function scope
			c.Scope().WithLocalScope(func(paramScope Scope) PValue {
				ap := make([]PValue, np)
				copy(ap, args)
				for idx := na; idx < np; idx++ {
					p := parameters[idx]
					if p.pExpr.Value() == nil {
						ap[idx] = UNDEF
						continue
					}
					d := c.EvaluateIn(p.pExpr.Value(), paramScope)
					if !IsInstance(p.pType, d) {
						panic(NewArgumentsError(name, fmt.Sprintf("expected default for parameter 1 to be %s, got %s", p.pType, d.Type())))
					}
					ap[idx] = d
				}
				args = ap
				return UNDEF
			})
		}

		for idx, arg := range args {
			AssertArgument(name, idx, parameters[idx].pType, arg)
		}

		for idx, p := range parameters {
			functionScope.Set(p.pExpr.Name(), args[idx])
		}
		v = c.EvaluateIn(body, functionScope)
		if !IsInstance(signature.ReturnType(), v) {
			panic(fmt.Sprintf(`Value returned from function '%s' has incorrect type. Expected %s, got %s`,
				name, signature.ReturnType().String(), DetailedValueType(v).String()))
		}
		return
	})
}

func AssertArgument(name string, index int, pt PType, arg PValue) {
	if !IsInstance(pt, arg) {
		panic(NewIllegalArgumentType2(name, index, pt.String(), arg))
	}
}

func (f *puppetFunction) Dispatchers() []Lambda {
	return []Lambda{f}
}

func (f *puppetFunction) Equals(other interface{}, guard Guard) bool {
	of, ok := other.(*puppetFunction)
	return ok && f.signature.Equals(of.signature, guard)
}

func (f *puppetFunction) Name() string {
	return f.expression.Name()
}

func (f *puppetFunction) ToString(bld Writer, format FormatContext, g RDetect) {
	WriteString(bld, `function`)
}

func (f *puppetFunction) Type() PType {
	return f.signature
}

func (f *puppetFunction) Resolve(tr TypeResolver) {
	if f.parameters != nil {
		panic(fmt.Sprintf(`Attempt to resolve already resolved function %s`, f.Name()))
	}
	f.parameters = resolveParameters(tr, f.expression.Parameters())
	f.signature = NewCallableType(createTupleType(f.parameters), resolveReturnType(tr, f.expression.ReturnType()), nil)
}

func createTupleType(params []*parameter) *TupleType {
	min := 0
	max := len(params)
	tps := make([]PType, max)
	for idx, p := range params {
		tps[idx] = p.pType
		if p.pExpr.Value() == nil {
			min++
		}
		if p.pExpr.CapturesRest() {
			max = math.MaxInt64
		}
	}
	return NewTupleType(tps, NewIntegerType(int64(min), int64(max)))
}

func resolveReturnType(tr TypeResolver, typeExpr Expression) PType {
	if typeExpr == nil {
		return DefaultAnyType()
	}
	return tr.ResolveType(typeExpr)
}

func resolveParameters(tr TypeResolver, eps []Expression) []*parameter {
	pps := make([]*parameter, len(eps))
	for idx, ep := range eps {
		pd := ep.(*Parameter)
		var pt PType
		if pd.Type() == nil {
			pt = DefaultAnyType()
		} else {
			pt = tr.ResolveType(pd.Type())
		}
		pps[idx] = &parameter{pt, pd}
	}
	return pps
}

func (f *puppetFunction) String() string {
	// TODO: Return proper format suitable for stack trace entry
	return f.Name()
}

func init() {

	NewGoFunction = func(name string, creators ...DispatchCreator) {
		RegisterGoFunction(buildFunction(name, nil, creators))
	}

	NewGoFunction2 = func(name string, localTypes LocalTypesCreator, creators ...DispatchCreator) {
		RegisterGoFunction(buildFunction(name, localTypes, creators))
	}

	NewGoConstructor = func(typeName string, creators ...DispatchCreator) {
		RegisterGoConstructor(buildFunction(typeName, nil, creators))
	}

	NewGoConstructor2 = func(typeName string, localTypes LocalTypesCreator, creators ...DispatchCreator) {
		RegisterGoConstructor(buildFunction(typeName, localTypes, creators))
	}
}
