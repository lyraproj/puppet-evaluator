package types

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"sync"

	"github.com/puppetlabs/go-evaluator/errors"
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/utils"
)

type (
	ArrayType struct {
		size *IntegerType
		typ  eval.PType
	}

	ArrayValue struct {
		lock         sync.Mutex
		reducedType  eval.PType
		detailedType eval.PType
		elements     []eval.PValue
	}
)

var Array_Type eval.ObjectType

func init() {
	Array_Type = newObjectType(`Pcore::ArrayType`,
		`Pcore::CollectionType {
  attributes => {
    'element_type' => { type => Type, value => Any }
  },
  serialization => [ 'element_type', 'size_type' ]
}`, func(ctx eval.EvalContext, args []eval.PValue) eval.PValue {
			return NewArrayType2(args...)
		})
}

func DefaultArrayType() *ArrayType {
	return arrayType_DEFAULT
}

func EmptyArrayType() *ArrayType {
	return arrayType_EMPTY
}

func NewArrayType(element eval.PType, rng *IntegerType) *ArrayType {
	if element == nil {
		element = anyType_DEFAULT
	}
	if rng == nil {
		rng = integerType_POSITIVE
	}
	if *rng == *integerType_POSITIVE && element == anyType_DEFAULT {
		return DefaultArrayType()
	}
	if *rng == *integerType_ZERO && element == unitType_DEFAULT {
		return EmptyArrayType()
	}
	return &ArrayType{rng, element}
}

func NewArrayType2(args ...eval.PValue) *ArrayType {
	argc := len(args)
	if argc == 0 {
		return DefaultArrayType()
	}

	offset := 0
	element, ok := args[0].(eval.PType)
	if ok {
		offset++
	} else {
		element = DefaultAnyType()
	}

	var rng *IntegerType
	switch argc - offset {
	case 0:
		rng = integerType_POSITIVE
	case 1:
		sizeArg := args[offset]
		if rng, ok = sizeArg.(*IntegerType); !ok {
			var sz int64
			sz, ok = toInt(sizeArg)
			if !ok {
				panic(NewIllegalArgumentType2(`Array[]`, offset, `Variant[Integer, Type[Integer]]`, sizeArg))
			}
			rng = NewIntegerType(sz, math.MaxInt64)
		}
	case 2:
		var min, max int64
		arg := args[offset]
		if min, ok = toInt(arg); !ok {
			if _, ok = arg.(*DefaultValue); !ok {
				panic(NewIllegalArgumentType2(`Array[]`, offset, `Integer`, arg))
			}
			min = 0
		}
		offset++
		arg = args[offset]
		if max, ok = toInt(args[offset]); !ok {
			if _, ok = arg.(*DefaultValue); !ok {
				panic(NewIllegalArgumentType2(`Array[]`, offset, `Integer`, arg))
			}
			max = math.MaxInt64
		}
		rng = NewIntegerType(min, max)
	default:
		panic(errors.NewIllegalArgumentCount(`Array[]`, `0 - 3`, argc))
	}
	return NewArrayType(element, rng)
}

func (t *ArrayType) ElementType() eval.PType {
	return t.typ
}

func (t *ArrayType) Accept(v eval.Visitor, g eval.Guard) {
	v(t)
	t.size.Accept(v, g)
	t.typ.Accept(v, g)
}

func (t *ArrayType) Equals(o interface{}, g eval.Guard) bool {
	if ot, ok := o.(*ArrayType); ok {
		return t.typ.Equals(ot.typ, g)
	}
	return false
}

func (t *ArrayType) Generic() eval.PType {
	if t.typ == anyType_DEFAULT {
		return arrayType_DEFAULT
	}
	return NewArrayType(eval.Generalize(t.typ), nil)
}

func (t *ArrayType) Get(key string) (value eval.PValue, ok bool) {
	switch key {
	case `element_type`:
		return t.typ, true
	case `size_type`:
		return t.size, true
	}
	return nil, false
}

func (t *ArrayType) Default() eval.PType {
	return arrayType_DEFAULT
}

func (t *ArrayType) IsAssignable(o eval.PType, g eval.Guard) bool {
	switch o.(type) {
	case *ArrayType:
		oa := o.(*ArrayType)
		return t.size.IsAssignable(oa.size, g) && GuardedIsAssignable(t.typ, oa.typ, g)
	case *TupleType:
		ot := o.(*TupleType)
		return t.size.IsAssignable(ot.givenOrActualSize, g) && allAssignableTo(ot.types, t.typ, g)
	default:
		return false
	}
	return true
}

