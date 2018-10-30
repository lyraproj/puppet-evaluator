package eval

type (
	Iterator interface {
		All(predicate Predicate) bool

		Any(predicate Predicate) bool

		AsArray() List

		Each(consumer Consumer)

		EachWithIndex(consumer BiConsumer)

		ElementType() Type

		Find(predicate Predicate) Value

		Find2(predicate Predicate, dflt Value) Value

		Find3(predicate Predicate, dflt Producer) Value

		Map(elementType Type, function Mapper) IteratorValue

		Next() (Value, bool)

		Reduce(redactor BiMapper) Value

		Reduce2(initialValue Value, redactor BiMapper) Value

		Reject(predicate Predicate) IteratorValue

		Select(predicate Predicate) IteratorValue
	}
)
