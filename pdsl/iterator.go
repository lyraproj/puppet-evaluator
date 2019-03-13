package pdsl

import "github.com/lyraproj/pcore/px"

type (
	Iterator interface {
		All(predicate px.Predicate) bool

		Any(predicate px.Predicate) bool

		AsArray() px.List

		Each(consumer px.Consumer)

		EachWithIndex(consumer px.BiConsumer)

		ElementType() px.Type

		Find(predicate px.Predicate) px.Value

		Find2(predicate px.Predicate, dflt px.Value) px.Value

		Find3(predicate px.Predicate, dflt px.Producer) px.Value

		Map(elementType px.Type, function px.Mapper) px.IteratorValue

		Next() (px.Value, bool)

		Reduce(redactor px.BiMapper) px.Value

		Reduce2(initialValue px.Value, redactor px.BiMapper) px.Value

		Reject(predicate px.Predicate) px.IteratorValue

		Select(predicate px.Predicate) px.IteratorValue
	}
)
