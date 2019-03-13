package evaluator

import (
	"io"

	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
	"github.com/lyraproj/pcore/utils"
	"github.com/lyraproj/puppet-evaluator/errors"
	"github.com/lyraproj/puppet-evaluator/pdsl"
)

type (
	indexedIterator struct {
		elementType px.Type
		pos         int
		indexed     px.Indexed
	}

	mappingIterator struct {
		elementType px.Type
		mapFunc     px.Mapper
		base        pdsl.Iterator
	}

	predicateIterator struct {
		predicate px.Predicate
		outcome   bool
		base      pdsl.Iterator
	}

	iteratorValue struct {
		iterator pdsl.Iterator
	}
)

func stopIteration() {
	if err := recover(); err != nil {
		if _, ok := err.(*errors.StopIteration); !ok {
			panic(err)
		}
	}
}

func find(iter pdsl.Iterator, predicate px.Predicate, dflt px.Value, dfltProducer px.Producer) (result px.Value) {
	defer stopIteration()

	result = px.Undef
	var ok bool
	for {
		result, ok = iter.Next()
		if !ok {
			if dfltProducer != nil {
				result = dfltProducer()
			} else {
				result = dflt
			}
			break
		}
		if predicate(result) {
			break
		}
	}
	return
}

func each(iter pdsl.Iterator, consumer px.Consumer) {
	defer stopIteration()

	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		consumer(v)
	}
}

func eachWithIndex(iter pdsl.Iterator, consumer px.BiConsumer) {
	defer stopIteration()

	for idx := int64(0); ; idx++ {
		v, ok := iter.Next()
		if !ok {
			break
		}
		consumer(types.WrapInteger(idx), v)
	}
}

func all(iter pdsl.Iterator, predicate px.Predicate) (result bool) {
	defer stopIteration()

	result = true
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if !predicate(v) {
			result = false
			break
		}
	}
	return
}

func any(iter pdsl.Iterator, predicate px.Predicate) (result bool) {
	defer stopIteration()

	result = false
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if predicate(v) {
			result = true
			break
		}
	}
	return
}

func reduce2(iter pdsl.Iterator, value px.Value, redactor px.BiMapper) (result px.Value) {
	defer stopIteration()

	result = value
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		result = redactor(result, v)
	}
	return
}

func reduce(iter pdsl.Iterator, redactor px.BiMapper) px.Value {
	v, ok := iter.Next()
	if !ok {
		return px.Undef
	}
	return reduce2(iter, v, redactor)
}

func asArray(iter pdsl.Iterator) (result px.List) {
	el := make([]px.Value, 0, 16)
	defer func() {
		if err := recover(); err != nil {
			if _, ok := err.(*errors.StopIteration); ok {
				result = types.WrapValues(el)
			} else {
				panic(err)
			}
		}
	}()

	for {
		v, ok := iter.Next()
		if !ok {
			result = types.WrapValues(el)
			break
		}
		if it, ok := v.(px.IteratorValue); ok {
			v = it.AsArray()
		}
		el = append(el, v)
	}
	return
}

func (ai *indexedIterator) All(predicate px.Predicate) bool {
	return all(ai, predicate)
}

func (ai *indexedIterator) Any(predicate px.Predicate) bool {
	return any(ai, predicate)
}

func (ai *indexedIterator) Each(consumer px.Consumer) {
	each(ai, consumer)
}

func (ai *indexedIterator) EachWithIndex(consumer px.BiConsumer) {
	eachWithIndex(ai, consumer)
}

func (ai *indexedIterator) ElementType() px.Type {
	return ai.elementType
}

func (ai *indexedIterator) Find(predicate px.Predicate) px.Value {
	return find(ai, predicate, px.Undef, nil)
}

func (ai *indexedIterator) Find2(predicate px.Predicate, dflt px.Value) px.Value {
	return find(ai, predicate, dflt, nil)
}

func (ai *indexedIterator) Find3(predicate px.Predicate, dflt px.Producer) px.Value {
	return find(ai, predicate, nil, dflt)
}

func (ai *indexedIterator) Next() (px.Value, bool) {
	pos := ai.pos + 1
	if pos < ai.indexed.Len() {
		ai.pos = pos
		return ai.indexed.At(pos), true
	}
	return px.Undef, false
}

func (ai *indexedIterator) Map(elementType px.Type, mapFunc px.Mapper) px.IteratorValue {
	return WrapIterator(&mappingIterator{elementType, mapFunc, ai})
}

func (ai *indexedIterator) Reduce(redactor px.BiMapper) px.Value {
	return reduce(ai, redactor)
}

func (ai *indexedIterator) Reduce2(initialValue px.Value, redactor px.BiMapper) px.Value {
	return reduce2(ai, initialValue, redactor)
}

func (ai *indexedIterator) Reject(predicate px.Predicate) px.IteratorValue {
	return WrapIterator(&predicateIterator{predicate, false, ai})
}

func (ai *indexedIterator) Select(predicate px.Predicate) px.IteratorValue {
	return WrapIterator(&predicateIterator{predicate, true, ai})
}

func (ai *indexedIterator) AsArray() px.List {
	return ai.indexed.(px.Arrayable).AsArray()
}

func (ai *predicateIterator) All(predicate px.Predicate) bool {
	return all(ai, predicate)
}

func (ai *predicateIterator) Any(predicate px.Predicate) bool {
	return any(ai, predicate)
}

