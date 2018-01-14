package types

import (
	. "io"

	. "github.com/puppetlabs/go-evaluator/errors"
	. "github.com/puppetlabs/go-evaluator/evaluator"
)

type (
	IteratorType struct {
		typ PType
	}

	iteratorValue struct {
		iterator Iterator
	}

	indexedIterator struct {
		elementType PType
		pos         int
		indexed     IndexedValue
	}

	mappingIterator struct {
		elementType PType
		mapFunc     Mapper
		base        Iterator
	}

	predicateIterator struct {
		predicate Predicate
		outcome   bool
		base      Iterator
	}
)

var iteratorType_DEFAULT = &IteratorType{typ: DefaultAnyType()}

func DefaultIteratorType() *IteratorType {
	return iteratorType_DEFAULT
}

func NewIteratorType(elementType PType) *IteratorType {
	if elementType == nil || elementType == anyType_DEFAULT {
		return DefaultIteratorType()
	}
	return &IteratorType{elementType}
}

func NewIteratorType2(args ...PValue) *IteratorType {
	switch len(args) {
	case 0:
		return DefaultIteratorType()
	case 1:
		containedType, ok := args[0].(PType)
		if !ok {
			panic(NewIllegalArgumentType2(`Iterator[]`, 0, `Type`, args[0]))
		}
		return NewIteratorType(containedType)
	default:
		panic(NewIllegalArgumentCount(`Iterator[]`, `0 - 1`, len(args)))
	}
}

func (t *IteratorType) Accept(v Visitor, g Guard) {
	v(t)
	t.typ.Accept(v, g)
}

func (t *IteratorType) Default() PType {
	return iteratorType_DEFAULT
}

func (t *IteratorType) Equals(o interface{}, g Guard) bool {
	if ot, ok := o.(*IteratorType); ok {
		return t.typ.Equals(ot.typ, g)
	}
	return false
}

func (t *IteratorType) Generic() PType {
	return NewIteratorType(GenericType(t.typ))
}

func (t *IteratorType) IsAssignable(o PType, g Guard) bool {
	if it, ok := o.(*IteratorType); ok {
		return GuardedIsAssignable(t.typ, it.typ, g)
	}
	return false
}

func (t *IteratorType) IsInstance(o PValue, g Guard) bool {
	if it, ok := o.(Iterator); ok {
		return GuardedIsInstance(t.typ, it.ElementType(), g)
	}
	return false
}

func (t *IteratorType) Name() string {
	return `Iterator`
}

func (t *IteratorType) Parameters() []PValue {
	if t.typ == DefaultAnyType() {
		return EMPTY_VALUES
	}
	return []PValue{t.typ}
}

func (t *IteratorType) String() string {
	return ToString2(t, NONE)
}

func (t *IteratorType) ElementType() PType {
	return t.typ
}

func (t *IteratorType) ToString(b Writer, s FormatContext, g RDetect) {
	TypeToString(t, b, s, g)
}

func (t *IteratorType) Type() PType {
	return &TypeType{t}
}

func WrapIterator(iter Iterator) IteratorValue {
	return &iteratorValue{iter}
}

func (it *iteratorValue) AsArray() IndexedValue {
	return it.iterator.AsArray()
}

func (it *iteratorValue) Equals(o interface{}, g Guard) bool {
	if ot, ok := o.(*iteratorValue); ok {
		return it.iterator.ElementType().Equals(ot.iterator.ElementType(), g)
	}
	return false
}

func (it *iteratorValue) Type() PType {
	return NewIteratorType(it.iterator.ElementType())
}

func (it *iteratorValue) DynamicValue() Iterator {
	return it.iterator
}

func (it *iteratorValue) String() string {
	return ToString2(it, NONE)
}

func (it *iteratorValue) ToString(b Writer, s FormatContext, g RDetect) {
	if it.iterator.ElementType() != DefaultAnyType() {
		WriteString(b, `Iterator[`)
		GenericType(it.iterator.ElementType()).ToString(b, s, g)
		WriteString(b, `]-Value`)
	} else {
		WriteString(b, `Iterator-Value`)
	}
}

