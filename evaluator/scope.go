package evaluator

type (
	ValueProducer func(scope Scope) PValue

	Scope interface {
		WithLocalScope(producer ValueProducer) PValue

		Get(name string) (value PValue, found bool)

		Set(name string, value PValue) bool

		RxSet(variables []string)

		RxGet(index int) (value PValue, found bool)
	}
)