func (t *ArrayType) IsInstance(v eval.PValue, g eval.Guard) bool {
	iv, ok := v.(*ArrayValue)
	if !ok {
		return false
	}

	osz := iv.Len()
	if !t.size.IsInstance3(osz) {
		return false
	}

	if t.typ == anyType_DEFAULT {
		return true
	}

	for idx := 0; idx < osz; idx++ {
		if !GuardedIsInstance(t.typ, iv.At(idx), g) {
			return false
		}
	}
	return true
}

func (t *ArrayType) MetaType() eval.ObjectType {
	return Array_Type
}

func (t *ArrayType) Name() string {
	return `Array`
}

func (t *ArrayType) Size() *IntegerType {
	return t.size
}

func (t *ArrayType) String() string {
	return eval.ToString2(t, NONE)
}

func (t *ArrayType) Type() eval.PType {
	return &TypeType{t}
}

func writeTypes(bld io.Writer, format eval.FormatContext, g eval.RDetect, types ...eval.PType) bool {
	top := len(types)
	if top == 0 {
		return false
	}
	types[0].ToString(bld, format, g)
	for idx := 1; idx < top; idx++ {
		utils.WriteByte(bld, ',')
		types[idx].ToString(bld, format, g)
	}
	return true
}

func writeRange(bld io.Writer, t *IntegerType, needComma bool, skipDefault bool) bool {
	if skipDefault && *t == *integerType_POSITIVE {
		return false
	}
	if needComma {
		utils.WriteByte(bld, ',')
	}
	fmt.Fprintf(bld, "%d,", t.min)
	if t.max == math.MaxInt64 {
		io.WriteString(bld, `default`)
	} else {
		fmt.Fprintf(bld, "%d", t.max)
	}
	return true
}

func (t *ArrayType) Parameters() []eval.PValue {
	if t.typ.Equals(unitType_DEFAULT, nil) && *t.size == *integerType_ZERO {
		return t.size.SizeParameters()
	}

	params := make([]eval.PValue, 0)
	if !t.typ.Equals(DefaultAnyType(), nil) {
		params = append(params, t.typ)
	}
	if *t.size != *integerType_POSITIVE {
		params = append(params, t.size.SizeParameters()...)
	}
	return params
}

func (t *ArrayType) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	TypeToString(t, b, s, g)
}

var arrayType_DEFAULT = &ArrayType{integerType_POSITIVE, anyType_DEFAULT}
var arrayType_EMPTY = &ArrayType{integerType_ZERO, unitType_DEFAULT}

func SingletonArray(element eval.PValue) *ArrayValue {
	return &ArrayValue{elements: []eval.PValue{element}}
}

func WrapArray(elements []eval.PValue) *ArrayValue {
	return &ArrayValue{elements: elements}
}

func WrapArray2(elements []interface{}) *ArrayValue {
	els := make([]eval.PValue, len(elements))
	for i, e := range elements {
		els[i] = wrap(e)
	}
	return &ArrayValue{elements: els}
}

func WrapArray3(iv eval.IndexedValue) *ArrayValue {
	if ar, ok := iv.(*ArrayValue); ok {
		return ar
	}
	return WrapArray(iv.AppendTo(make([]eval.PValue, 0, iv.Len())))
}

func (av *ArrayValue) Add(ov eval.PValue) eval.IndexedValue {
	return WrapArray(append(av.elements, ov))
}

func (av *ArrayValue) AddAll(ov eval.IndexedValue) eval.IndexedValue {
	if ar, ok := ov.(*ArrayValue); ok {
		return WrapArray(append(av.elements, ar.elements...))
	}

	aLen := len(av.elements)
	sLen := aLen + ov.Len()
	el := make([]eval.PValue, sLen)
	copy(el, av.elements)
	for idx := aLen; idx < sLen; idx++ {
		el[idx] = ov.At(idx - aLen)
	}
	return WrapArray(el)
}

func (av *ArrayValue) All(predicate eval.Predicate) bool {
	for _, e := range av.elements {
		if !predicate(e) {
			return false
		}
	}
	return true
}

func (av *ArrayValue) Any(predicate eval.Predicate) bool {
	for _, e := range av.elements {
		if predicate(e) {
			return true
		}
	}
	return false
}

func (av *ArrayValue) AppendTo(slice []eval.PValue) []eval.PValue {
	return append(slice, av.elements...)
}

func (av *ArrayValue) At(i int) eval.PValue {
	if i >= 0 && i < len(av.elements) {
		return av.elements[i]
	}
	return _UNDEF
}

func (av *ArrayValue) Delete(ov eval.PValue) eval.IndexedValue {
	return av.Reject(func(elem eval.PValue) bool {
		return elem.Equals(ov, nil)
	})
}

func (av *ArrayValue) DeleteAll(ov eval.IndexedValue) eval.IndexedValue {
	return av.Reject(func(elem eval.PValue) bool {
		return ov.Any(func(oe eval.PValue) bool {
			return elem.Equals(oe, nil)
		})
	})
}

