package eval

type (
	Doer func()

	ContextDoer func(c Context)

	Consumer func(value Value)

	EntryMapper func(value MapEntry) MapEntry

	IndexedConsumer func(value Value, index int)

	SliceConsumer func(value List)

	Mapper func(value Value) Value

	Predicate func(value Value) bool

	Producer func() Value

	TypeMapper func(value Type) Value

	BiConsumer func(v1 Value, v2 Value)

	BiPredicate func(v1 Value, v2 Value) bool

	BiMapper func(v1 Value, v2 Value) Value
)
