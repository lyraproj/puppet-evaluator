package types

import (
	. "fmt"
	. "io"
	. "math"

	"sync"

	"bytes"

	. "github.com/puppetlabs/go-evaluator/errors"
	. "github.com/puppetlabs/go-evaluator/evaluator"
	. "github.com/puppetlabs/go-evaluator/utils"
)

type (
	ArrayType struct {
		size *IntegerType
		typ  PType
	}

	ArrayValue struct {
		lock         sync.Mutex
		reducedType  PType
		detailedType PType
		elements     []PValue
	}
)

func DefaultArrayType() *ArrayType {
	return arrayType_DEFAULT
}

func EmptyArrayType() *ArrayType {
	return arrayType_EMPTY
}

func NewArrayType(element PType, rng *IntegerType) *ArrayType {
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

func NewArrayType2(args ...PValue) *ArrayType {
	argc := len(args)
	if argc == 0 {
		return DefaultArrayType()
	}

	offset := 0
	element, ok := args[0].(PType)
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
			rng = NewIntegerType(sz, MaxInt64)
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
			max = MaxInt64
		}
		rng = NewIntegerType(min, max)
	default:
		panic(NewIllegalArgumentCount(`Array[]`, `0 - 3`, argc))
	}
	return NewArrayType(element, rng)
}

func (t *ArrayType) ElementType() PType {
	return t.typ
}

func (t *ArrayType) Accept(v Visitor, g Guard) {
	v(t)
	t.size.Accept(v, g)
	t.typ.Accept(v, g)
}

func (t *ArrayType) Equals(o interface{}, g Guard) bool {
	if ot, ok := o.(*ArrayType); ok {
		return t.typ.Equals(ot.typ, g)
	}
	return false
}

func (t *ArrayType) Generic() PType {
	if t.typ == anyType_DEFAULT {
		return arrayType_DEFAULT
	}
	return NewArrayType(Generalize(t.typ), nil)
}

func (t *ArrayType) Default() PType {
	return arrayType_DEFAULT
}

