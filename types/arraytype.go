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
	"reflect"
	"sort"
)

type (
	ArrayType struct {
		size *IntegerType
		typ  eval.Type
	}

	ArrayValue struct {
		lock         sync.Mutex
		reducedType  eval.Type
		detailedType eval.Type
		elements     []eval.Value
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
}`, func(ctx eval.Context, args []eval.Value) eval.Value {
			return NewArrayType2(args...)
		})
}

func DefaultArrayType() *ArrayType {
	return arrayType_DEFAULT
}

func EmptyArrayType() *ArrayType {
	return arrayType_EMPTY
}

func NewArrayType(element eval.Type, rng *IntegerType) *ArrayType {
	if element == nil {
		element = anyType_DEFAULT
	}
	if rng == nil {
		rng = IntegerType_POSITIVE
	}
	if *rng == *IntegerType_POSITIVE && element == anyType_DEFAULT {
		return DefaultArrayType()
	}
	if *rng == *IntegerType_ZERO && element == unitType_DEFAULT {
		return EmptyArrayType()
	}
	return &ArrayType{rng, element}
}

func NewArrayType2(args ...eval.Value) *ArrayType {
	argc := len(args)
	if argc == 0 {
		return DefaultArrayType()
	}

	offset := 0
	element, ok := args[0].(eval.Type)
	if ok {
		offset++
	} else {
		element = DefaultAnyType()
	}

	var rng *IntegerType
	switch argc - offset {
	case 0:
		rng = IntegerType_POSITIVE
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

func (t *ArrayType) ElementType() eval.Type {
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

func (t *ArrayType) Generic() eval.Type {
	if t.typ == anyType_DEFAULT {
		return arrayType_DEFAULT
	}
	return NewArrayType(eval.Generalize(t.typ), nil)
}

func (t *ArrayType) Get(key string) (value eval.Value, ok bool) {
	switch key {
	case `element_type`:
		return t.typ, true
	case `size_type`:
		return t.size, true
	}
	return nil, false
}

func (t *ArrayType) Default() eval.Type {
	return arrayType_DEFAULT
}

func (t *ArrayType) IsAssignable(o eval.Type, g eval.Guard) bool {
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

func (t *ArrayType) IsInstance(v eval.Value, g eval.Guard) bool {
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

func (t *ArrayType) Resolve(c eval.Context) eval.Type {
	t.typ = resolve(c, t.typ)
	return t
}

func (t *ArrayType) Size() *IntegerType {
	return t.size
}

func (t *ArrayType) String() string {
	return eval.ToString2(t, NONE)
}

func (t *ArrayType) PType() eval.Type {
	return &TypeType{t}
}

func writeTypes(bld io.Writer, format eval.FormatContext, g eval.RDetect, types ...eval.Type) bool {
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
	if skipDefault && *t == *IntegerType_POSITIVE {
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

func (t *ArrayType) Parameters() []eval.Value {
	if t.typ.Equals(unitType_DEFAULT, nil) && *t.size == *IntegerType_ZERO {
		return t.size.SizeParameters()
	}

	params := make([]eval.Value, 0)
	if !t.typ.Equals(DefaultAnyType(), nil) {
		params = append(params, t.typ)
	}
	if *t.size != *IntegerType_POSITIVE {
		params = append(params, t.size.SizeParameters()...)
	}
	return params
}

func (t *ArrayType) ReflectType(c eval.Context) (reflect.Type, bool) {
	if et, ok := ReflectType(c, t.ElementType()); ok {
		return reflect.SliceOf(et), true
	}
	return nil, false
}

func (t *ArrayType) CanSerializeAsString() bool {
	ts, ok := t.typ.(eval.SerializeAsString)
	return ok && ts.CanSerializeAsString()
}

func (t *ArrayType) SerializationString() string {
	return t.String()
}

func (t *ArrayType) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	TypeToString(t, b, s, g)
}

var arrayType_DEFAULT = &ArrayType{IntegerType_POSITIVE, anyType_DEFAULT}
var arrayType_EMPTY = &ArrayType{IntegerType_ZERO, unitType_DEFAULT}

func SingletonArray(element eval.Value) *ArrayValue {
	return &ArrayValue{elements: []eval.Value{element}}
}

func WrapTypes(elements []eval.Type) *ArrayValue {
	els := make([]eval.Value, len(elements))
	for i, e := range elements {
		els[i] = e
	}
	return &ArrayValue{elements: els}
}

func WrapValues(elements []eval.Value) *ArrayValue {
	return &ArrayValue{elements: elements}
}

func WrapInterfaces(c eval.Context, elements []interface{}) *ArrayValue {
	els := make([]eval.Value, len(elements))
	for i, e := range elements {
		els[i] = wrap(c, e)
	}
	return &ArrayValue{elements: els}
}

func WrapInts(ints []int) *ArrayValue {
	els := make([]eval.Value, len(ints))
	for i, e := range ints {
		els[i] = WrapInteger(int64(e))
	}
	return &ArrayValue{elements: els}
}

func WrapStrings(strings []string) *ArrayValue {
	els := make([]eval.Value, len(strings))
	for i, e := range strings {
		els[i] = WrapString(e)
	}
	return &ArrayValue{elements: els}
}

func WrapArray3(iv eval.List) *ArrayValue {
	if ar, ok := iv.(*ArrayValue); ok {
		return ar
	}
	return WrapValues(iv.AppendTo(make([]eval.Value, 0, iv.Len())))
}

func (av *ArrayValue) Add(ov eval.Value) eval.List {
	return WrapValues(append(av.elements, ov))
}

func (av *ArrayValue) AddAll(ov eval.List) eval.List {
	if ar, ok := ov.(*ArrayValue); ok {
		return WrapValues(append(av.elements, ar.elements...))
	}

	aLen := len(av.elements)
	sLen := aLen + ov.Len()
	el := make([]eval.Value, sLen)
	copy(el, av.elements)
	for idx := aLen; idx < sLen; idx++ {
		el[idx] = ov.At(idx - aLen)
	}
	return WrapValues(el)
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

func (av *ArrayValue) AppendTo(slice []eval.Value) []eval.Value {
	return append(slice, av.elements...)
}

func (av *ArrayValue) At(i int) eval.Value {
	if i >= 0 && i < len(av.elements) {
		return av.elements[i]
	}
	return _UNDEF
}

func (av *ArrayValue) Delete(ov eval.Value) eval.List {
	return av.Reject(func(elem eval.Value) bool {
		return elem.Equals(ov, nil)
	})
}

func (av *ArrayValue) DeleteAll(ov eval.List) eval.List {
	return av.Reject(func(elem eval.Value) bool {
		return ov.Any(func(oe eval.Value) bool {
			return elem.Equals(oe, nil)
		})
	})
}

func (av *ArrayValue) DetailedType() eval.Type {
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
		consumer(WrapValues(av.elements[i:e]))
	}
}

func (av *ArrayValue) ElementType() eval.Type {
	return av.PType().(*ArrayType).ElementType()
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

func (av *ArrayValue) Find(predicate eval.Predicate) (eval.Value, bool) {
	for _, e := range av.elements {
		if predicate(e) {
			return e, true
		}
	}
	return nil, false
}

func (av *ArrayValue) Flatten() eval.List {
	for _, e := range av.elements {
		switch e.(type) {
		case *ArrayValue, *HashEntry:
			return WrapValues(flattenElements(av.elements, make([]eval.Value, 0, len(av.elements)*2)))
		}
	}
	return av
}

func flattenElements(elements, receiver []eval.Value) []eval.Value {
	for _, e := range elements {
		switch e.(type) {
		case *ArrayValue:
			receiver = flattenElements(e.(*ArrayValue).elements, receiver)
		case *HashEntry:
			he := e.(*HashEntry)
			receiver = flattenElements([]eval.Value{he.key, he.value}, receiver)
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

func (av *ArrayValue) Map(mapper eval.Mapper) eval.List {
	mapped := make([]eval.Value, len(av.elements))
	for i, e := range av.elements {
		mapped[i] = mapper(e)
	}
	return WrapValues(mapped)
}

func (av *ArrayValue) Reduce(redactor eval.BiMapper) eval.Value {
	if av.IsEmpty() {
		return _UNDEF
	}
	return reduceSlice(av.elements[1:], av.At(0), redactor)
}

func (av *ArrayValue) Reduce2(initialValue eval.Value, redactor eval.BiMapper) eval.Value {
	return reduceSlice(av.elements, initialValue, redactor)
}

func (av *ArrayValue) Reflect(c eval.Context) reflect.Value {
	at, ok := ReflectType(c, av.PType())
	if !ok {
		at = reflect.TypeOf([]interface{}{})
	}
	s := reflect.MakeSlice(at, av.Len(), av.Len())
	rf := c.Reflector()
	for i, e := range av.elements {
		rf.ReflectTo(e, s.Index(i))
	}
	return s
}

func (av *ArrayValue) ReflectTo(c eval.Context, value reflect.Value) {
	s := reflect.MakeSlice(value.Type(), av.Len(), av.Len())
	rf := c.Reflector()
	for i, e := range av.elements {
		rf.ReflectTo(e, s.Index(i))
	}
	value.Set(s)
}

func (av *ArrayValue) Reject(predicate eval.Predicate) eval.List {
	all := true
	selected := make([]eval.Value, 0)
	for _, e := range av.elements {
		if !predicate(e) {
			selected = append(selected, e)
		} else {
			all = false
		}
	}
	if all {
		return av
	}
	return WrapValues(selected)
}

func (av *ArrayValue) Select(predicate eval.Predicate) eval.List {
	all := true
	selected := make([]eval.Value, 0)
	for _, e := range av.elements {
		if predicate(e) {
			selected = append(selected, e)
		} else {
			all = false
		}
	}
	if all {
		return av
	}
	return WrapValues(selected)
}

func (av *ArrayValue) Slice(i int, j int) eval.List {
	return WrapValues(av.elements[i:j])
}

type arraySorter struct {
	values     []eval.Value
	comparator eval.Comparator
}

func (s *arraySorter) Len() int {
	return len(s.values)
}

func (s *arraySorter) Less(i, j int) bool {
	vs := s.values
	return s.comparator(vs[i], vs[j])
}

func (s *arraySorter) Swap(i, j int) {
	vs := s.values
	v := vs[i]
	vs[i] = vs[j]
	vs[j] = v
}

func (av *ArrayValue) Sort(comparator eval.Comparator) eval.List {
	s := &arraySorter{make([]eval.Value, len(av.elements)), comparator}
	copy(s.values, av.elements)
	sort.Sort(s)
	return WrapValues(s.values)
}

func (av *ArrayValue) String() string {
	return eval.ToString2(av, NONE)
}

func (av *ArrayValue) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	av.ToString2(b, s, eval.GetFormat(s.FormatMap(), av.PType()), '[', g)
}

func (av *ArrayValue) ToString2(b io.Writer, s eval.FormatContext, f eval.Format, delim byte, g eval.RDetect) {
	switch f.FormatChar() {
	case 'a', 's', 'p':
	default:
		panic(s.UnsupportedFormat(av.PType(), `asp`, f))
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
			arrayOrHash[idx] = isContainer(v, s)
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

func (av *ArrayValue) Unique() eval.List {
	top := len(av.elements)
	if top < 2 {
		return av
	}

	result := make([]eval.Value, 0, top)
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
	return WrapValues(result)
}

func childToString(child eval.Value, indent eval.Indentation, parentCtx eval.FormatContext, cf eval.FormatMap, g eval.RDetect) string {
	var childrenCtx eval.FormatContext
	if isContainer(child, parentCtx) {
		childrenCtx = newFormatContext2(indent, parentCtx.FormatMap(), parentCtx.Properties())
	} else {
		childrenCtx = newFormatContext2(indent, cf, parentCtx.Properties())
	}
	b := bytes.NewBufferString(``)
	child.ToString(b, childrenCtx, g)
	return b.String()
}

func isContainer(child eval.Value, s eval.FormatContext) bool {
	switch child.(type) {
	case *ArrayValue, *HashValue:
		return true
	case eval.ObjectType, eval.TypeSet:
		if ex, ok := s.Property(`expanded`); ok && ex == `true` {
			return true
		}
		return false
	case eval.PuppetObject:
		return true
	default:
		return false
	}
}

func (av *ArrayValue) PType() eval.Type {
	av.lock.Lock()
	t := av.prtvReducedType()
	av.lock.Unlock()
	return t
}

func (av *ArrayValue) prtvDetailedType() eval.Type {
	if av.detailedType == nil {
		if len(av.elements) == 0 {
			av.detailedType = av.prtvReducedType()
		} else {
			types := make([]eval.Type, len(av.elements))
			for idx, element := range av.elements {
				types[idx] = eval.DetailedValueType(element)
			}
			av.detailedType = NewTupleType(types, nil)
		}
	}
	return av.detailedType
}

func (av *ArrayValue) prtvReducedType() eval.Type {
	if av.reducedType == nil {
		top := len(av.elements)
		if top == 0 {
			av.reducedType = EmptyArrayType()
		} else {
			elemType := av.elements[0].PType()
			for idx := 1; idx < top; idx++ {
				elemType = commonType(elemType, av.elements[idx].PType())
			}
			av.reducedType = NewArrayType(elemType, NewIntegerType(int64(top), int64(top)))
		}
	}
	return av.reducedType
}

func reduceSlice(slice []eval.Value, initialValue eval.Value, redactor eval.BiMapper) eval.Value {
	memo := initialValue
	for _, v := range slice {
		memo = redactor(memo, v)
	}
	return memo
}
