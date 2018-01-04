package evaluator

type (
	Lambda interface {
		PValue

		Call(c EvalContext, block Lambda, args ...PValue) PValue

		Signature() Signature
	}

	Function interface {
		PValue

		Call(c EvalContext, block Lambda, args ...PValue) PValue

		Dispatchers() []Lambda

		Name() string
	}

	DispatchFunction func(c EvalContext, args []PValue) PValue

	DispatchFunctionWithBlock func(c EvalContext, args []PValue, block Lambda) PValue

	LocalTypes interface {
		Type(name string, decl string)
		Type2(name string, tp PType)
	}

	// Dispatch is a builder to build function dispatchers (Lambdas)
	Dispatch interface {
		// Name returns the name of the owner function
		Name() string

		Param(typeString string)
		Param2(puppetType PType)

		OptionalParam(typeString string)
		OptionalParam2(puppetType PType)

		RepeatedParam(typeString string)
		RepeatedParam2(puppetType PType)

		RequiredRepeatedParam(typeString string)
		RequiredRepeatedParam2(puppetType PType)

		Block(typeString string)
		Block2(puppetType PType)

		OptionalBlock(typeString string)
		OptionalBlock2(puppetType PType)

		Returns(typeString string)
		Returns2(puppetType PType)

		Function(f DispatchFunction)
		Function2(f DispatchFunctionWithBlock)
	}

	Signature interface {
		PType

		CallableWith(args []PValue, block Lambda) bool

		ParametersType() PType

		ReturnType() PType

		// BlockType returns a Callable, Optional[Callable], or nil to denote if a
		// block is required, optional, or invalid
		BlockType() PType
	}

	DispatchCreator func(db Dispatch)

	LocalTypesCreator func(lt LocalTypes)
)

var NewGoFunction func(name string, creators ...DispatchCreator)

var NewGoFunction2 func(name string, localTypes LocalTypesCreator, creators ...DispatchCreator)

var NewGoConstructor func(typeName string, creators ...DispatchCreator)

var NewGoConstructor2 func(typeName string, localTypes LocalTypesCreator, creators ...DispatchCreator)
