package eval

type (
	Lambda interface {
		PValue

		Call(c Context, block Lambda, args ...PValue) PValue

		Signature() Signature
	}

	Function interface {
		PValue

		Call(c Context, block Lambda, args ...PValue) PValue

		Dispatchers() []Lambda

		Name() string
	}

	ResolvableFunction interface {
		Name() string
		Resolve(c Context) Function
	}

	DispatchFunction func(c Context, args []PValue) PValue

	DispatchFunctionWithBlock func(c Context, args []PValue, block Lambda) PValue

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

		CallableWith(c Context, args IndexedValue, block Lambda) bool

		ParametersType() PType

		ReturnType() PType

		// BlockType returns a Callable, Optional[Callable], or nil to denote if a
		// block is required, optional, or invalid
		BlockType() PType

		// BlockName will typically return the string "block"
		BlockName() string

		// ParameterNames returns the names of the parameters. Will return the strings "1", "2", etc.
		// for unnamed parameters.
		ParameterNames() []string
	}

	DispatchCreator func(db Dispatch)

	LocalTypesCreator func(lt LocalTypes)
)

var BuildFunction func(name string, localTypes LocalTypesCreator, creators []DispatchCreator) ResolvableFunction

var NewGoFunction func(name string, creators ...DispatchCreator)

var NewGoFunction2 func(name string, localTypes LocalTypesCreator, creators ...DispatchCreator)

var NewGoConstructor func(typeName string, creators ...DispatchCreator)

var MakeGoAllocator func(allocFunc DispatchFunction) Lambda

var NewGoConstructor2 func(typeName string, localTypes LocalTypesCreator, creators ...DispatchCreator)

var MakeGoConstructor func(typeName string, creators ...DispatchCreator) ResolvableFunction

var MakeGoConstructor2 func(typeName string, localTypes LocalTypesCreator, creators ...DispatchCreator) ResolvableFunction