func stopIteration() {
	if err := recover(); err != nil {
		if _, ok := err.(*StopIteration); !ok {
			panic(err)
		}
	}
}

func find(iter Iterator, predicate Predicate, dflt PValue, dfltProducer Producer) (result PValue) {
	defer stopIteration()

	result = UNDEF
	ok := false
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

func each(iter Iterator, consumer Consumer) {
	defer stopIteration()

	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		consumer(v)
	}
}

func eachWithIndex(iter Iterator, consumer BiConsumer) {
	defer stopIteration()

	for idx := int64(0); ; idx++ {
		v, ok := iter.Next()
		if !ok {
			break
		}
		consumer(WrapInteger(idx), v)
	}
}

func all(iter Iterator, predicate Predicate) (result bool) {
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

func any(iter Iterator, predicate Predicate) (result bool) {
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

func reduce2(iter Iterator, value PValue, redactor BiMapper) (result PValue) {
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

func reduce(iter Iterator, redactor BiMapper) PValue {
	v, ok := iter.Next()
	if !ok {
		return _UNDEF
	}
	return reduce2(iter, v, redactor)
}

func asArray(iter Iterator) (result IndexedValue) {
	el := make([]PValue, 0, 16)
	defer func() {
		if err := recover(); err != nil {
			if _, ok := err.(*StopIteration); ok {
				result = WrapArray(el)
			} else {
				panic(err)
			}
		}
	}()

	for {
		v, ok := iter.Next()
		if !ok {
			result = WrapArray(el)
			break
		}
		if it, ok := v.(IteratorValue); ok {
			v = asArray(it.DynamicValue())
		}
		el = append(el, v)
	}
	return
}

func (ai *indexedIterator) All(predicate Predicate) bool {
	return all(ai, predicate)
}

func (ai *indexedIterator) Any(predicate Predicate) bool {
	return any(ai, predicate)
}

func (ai *indexedIterator) Each(consumer Consumer) {
	each(ai, consumer)
}

func (ai *indexedIterator) EachWithIndex(consumer BiConsumer) {
	eachWithIndex(ai, consumer)
}

func (ai *indexedIterator) ElementType() PType {
	return ai.elementType
}

func (ai *indexedIterator) Find(predicate Predicate) PValue {
	return find(ai, predicate, _UNDEF, nil)
}

func (ai *indexedIterator) Find2(predicate Predicate, dflt PValue) PValue {
	return find(ai, predicate, dflt, nil)
}

func (ai *indexedIterator) Find3(predicate Predicate, dflt Producer) PValue {
	return find(ai, predicate, nil, dflt)
}

func (ai *indexedIterator) Next() (PValue, bool) {
	pos := ai.pos + 1
	if pos < ai.indexed.Len() {
		ai.pos = pos
		return ai.indexed.At(pos), true
	}
	return _UNDEF, false
}

func (ai *indexedIterator) Map(elementType PType, mapFunc Mapper) IteratorValue {
	return WrapIterator(&mappingIterator{elementType, mapFunc, ai})
}

func (ai *indexedIterator) Reduce(redactor BiMapper) PValue {
	return reduce(ai, redactor)
}

func (ai *indexedIterator) Reduce2(initialValue PValue, redactor BiMapper) PValue {
	return reduce2(ai, initialValue, redactor)
}

func (ai *indexedIterator) Reject(predicate Predicate) IteratorValue {
	return WrapIterator(&predicateIterator{predicate, false, ai})
}

func (ai *indexedIterator) Select(predicate Predicate) IteratorValue {
	return WrapIterator(&predicateIterator{predicate, true, ai})
}

func (ai *indexedIterator) AsArray() IndexedValue {
	return ai.indexed
}

func (ai *predicateIterator) All(predicate Predicate) bool {
	return all(ai, predicate)
}

func (ai *predicateIterator) Any(predicate Predicate) bool {
	return any(ai, predicate)
}

func (ai *predicateIterator) Next() (v PValue, ok bool) {
	defer func() {
		if err := recover(); err != nil {
			if _, ok = err.(*StopIteration); ok {
				ok = false
				v = _UNDEF
			} else {
				panic(err)
			}
		}
	}()

	for {
		v, ok = ai.base.Next()
		if !ok {
			v = _UNDEF
			break
		}
		if ai.predicate(v) == ai.outcome {
			break
		}
	}
	return
}

func (ai *predicateIterator) Each(consumer Consumer) {
	each(ai, consumer)
}

func (ai *predicateIterator) EachWithIndex(consumer BiConsumer) {
	eachWithIndex(ai, consumer)
}

func (ai *predicateIterator) ElementType() PType {
	return ai.base.ElementType()
}

func (ai *predicateIterator) Find(predicate Predicate) PValue {
	return find(ai, predicate, _UNDEF, nil)
}

func (ai *predicateIterator) Find2(predicate Predicate, dflt PValue) PValue {
	return find(ai, predicate, dflt, nil)
}

func (ai *predicateIterator) Find3(predicate Predicate, dflt Producer) PValue {
	return find(ai, predicate, nil, dflt)
}

func (ai *predicateIterator) Map(elementType PType, mapFunc Mapper) IteratorValue {
	return WrapIterator(&mappingIterator{elementType, mapFunc, ai})
}

func (ai *predicateIterator) Reduce(redactor BiMapper) PValue {
	return reduce(ai, redactor)
}

func (ai *predicateIterator) Reduce2(initialValue PValue, redactor BiMapper) PValue {
	return reduce2(ai, initialValue, redactor)
}

func (ai *predicateIterator) Reject(predicate Predicate) IteratorValue {
	return WrapIterator(&predicateIterator{predicate, false, ai})
}

func (ai *predicateIterator) Select(predicate Predicate) IteratorValue {
	return WrapIterator(&predicateIterator{predicate, true, ai})
}

func (ai *predicateIterator) AsArray() IndexedValue {
	return asArray(ai)
}

func (ai *mappingIterator) All(predicate Predicate) bool {
	return all(ai, predicate)
}

func (ai *mappingIterator) Any(predicate Predicate) bool {
	return any(ai, predicate)
}

func (ai *mappingIterator) Next() (v PValue, ok bool) {
	v, ok = ai.base.Next()
	if !ok {
		v = _UNDEF
	} else {
		v = ai.mapFunc(v)
	}
	return
}

func (ai *mappingIterator) Each(consumer Consumer) {
	each(ai, consumer)
}

func (ai *mappingIterator) EachWithIndex(consumer BiConsumer) {
	eachWithIndex(ai, consumer)
}

func (ai *mappingIterator) ElementType() PType {
	return ai.elementType
}

func (ai *mappingIterator) Find(predicate Predicate) PValue {
	return find(ai, predicate, _UNDEF, nil)
}

func (ai *mappingIterator) Find2(predicate Predicate, dflt PValue) PValue {
	return find(ai, predicate, dflt, nil)
}

func (ai *mappingIterator) Find3(predicate Predicate, dflt Producer) PValue {
	return find(ai, predicate, nil, dflt)
}

func (ai *mappingIterator) Map(elementType PType, mapFunc Mapper) IteratorValue {
	return WrapIterator(&mappingIterator{elementType, mapFunc, ai})
}

func (ai *mappingIterator) Reduce(redactor BiMapper) PValue {
	return reduce(ai, redactor)
}

func (ai *mappingIterator) Reduce2(initialValue PValue, redactor BiMapper) PValue {
	return reduce2(ai, initialValue, redactor)
}

func (ai *mappingIterator) Reject(predicate Predicate) IteratorValue {
	return WrapIterator(&predicateIterator{predicate, false, ai})
}

func (ai *mappingIterator) Select(predicate Predicate) IteratorValue {
	return WrapIterator(&predicateIterator{predicate, true, ai})
}

func (ai *mappingIterator) AsArray() IndexedValue {
	return asArray(ai)
}
