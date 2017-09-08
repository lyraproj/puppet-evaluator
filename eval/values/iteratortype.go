package values

import (
	. "io"

	. "github.com/puppetlabs/go-evaluator/eval/errors"
	. "github.com/puppetlabs/go-evaluator/eval/utils"
	. "github.com/puppetlabs/go-evaluator/eval/values/api"
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

func (t *IteratorType) String() string {
	return ToString2(t, NONE)
}

func (t *IteratorType) ElementType() PType {
	return t.typ
}

func (t *IteratorType) ToString(bld Writer, format FormatContext, g RDetect) {
	WriteString(bld, `Iterator`)
	if t.typ != DefaultAnyType() {
		WriteByte(bld, '[')
		t.typ.ToString(bld, format, g)
		WriteByte(bld, ']')
	}
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

func find(iter Iterator, predicate Predicate, dflt PValue, dfltProducer Producer) PValue {
	for {
		v, ok := iter.Next()
		if !ok {
			if dfltProducer != nil {
				return dfltProducer()
			}
			return dflt
		}
		if predicate(v) {
			return v
		}
	}
}

func each(iter Iterator, consumer Consumer) {
	for {
		v, ok := iter.Next()
		if !ok {
			return
		}
		consumer(v)
	}
}

func all(iter Iterator, predicate Predicate) bool {
	for {
		v, ok := iter.Next()
		if !ok {
			return true
		}
		if !predicate(v) {
			return false
		}
	}
}

func any(iter Iterator, predicate Predicate) bool {
	for {
		v, ok := iter.Next()
		if !ok {
			return false
		}
		if predicate(v) {
			return true
		}
	}
}

func reduce(iter Iterator, value PValue, redactor BiMapper) PValue {
	for {
		v, ok := iter.Next()
		if !ok {
			return value
		}
		value = redactor(value, v)
	}
}

func asArray(iter Iterator) IndexedValue {
	el := make([]PValue, 0, 16)
	for {
		v, ok := iter.Next()
		if !ok {
			return WrapArray(el)
		}
		if it, ok := v.(IteratorValue); ok {
			v = asArray(it.DynamicValue())
		}
		el = append(el, v)
	}
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

func (ai *indexedIterator) Reduce(initialValue PValue, redactor BiMapper) PValue {
	return reduce(ai, initialValue, redactor)
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

func (ai *predicateIterator) Next() (PValue, bool) {
	for {
		v, ok := ai.base.Next()
		if !ok {
			return _UNDEF, false
		}
		if ai.predicate(v) == ai.outcome {
			return v, true
		}
	}
}

func (ai *predicateIterator) Each(consumer Consumer) {
	each(ai, consumer)
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

func (ai *predicateIterator) Reduce(initialValue PValue, redactor BiMapper) PValue {
	return reduce(ai, initialValue, redactor)
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

func (ai *mappingIterator) Next() (PValue, bool) {
	for {
		v, ok := ai.base.Next()
		if !ok {
			return _UNDEF, false
		}
		return ai.mapFunc(v), true
	}
}

func (ai *mappingIterator) Each(consumer Consumer) {
	each(ai, consumer)
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

func (ai *mappingIterator) Reduce(initialValue PValue, redactor BiMapper) PValue {
	return reduce(ai, initialValue, redactor)
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