func (av *ArrayValue) DetailedType() eval.PType {
	av.lock.Lock()
	t := av.prtvDetailedType()
	av.lock.Unlock()
	return t
}

func (av *ArrayValue) Each(consumer eval.Consumer) {
	for _, e := range av.elements {
		consumer(e)
	}
}

func (av *ArrayValue) EachWithIndex(consumer eval.IndexedConsumer) {
	for i, e := range av.elements {
		consumer(e, i)
	}
}

func (av *ArrayValue) EachSlice(n int, consumer eval.SliceConsumer) {
	top := len(av.elements)
	for i := 0; i < top; i += n {
		e := i + n
		if e > top {
			e = top
		}
		consumer(WrapArray(av.elements[i:e]))
	}
}

func (av *ArrayValue) ElementType() eval.PType {
	return av.Type().(*ArrayType).ElementType()
}

func (av *ArrayValue) Equals(o interface{}, g eval.Guard) bool {
	if ov, ok := o.(*ArrayValue); ok {
		if top := len(av.elements); top == len(ov.elements) {
			for idx := 0; idx < top; idx++ {
				if !av.elements[idx].Equals(ov.elements[idx], g) {
					return false
				}
			}
			return true
		}
	}
	if len(av.elements) == 2 {
		if he, ok := o.(*HashEntry); ok {
			return av.elements[0].Equals(he.key, g) && av.elements[1].Equals(he.value, g)
		}
	}
	return false
}

func (av *ArrayValue) Find(predicate eval.Predicate) (eval.PValue, bool) {
	for _, e := range av.elements {
		if predicate(e) {
			return e, true
		}
	}
	return nil, false
}

func (av *ArrayValue) Flatten() eval.IndexedValue {
	for _, e := range av.elements {
		switch e.(type) {
		case *ArrayValue, *HashEntry:
			return WrapArray(flattenElements(av.elements, make([]eval.PValue, 0, len(av.elements) * 2)))
		}
	}
	return av
}

func flattenElements(elements, receiver []eval.PValue) []eval.PValue {
	for _, e := range elements {
		switch e.(type) {
		case *ArrayValue:
			receiver = flattenElements(e.(*ArrayValue).elements, receiver)
		case *HashEntry:
			he := e.(*HashEntry)
			receiver = flattenElements([]eval.PValue{he.key, he.value}, receiver)
		default:
			receiver = append(receiver, e)
		}
	}
	return receiver
}

func (av *ArrayValue) IsEmpty() bool {
	return len(av.elements) == 0
}

func (av *ArrayValue) IsHashStyle() bool {
	return false
}

func (av *ArrayValue) Iterator() eval.Iterator {
	return &indexedIterator{av.ElementType(), -1, av}
}

func (av *ArrayValue) Len() int {
	return len(av.elements)
}

func (av *ArrayValue) Map(mapper eval.Mapper) eval.IndexedValue {
	mapped := make([]eval.PValue, len(av.elements))
	for i, e := range av.elements {
		mapped[i] = mapper(e)
	}
	return WrapArray(mapped)
}

func (av *ArrayValue) Reduce(redactor eval.BiMapper) eval.PValue {
	if av.IsEmpty() {
		return _UNDEF
	}
	return reduceSlice(av.elements[1:], av.At(0), redactor)
}

func (av *ArrayValue) Reduce2(initialValue eval.PValue, redactor eval.BiMapper) eval.PValue {
	return reduceSlice(av.elements, initialValue, redactor)
}

func (av *ArrayValue) Reject(predicate eval.Predicate) eval.IndexedValue {
	selected := make([]eval.PValue, 0)
	for _, e := range av.elements {
		if !predicate(e) {
			selected = append(selected, e)
		}
	}
	return WrapArray(selected)
}

func (av *ArrayValue) Select(predicate eval.Predicate) eval.IndexedValue {
	selected := make([]eval.PValue, 0)
	for _, e := range av.elements {
		if predicate(e) {
			selected = append(selected, e)
		}
	}
	return WrapArray(selected)
}

func (av *ArrayValue) Slice(i int, j int) eval.IndexedValue {
	return WrapArray(av.elements[i:j])
}

func (av *ArrayValue) String() string {
	return eval.ToString2(av, NONE)
}

func (av *ArrayValue) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	av.ToString2(b, s, eval.GetFormat(s.FormatMap(), av.Type()), '[', g)
}