func (t *ArrayType) IsAssignable(o PType, g Guard) bool {
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

func (t *ArrayType) IsInstance(v PValue, g Guard) bool {
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

func (t *ArrayType) Name() string {
	return `Array`
}

func (t *ArrayType) Size() *IntegerType {
	return t.size
}

func (t *ArrayType) String() string {
	return ToString2(t, NONE)
}

func (t *ArrayType) Type() PType {
	return &TypeType{t}
}

func writeTypes(bld Writer, format FormatContext, g RDetect, types ...PType) bool {
	top := len(types)
	if top == 0 {
		return false
	}
	types[0].ToString(bld, format, g)
	for idx := 1; idx < top; idx++ {
		WriteByte(bld, ',')
		types[idx].ToString(bld, format, g)
	}
	return true
}

func writeRange(bld Writer, t *IntegerType, needComma bool, skipDefault bool) bool {
	if skipDefault && *t == *integerType_POSITIVE {
		return false
	}
	if needComma {
		WriteByte(bld, ',')
	}
	Fprintf(bld, "%d,", t.min)
	if t.max == MaxInt64 {
		WriteString(bld, `default`)
	} else {
		Fprintf(bld, "%d", t.max)
	}
	return true
}

func (t *ArrayType) Parameters() []PValue {
	params := make([]PValue, 0)
	if !t.typ.Equals(DefaultAnyType(), nil) {
		params = append(params, t.typ)
	}
	if *t.size != *integerType_POSITIVE {
		params = append(params, t.size.SizeParameters()...)
	}
	return params
}

func (t *ArrayType) ToString(b Writer, s FormatContext, g RDetect) {
	TypeToString(t, b, s, g)
}

var arrayType_DEFAULT = &ArrayType{integerType_POSITIVE, anyType_DEFAULT}
var arrayType_EMPTY = &ArrayType{integerType_ZERO, unitType_DEFAULT}

func WrapArray(elements []PValue) *ArrayValue {
	return &ArrayValue{elements: elements}
}

func (av *ArrayValue) Add(ov PValue) IndexedValue {
	return WrapArray(append(av.elements, ov))
}

func (av *ArrayValue) AddAll(ov IndexedValue) IndexedValue {
	if ar, ok := ov.(*ArrayValue); ok {
		return WrapArray(append(av.elements, ar.elements...))
	}

	aLen := len(av.elements)
	sLen := aLen + ov.Len()
	el := make([]PValue, sLen)
	copy(el, av.elements)
	for idx := aLen; idx < sLen; idx++ {
		el[idx] = ov.At(idx - aLen)
	}
	return WrapArray(el)
}

func (av *ArrayValue) At(i int) PValue {
	if i >= 0 && i < len(av.elements) {
		return av.elements[i]
	}
	return _UNDEF
}

func (av *ArrayValue) Delete(ov PValue) IndexedValue {
	result := make([]PValue, 0, len(av.elements))
	for _, elem := range av.elements {
		if elem.Equals(ov, nil) {
			continue
		}
		result = append(result, elem)
	}
	return WrapArray(result)
}

func (av *ArrayValue) DeleteAll(ov IndexedValue) IndexedValue {
	result := make([]PValue, 0, len(av.elements))
	ovEls := ov.Elements()
	oLen := len(ovEls)
outer:
	for _, elem := range av.elements {
		for idx := 0; idx < oLen; idx++ {
			if elem.Equals(ovEls[idx], nil) {
				continue outer
			}
		}
		result = append(result, elem)
	}
	return WrapArray(result)
}

func (av *ArrayValue) DetailedType() PType {
	av.lock.Lock()
	defer av.lock.Unlock()
	return av.prtvDetailedType()
}

func (av *ArrayValue) Elements() []PValue {
	return av.elements
}

func (t *ArrayValue) ElementType() PType {
	return t.Type().(*ArrayType).ElementType()
}

func (av *ArrayValue) Equals(o interface{}, g Guard) bool {
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
	return false
}

func (av *ArrayValue) IsHashStyle() bool {
	return false
}

func (av *ArrayValue) Iterator() Iterator {
	return &indexedIterator{av.ElementType(), -1, av}
}

func (av *ArrayValue) Len() int {
	return len(av.elements)
}

func (av *ArrayValue) Slice(i int, j int) *ArrayValue {
	return WrapArray(av.elements[i:j])
}

func (av *ArrayValue) String() string {
	return ToString2(av, NONE)
}

func (av *ArrayValue) ToString(b Writer, s FormatContext, g RDetect) {
	f := GetFormat(s.FormatMap(), av.Type())
	switch f.FormatChar() {
	case 'a', 's', 'p':
	default:
		panic(s.UnsupportedFormat(av.Type(), `asp`, f))
	}
	indent := s.Indentation()
	indent = indent.Indenting(f.IsAlt() || indent.IsIndenting())

	if indent.Breaks() {
		WriteString(b, "\n")
		WriteString(b, indent.Padding())
	}
	delims := delimiterPairs[f.LeftDelimiter()]
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
					WriteString(b, ` `)
				}
			} else {
				WriteString(b, sep)
				// if break on each (and breaking will not occur because next is an array or hash)
				// or, if indenting, and previous was an array or hash, then break and continue on next line
				// indented.
				if !ah && (szBreak || f.IsAlt() && arrayOrHash[idx-1]) {
					WriteString(b, "\n")
					WriteString(b, childrenIndent.Padding())
				} else if !(f.IsAlt() && ah) {
					WriteString(b, ` `)
				}
			}
			WriteString(b, mapped[idx])
		}
	}
	if delims[1] != 0 {
		b.Write(delims[1:])
	}
}

func childToString(child PValue, indent Indentation, parentCtx FormatContext, cf FormatMap, g RDetect) string {
	var childrenCtx FormatContext
	if isContainer(child) {
		childrenCtx = newFormatContext2(indent, parentCtx.FormatMap())
	} else {
		childrenCtx = newFormatContext2(indent, cf)
	}
	b := bytes.NewBufferString(``)
	child.ToString(b, childrenCtx, g)
	return b.String()
}

func isContainer(child PValue) bool {
	switch child.(type) {
	case *ArrayValue, *HashValue, PuppetObject:
		return true
	default:
		return false
	}
}

func isArrayOrHash(child PValue) bool {
	switch child.(type) {
	case *ArrayValue, *HashValue:
		return true
	default:
		return false
	}
}

func (av *ArrayValue) Type() PType {
	av.lock.Lock()
	defer av.lock.Unlock()
	return av.prtvReducedType()
}

func (av *ArrayValue) prtvDetailedType() PType {
	if av.detailedType == nil {
		if len(av.elements) == 0 {
			av.detailedType = av.prtvReducedType()
		} else {
			types := make([]PType, len(av.elements))
			for idx, element := range av.elements {
				types[idx] = element.Type()
			}
			av.detailedType = NewTupleType(types, nil)
		}
	}
	return av.detailedType
}

func (av *ArrayValue) prtvReducedType() PType {
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
