package evaluator

type (
	Consumer func(value PValue)

	Mapper func(value PValue) PValue

	TypeMapper func(value PType) PValue

	BiMapper func(v1 PValue, v2 PValue) PValue

	Predicate func(value PValue) bool

	Producer func() PValue
)
