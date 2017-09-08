package api

type (
	Iterator interface {
		All(predicate Predicate) bool

		Any(predicate Predicate) bool

		AsArray() IndexedValue

		Each(consumer Consumer)

		ElementType() PType

		Find(predicate Predicate) PValue

		Find2(predicate Predicate, dflt PValue) PValue

		Find3(predicate Predicate, dflt Producer) PValue

		Map(elementType PType, function Mapper) IteratorValue

		Next() (PValue, bool)

		Reduce(initialValue PValue, redactor BiMapper) PValue

		Reject(predicate Predicate) IteratorValue

		Select(predicate Predicate) IteratorValue
	}
)
