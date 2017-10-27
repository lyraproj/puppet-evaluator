package evaluator

type (
	Loader interface {
		Load(name TypedName) (interface{}, bool)

		NameAuthority() URI
	}

	DefiningLoader interface {
		Loader

		ResolveGoFunctions(c EvalContext)

		SetEntry(name TypedName, value interface{})
	}
)