func (ai *predicateIterator) Next() (v px.Value, ok bool) {
	defer func() {
		if err := recover(); err != nil {
			if _, ok = err.(*errors.StopIteration); ok {
				ok = false
				v = px.Undef
			} else {
				panic(err)
			}
		}
	}()

	for {
		v, ok = ai.base.Next()
		if !ok {
			v = px.Undef
			break
		}
		if ai.predicate(v) == ai.outcome {
			break
		}
	}
	return
}

func (ai *predicateIterator) Each(consumer px.Consumer) {
	each(ai, consumer)
}

func (ai *predicateIterator) EachWithIndex(consumer px.BiConsumer) {
	eachWithIndex(ai, consumer)
}

func (ai *predicateIterator) ElementType() px.Type {
	return ai.base.ElementType()
}

func (ai *predicateIterator) Find(predicate px.Predicate) px.Value {
	return find(ai, predicate, px.Undef, nil)
}

func (ai *predicateIterator) Find2(predicate px.Predicate, dflt px.Value) px.Value {
	return find(ai, predicate, dflt, nil)
}

func (ai *predicateIterator) Find3(predicate px.Predicate, dflt px.Producer) px.Value {
	return find(ai, predicate, nil, dflt)
}

func (ai *predicateIterator) Map(elementType px.Type, mapFunc px.Mapper) px.IteratorValue {
	return WrapIterator(&mappingIterator{elementType, mapFunc, ai})
}

func (ai *predicateIterator) Reduce(redactor px.BiMapper) px.Value {
	return reduce(ai, redactor)
}

func (ai *predicateIterator) Reduce2(initialValue px.Value, redactor px.BiMapper) px.Value {
	return reduce2(ai, initialValue, redactor)
}

func (ai *predicateIterator) Reject(predicate px.Predicate) px.IteratorValue {
	return WrapIterator(&predicateIterator{predicate, false, ai})
}

func (ai *predicateIterator) Select(predicate px.Predicate) px.IteratorValue {
	return WrapIterator(&predicateIterator{predicate, true, ai})
}

func (ai *predicateIterator) AsArray() px.List {
	return asArray(ai)
}

func (ai *mappingIterator) All(predicate px.Predicate) bool {
	return all(ai, predicate)
}

func (ai *mappingIterator) Any(predicate px.Predicate) bool {
	return any(ai, predicate)
}

func (ai *mappingIterator) Next() (v px.Value, ok bool) {
	v, ok = ai.base.Next()
	if !ok {
		v = px.Undef
	} else {
		v = ai.mapFunc(v)
	}
	return
}

func (ai *mappingIterator) Each(consumer px.Consumer) {
	each(ai, consumer)
}

func (ai *mappingIterator) EachWithIndex(consumer px.BiConsumer) {
	eachWithIndex(ai, consumer)
}

func (ai *mappingIterator) ElementType() px.Type {
	return ai.elementType
}

func (ai *mappingIterator) Find(predicate px.Predicate) px.Value {
	return find(ai, predicate, px.Undef, nil)
}

func (ai *mappingIterator) Find2(predicate px.Predicate, dflt px.Value) px.Value {
	return find(ai, predicate, dflt, nil)
}

func (ai *mappingIterator) Find3(predicate px.Predicate, dflt px.Producer) px.Value {
	return find(ai, predicate, nil, dflt)
}

func (ai *mappingIterator) Map(elementType px.Type, mapFunc px.Mapper) px.IteratorValue {
	return WrapIterator(&mappingIterator{elementType, mapFunc, ai})
}

func (ai *mappingIterator) Reduce(redactor px.BiMapper) px.Value {
	return reduce(ai, redactor)
}

func (ai *mappingIterator) Reduce2(initialValue px.Value, redactor px.BiMapper) px.Value {
	return reduce2(ai, initialValue, redactor)
}

func (ai *mappingIterator) Reject(predicate px.Predicate) px.IteratorValue {
	return WrapIterator(&predicateIterator{predicate, false, ai})
}

func (ai *mappingIterator) Select(predicate px.Predicate) px.IteratorValue {
	return WrapIterator(&predicateIterator{predicate, true, ai})
}

func (ai *mappingIterator) AsArray() px.List {
	return asArray(ai)
}

func WrapIterable(it px.Indexed) pdsl.Iterator {
	return &indexedIterator{elementType: it.ElementType(), indexed: it, pos: -1}
}

func WrapIterator(iter pdsl.Iterator) px.IteratorValue {
	return &iteratorValue{iter}
}

func (it *iteratorValue) AsArray() px.List {
	return it.iterator.AsArray()
}

func (it *iteratorValue) ElementType() px.Type {
	return it.iterator.ElementType()
}

func (it *iteratorValue) Equals(o interface{}, g px.Guard) bool {
	return it == o
}

func (it *iteratorValue) Next() (px.Value, bool) {
	return it.iterator.Next()
}

func (it *iteratorValue) PType() px.Type {
	return types.NewIteratorType(it.iterator.ElementType())
}

func (it *iteratorValue) String() string {
	return px.ToString2(it, types.None)
}

func (it *iteratorValue) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	if it.iterator.ElementType() != types.DefaultAnyType() {
		utils.WriteString(b, `Iterator[`)
		px.GenericType(it.iterator.ElementType()).ToString(b, s, g)
		utils.WriteString(b, `]-Value`)
	} else {
		utils.WriteString(b, `Iterator-Value`)
	}
}
