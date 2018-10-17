package eval

type (
	Doer func()

	ContextDoer func(c Context)

	Consumer func(value PValue)

	EntryMapper func(value EntryValue) EntryValue

	IndexedConsumer func(value PValue, index int)

	SliceConsumer func(value IndexedValue)

	Mapper func(value PValue) PValue

	Predicate func(value PValue) bool

	Producer func() PValue

	TypeMapper func(value PType) PValue

	BiConsumer func(v1 PValue, v2 PValue)

	BiPredicate func(v1 PValue, v2 PValue) bool

	BiMapper func(v1 PValue, v2 PValue) PValue
)
