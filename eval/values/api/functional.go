package api

type (
	Consumer func(value PValue)

	Mapper func(value PValue) PValue

	BiMapper func(v1 PValue, v2 PValue) PValue

	Predicate func(value PValue) bool

	Producer func() PValue
)