func (av *ArrayValue) ToString2(b io.Writer, s eval.FormatContext, f eval.Format, delim byte, g eval.RDetect) {
	switch f.FormatChar() {
	case 'a', 's', 'p':
	default:
		panic(s.UnsupportedFormat(av.Type(), `asp`, f))
	}
	indent := s.Indentation()
	indent = indent.Indenting(f.IsAlt() || indent.IsIndenting())

	if indent.Breaks() {
		io.WriteString(b, "\n")
		io.WriteString(b, indent.Padding())
	}

	var delims [2]byte
	if f.LeftDelimiter() == 0 {
		delims = delimiterPairs[delim]
	} else {
		delims = delimiterPairs[f.LeftDelimiter()]
	}
	if delims[0] != 0 {
		b.Write(delims[:1])
	}

	top := len(av.elements)
	if top > 0 {
		mapped := make([]string, top)
		arrayOrHash := make([]bool, top)
		childrenIndent := indent.Increase(f.IsAlt())

		cf := f.ContainerFormats()
		if cf == nil {
			cf = DEFAULT_CONTAINER_FORMATS
		}
		for idx, v := range av.elements {
			arrayOrHash[idx] = isArrayOrHash(v)
			mapped[idx] = childToString(v, childrenIndent.Subsequent(), s, cf, g)
		}

		szBreak := false
		if f.IsAlt() && f.Width() >= 0 {
			widest := 0
			for idx, ah := range arrayOrHash {
				if ah {
					widest = 0
				} else {
					widest += len(mapped[idx])
					if widest > f.Width() {
						szBreak = true
						break
					}
				}
			}
		}

		sep := f.Separator(`,`)
		for idx, ah := range arrayOrHash {
			if childrenIndent.IsFirst() {
				childrenIndent = childrenIndent.Subsequent()
				// if breaking, indent first element by one
				if szBreak && !ah {
					io.WriteString(b, ` `)
				}
			} else {
				io.WriteString(b, sep)
				// if break on each (and breaking will not occur because next is an array or hash)
				// or, if indenting, and previous was an array or hash, then break and continue on next line
				// indented.
				if !ah && (szBreak || f.IsAlt() && arrayOrHash[idx-1]) {
					io.WriteString(b, "\n")
					io.WriteString(b, childrenIndent.Padding())
				} else if !(f.IsAlt() && ah) {
					io.WriteString(b, ` `)
				}
			}
			io.WriteString(b, mapped[idx])
		}
	}
	if delims[1] != 0 {
		b.Write(delims[1:])
	}
}

func (av *ArrayValue) Unique() eval.IndexedValue {
	top := len(av.elements)
	if top < 2 {
		return av
	}

	result := make([]eval.PValue, 0, top)
	exists := make(map[eval.HashKey]bool, top)
	for _, v := range av.elements {
		key := eval.ToKey(v)
		if !exists[key] {
			exists[key] = true
			result = append(result, v)
		}
	}
	if len(result) == len(av.elements) {
		return av
	}
	return WrapArray(result)
}

func childToString(child eval.PValue, indent eval.Indentation, parentCtx eval.FormatContext, cf eval.FormatMap, g eval.RDetect) string {
	var childrenCtx eval.FormatContext
	if isContainer(child) {
		childrenCtx = newFormatContext2(indent, parentCtx.FormatMap())
	} else {
		childrenCtx = newFormatContext2(indent, cf)
	}
	b := bytes.NewBufferString(``)
	child.ToString(b, childrenCtx, g)
	return b.String()
}

func isContainer(child eval.PValue) bool {
	switch child.(type) {
	case *ArrayValue, *HashValue, eval.PType, eval.PuppetObject:
		return true
	default:
		return false
	}
}

func isArrayOrHash(child eval.PValue) bool {
	switch child.(type) {
	case *ArrayValue, *HashValue:
		return true
	default:
		return false
	}
}

func (av *ArrayValue) Type() eval.PType {
	av.lock.Lock()
	t := av.prtvReducedType()
	av.lock.Unlock()
	return t
}

func (av *ArrayValue) prtvDetailedType() eval.PType {
	if av.detailedType == nil {
		if len(av.elements) == 0 {
			av.detailedType = av.prtvReducedType()
		} else {
			types := make([]eval.PType, len(av.elements))
			for idx, element := range av.elements {
				types[idx] = eval.DetailedValueType(element)
			}
			av.detailedType = NewTupleType(types, nil)
		}
	}
	return av.detailedType
}

func (av *ArrayValue) prtvReducedType() eval.PType {
	if av.reducedType == nil {
		top := len(av.elements)
		if top == 0 {
			av.reducedType = EmptyArrayType()
		} else {
			elemType := av.elements[0].Type()
			for idx := 1; idx < top; idx++ {
				elemType = commonType(elemType, av.elements[idx].Type())
			}
			av.reducedType = NewArrayType(elemType, NewIntegerType(int64(top), int64(top)))
		}
	}
	return av.reducedType
}

func reduceSlice(slice []eval.PValue, initialValue eval.PValue, redactor eval.BiMapper) eval.PValue {
	memo := initialValue
	for _, v := range	slice {
		memo = redactor(memo, v)
	}
	return memo
}
