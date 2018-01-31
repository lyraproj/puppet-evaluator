package types

import (
	"fmt"
	. "io"
	"math"
	"sync"

	. "github.com/puppetlabs/go-evaluator/errors"
	. "github.com/puppetlabs/go-evaluator/evaluator"
	"github.com/puppetlabs/go-evaluator/hash"
	"sort"
)

type (
	HashType struct {
		size      *IntegerType
		keyType   PType
		valueType PType
	}

	HashEntry struct {
		key   PValue
		value PValue
	}

	HashValue struct {
		lock         sync.Mutex
		reducedType  PType
		detailedType PType
		entries      []*HashEntry
		index        map[HashKey]int
	}

	MutableHashValue struct {
		HashValue
	}
)

var hashType_EMPTY = &HashType{integerType_ZERO, unitType_DEFAULT, unitType_DEFAULT}
var hashType_DEFAULT = &HashType{integerType_POSITIVE, anyType_DEFAULT, anyType_DEFAULT}

func DefaultHashType() *HashType {
	return hashType_DEFAULT
}

func EmptyHashType() *HashType {
	return hashType_EMPTY
}

func NewHashType(keyType PType, valueType PType, rng *IntegerType) *HashType {
	if rng == nil {
		rng = integerType_POSITIVE
	}
	if keyType == nil {
		keyType = anyType_DEFAULT
	}
	if valueType == nil {
		valueType = anyType_DEFAULT
	}
	if keyType == anyType_DEFAULT && valueType == anyType_DEFAULT && rng == integerType_POSITIVE {
		return DefaultHashType()
	}
	return &HashType{rng, keyType, valueType}
}

func NewHashType2(args ...PValue) *HashType {
	argc := len(args)
	if argc == 0 {
		return hashType_DEFAULT
	}

	if argc == 1 || argc > 4 {
		panic(NewIllegalArgumentCount(`Hash[]`, `0, 2, or 3`, argc))
	}

	offset := 0
	var valueType PType
	keyType, ok := args[0].(PType)
	if ok {
		valueType, ok = args[1].(PType)
		if !ok {
			panic(NewIllegalArgumentType2(`Hash[]`, 1, `Type`, args[1]))
		}
		offset += 2
	} else {
		keyType = DefaultAnyType()
		valueType = DefaultAnyType()
	}

	var rng *IntegerType
	switch argc - offset {
	case 0:
		rng = integerType_POSITIVE
	case 1:
		sizeArg := args[offset]
		if rng, ok = sizeArg.(*IntegerType); !ok {
			var sz int64
			if sz, ok = toInt(sizeArg); !ok {
				panic(NewIllegalArgumentType2(`Hash[]`, offset, `Integer or Type[Integer]`, args[2]))
			}
			rng = NewIntegerType(sz, math.MaxInt64)
		}
	case 2:
		var min, max int64
		if min, ok = toInt(args[offset]); !ok {
			panic(NewIllegalArgumentType2(`Hash[]`, offset, `Integer`, args[offset]))
		}
		if max, ok = toInt(args[offset+1]); !ok {
			panic(NewIllegalArgumentType2(`Hash[]`, offset+1, `Integer`, args[offset+1]))
		}
		if min == 0 && max == 0 && offset == 0 {
			return hashType_EMPTY
		}
		rng = NewIntegerType(min, max)
	}
	return NewHashType(keyType, valueType, rng)
}

func (t *HashType) Accept(v Visitor, g Guard) {
	v(t)
	t.size.Accept(v, g)
	t.keyType.Accept(v, g)
	t.valueType.Accept(v, g)
}

func (t *HashType) Default() PType {
	return hashType_DEFAULT
}

func (t *HashType) EntryType() PType {
	return NewTupleType([]PType{t.keyType, t.valueType}, nil)
}

func (t *HashType) Equals(o interface{}, g Guard) bool {
	if ot, ok := o.(*HashType); ok {
		return t.size.Equals(ot.size, g) && t.keyType.Equals(ot.keyType, g) && t.valueType.Equals(ot.valueType, g)
	}
	return false
}

func (t *HashType) Generic() PType {
	return NewHashType(GenericType(t.keyType), GenericType(t.valueType), nil)
}

func (t *HashType) IsAssignable(o PType, g Guard) bool {
	switch o.(type) {
	case *HashType:
		if t.size.min == 0 && o == hashType_EMPTY {
			return true
		}
		ht := o.(*HashType)
		return t.size.IsAssignable(ht.size, g) && GuardedIsAssignable(t.keyType, ht.keyType, g) && GuardedIsAssignable(t.valueType, ht.valueType, g)
	case *StructType:
		st := o.(*StructType)
		if !t.size.IsInstance3(len(st.elements)) {
			return false
		}
		for _, element := range st.elements {
			if !(GuardedIsAssignable(t.keyType, element.ActualKeyType(), g) && GuardedIsAssignable(t.valueType, element.value, g)) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func (t *HashType) IsInstance(o PValue, g Guard) bool {
	if v, ok := o.(*HashValue); ok && t.size.IsInstance3(v.Len()) {
		for _, entry := range v.entries {
			if !(GuardedIsInstance(t.keyType, entry.key, g) && GuardedIsInstance(t.valueType, entry.value, g)) {
				return false
			}
		}
		return true
	}
	return false
}

func (t *HashType) KeyType() PType {
	return t.keyType
}

func (t *HashType) Name() string {
	return `Hash`
}

func (t *HashType) Parameters() []PValue {
	if *t == *hashType_DEFAULT {
		return EMPTY_VALUES
	}
	if *t == *hashType_EMPTY {
		return []PValue{ZERO, ZERO}
	}
	params := make([]PValue, 0, 4)
	params = append(params, t.keyType)
	params = append(params, t.valueType)
	if *t.size != *integerType_POSITIVE {
		params = append(params, t.size.SizeParameters()...)
	}
	return params
}

func (t *HashType) Size() *IntegerType {
	return t.size
}

func (t *HashType) String() string {
	return ToString2(t, NONE)
}

func (t *HashType) ValueType() PType {
	return t.valueType
}

func (t *HashType) ToString(b Writer, s FormatContext, g RDetect) {
	TypeToString(t, b, s, g)
}

func (t *HashType) Type() PType {
	return &TypeType{t}
}

func WrapHashEntry(key PValue, value PValue) *HashEntry {
	return &HashEntry{key, value}
}

func WrapHashEntry2(key string, value PValue) *HashEntry {
	return &HashEntry{WrapString(key), value}
}

func (he *HashEntry) Add(v PValue) IndexedValue {
	panic(`Operation not supported`)
}

func (he *HashEntry) AddAll(v IndexedValue) IndexedValue {
	panic(`Operation not supported`)
}

func (he *HashEntry) All(predicate Predicate) bool {
	return predicate(he.key) && predicate(he.value)
}

func (he *HashEntry) Any(predicate Predicate) bool {
	return predicate(he.key) || predicate(he.value)
}

func (he *HashEntry) At(i int) PValue {
	switch i {
	case 0:
		return he.key
	case 1:
		return he.value
	default:
		return _UNDEF
	}
}

func (he *HashEntry) Delete(v PValue) IndexedValue {
	panic(`Operation not supported`)
}

func (he *HashEntry) DeleteAll(v IndexedValue) IndexedValue {
	panic(`Operation not supported`)
}

func (he *HashEntry) DetailedType() PType {
	return NewTupleType([]PType{DetailedValueType(he.key), DetailedValueType(he.value)}, NewIntegerType(2, 2))
}

func (he *HashEntry) Each(consumer Consumer) {
	consumer(he.key)
	consumer(he.value)
}

func (he *HashEntry) EachWithIndex(consumer IndexedConsumer) {
	consumer(he.key, 0)
	consumer(he.value, 1)
}

func (he *HashEntry) Elements() []PValue {
	return []PValue{he.key, he.value}
}

func (he *HashEntry) ElementType() PType {
	return commonType(he.key.Type(), he.value.Type())
}

func (he *HashEntry) Equals(o interface{}, g Guard) bool {
	if ov, ok := o.(*HashEntry); ok {
		return he.key.Equals(ov.key, g) && he.value.Equals(ov.value, g)
	}
	if iv, ok := o.(*ArrayValue); ok && iv.Len() == 2 {
		return he.key.Equals(iv.At(0), g) && he.value.Equals(iv.At(1), g)
	}
	return false
}

func (he *HashEntry) IsEmpty() bool {
	return false
}

func (he *HashEntry) IsHashStyle() bool {
	return false
}

func (he *HashEntry) Iterator() Iterator {
	return &indexedIterator{he.ElementType(), -1, he}
}

func (he *HashEntry) Key() PValue {
	return he.key
}

func (he *HashEntry) Len() int {
	return 2
}

func (he *HashEntry) Select(predicate Predicate) IndexedValue {
	if predicate(he.key) {
		if predicate(he.value) {
			return he
		}
		return WrapArray([]PValue{he.key})
	}
	if predicate(he.value) {
		return WrapArray([]PValue{he.value})
	}
	return EMPTY_ARRAY
}

func (he *HashEntry) Slice(i int, j int) IndexedValue {
	if i > 1 || i >= j {
		return EMPTY_ARRAY
	}
	if i == 1 {
		return WrapArray([]PValue{he.value})
	}
	if j == 1 {
		return WrapArray([]PValue{he.key})
	}
	return he
}

func (he *HashEntry) Reject(predicate Predicate) IndexedValue {
	if predicate(he.key) {
		if predicate(he.value) {
			return EMPTY_ARRAY
		}
		return WrapArray([]PValue{he.value})
	}
	if predicate(he.value) {
		return WrapArray([]PValue{he.key})
	}
	return he
}

func (he *HashEntry) Type() PType {
	return NewArrayType(commonType(he.key.Type(), he.value.Type()), NewIntegerType(2, 2))
}

func (he *HashEntry) Value() PValue {
	return he.value
}

func (he *HashEntry) String() string {
	return ToString2(he, NONE)
}

func (he *HashEntry) ToString(b Writer, s FormatContext, g RDetect) {
	WrapArray([]PValue{he.key, he.value}).ToString(b, s, g)
}

func WrapHash(entries []*HashEntry) *HashValue {
	return &HashValue{entries: entries}
}

func WrapHash2(entries []PValue) *HashValue {
	hvEntries := make([]*HashEntry, len(entries))
	for idx, entry := range entries {
		hvEntries[idx] = entry.(*HashEntry)
	}
	return &HashValue{entries: hvEntries}
}

// This wrap variant does not preserve order since order is undefined in a Go map
func WrapHash3(hash map[string]PValue) *HashValue {
	hvEntries := make([]*HashEntry, len(hash))
	i := 0
	for k, v := range hash {
		hvEntries[i] = WrapHashEntry(WrapString(k), v)
		i++
	}

	// map order is undefined (and changes from one run to another) so entries must
	// be sorted to get a predictable order
	sort.Slice(hvEntries, func(i, j int) bool {
		return hvEntries[i].key.String() < hvEntries[j].key.String()
	})
	return &HashValue{entries: hvEntries}
}

// This wrap variant does not preserve order since order is undefined in a Go map
func WrapHash4(hash map[string]interface{}) *HashValue {
	hvEntries := make([]*HashEntry, len(hash))
	i := 0
	for k, v := range hash {
		hvEntries[i] = WrapHashEntry(WrapString(k), wrap(v))
		i++
	}

	// map order is undefined (and changes from one run to another) so entries must
	// be sorted to get a predictable order
	sort.Slice(hvEntries, func(i, j int) bool {
		return hvEntries[i].key.String() < hvEntries[j].key.String()
	})
	return &HashValue{entries: hvEntries}
}

// This wrap variant does not preserve order since order is undefined in a Go map
func WrapHash5(hash *hash.StringHash) *HashValue {
	hvEntries := make([]*HashEntry, hash.Size())
	i := 0
	hash.EachPair(func(k string, v interface{}) {
		hvEntries[i] = WrapHashEntry(WrapString(k), wrap(v))
		i++
	})
	return &HashValue{entries: hvEntries}
}

func WrapHashFromArray(a *ArrayValue) *HashValue {
	top := a.Len()
	switch a.Type().(*ArrayType).ElementType().(type) {
	case *ArrayType:
		// Array of arrays. Assume that each nested array is [key, value]
		entries := make([]*HashEntry, top)
		for idx, pair := range a.Elements() {
			pairArr := pair.(*ArrayValue)
			if pairArr.Len() != 2 {
				panic(NewArgumentsError(`Hash`, fmt.Sprintf(`hash entry array must have 2 elements, got %d`, pairArr.Len())))
			}
			entries[idx] = WrapHashEntry(pairArr.At(0), pairArr.At(1))
		}
		return WrapHash(entries)
	default:
		if (top % 2) != 0 {
			panic(NewArgumentsError(`Hash`, `odd number of arguments in Array`))
		}
		entries := make([]*HashEntry, top/2)
		elems := a.Elements()
		for idx := 0; idx < top; idx += 2 {
			entries[idx/2] = WrapHashEntry(elems[idx], elems[idx+1])
		}
		return WrapHash(entries)
	}
}

func SingletonHash(key, value PValue) *HashValue {
	return &HashValue{entries: []*HashEntry{WrapHashEntry(key, value)}}
}

func SingletonHash2(key string, value PValue) *HashValue {
	return &HashValue{entries: []*HashEntry{WrapHashEntry2(key, value)}}
}

func (hv *HashValue) Add(v PValue) IndexedValue {
	switch v.(type) {
	case *HashEntry:
		return hv.Merge(WrapHash([]*HashEntry{v.(*HashEntry)}))
	case *ArrayValue:
		a := v.(*ArrayValue)
		if a.Len() == 2 {
			return hv.Merge(WrapHash([]*HashEntry{WrapHashEntry(a.At(0), a.At(1))}))
		}
	}
	panic(`Operation not supported`)
}

func (hv *HashValue) AddAll(v IndexedValue) IndexedValue {
	switch v.(type) {
	case *HashValue:
		return hv.Merge(v.(*HashValue))
	case *ArrayValue:
		return hv.Merge(WrapHashFromArray(v.(*ArrayValue)))
	}
	panic(`Operation not supported`)
}

func (hv *HashValue) All(predicate Predicate) bool {
	for _, e := range hv.entries {
		if !predicate(e) {
			return false
		}
	}
	return true
}

func (hv *HashValue) AllPairs(predicate BiPredicate) bool {
	for _, e := range hv.entries {
		if !predicate(e.key, e.value) {
			return false
		}
	}
	return true
}

func (hv *HashValue) Any(predicate Predicate) bool {
	for _, e := range hv.entries {
		if predicate(e) {
			return true
		}
	}
	return false
}

func (hv *HashValue) AnyPair(predicate BiPredicate) bool {
	for _, e := range hv.entries {
		if predicate(e.key, e.value) {
			return true
		}
	}
	return false
}

func (hv *HashValue) At(i int) PValue {
	if i >= 0 && i < len(hv.entries) {
		return hv.entries[i]
	}
	return _UNDEF
}

func (hv *HashValue) Delete(key PValue) IndexedValue {
	if idx, ok := hv.valueIndex()[ToKey(key)]; ok {
		return WrapHash(append(hv.entries[:idx], hv.entries[idx+1:]...))
	}
	return hv
}

func (hv *HashValue) DeleteAll(keys IndexedValue) IndexedValue {
	entries := hv.entries
	valueIndex := hv.valueIndex()
	for _, key := range keys.Elements() {
		if idx, ok := valueIndex[ToKey(key)]; ok {
			entries = append(hv.entries[:idx], hv.entries[idx+1:]...)
		}
	}
	if len(hv.entries) == len(entries) {
		return hv
	}
	return WrapHash(entries)
}

func (hv *HashValue) DetailedType() PType {
	hv.lock.Lock()
	t := hv.prtvDetailedType()
	hv.lock.Unlock()
	return t
}

func (hv *HashValue) Elements() []PValue {
	elements := make([]PValue, len(hv.entries))
	for idx, entry := range hv.entries {
		elements[idx] = WrapArray(entry.Elements())
	}
	return elements
}

func (hv *HashValue) ElementType() PType {
	return hv.Type().(*HashType).EntryType()
}

func (hv *HashValue) Entries() IndexedValue {
	return hv
}

func (hv *HashValue) EntriesSlice() []*HashEntry {
	return hv.entries
}

func (hv *HashValue) Each(consumer Consumer) {
	for _, e := range hv.entries {
		consumer(e)
	}
}

func (hv *HashValue) EachKey(consumer Consumer) {
	for _, e := range hv.entries {
		consumer(e.key)
	}
}

func (hv *HashValue) Select(predicate Predicate) IndexedValue {
	selected := make([]*HashEntry, 0)
	for _, e := range hv.entries {
		if predicate(e) {
			selected = append(selected, e)
		}
	}
	return WrapHash(selected)
}

func (hv *HashValue) SelectPairs(predicate BiPredicate) KeyedValue {
	selected := make([]*HashEntry, 0)
	for _, e := range hv.entries {
		if predicate(e.key, e.value) {
			selected = append(selected, e)
		}
	}
	return WrapHash(selected)
}

func (hv *HashValue) Reject(predicate Predicate) IndexedValue {
	selected := make([]*HashEntry, 0)
	for _, e := range hv.entries {
		if !predicate(e) {
			selected = append(selected, e)
		}
	}
	return WrapHash(selected)
}

func (hv *HashValue) RejectPairs(predicate BiPredicate) KeyedValue {
	selected := make([]*HashEntry, 0)
	for _, e := range hv.entries {
		if !predicate(e.key, e.value) {
			selected = append(selected, e)
		}
	}
	return WrapHash(selected)
}

func (hv *HashValue) EachPair(consumer BiConsumer) {
	for _, e := range hv.entries {
		consumer(e.key, e.value)
	}
}

func (hv *HashValue) EachValue(consumer Consumer) {
	for _, e := range hv.entries {
		consumer(e.value)
	}
}

func (hv *HashValue) EachWithIndex(consumer IndexedConsumer) {
	for i, e := range hv.entries {
		consumer(e, i)
	}
}

func (hv *HashValue) Equals(o interface{}, g Guard) bool {
	if ov, ok := o.(*HashValue); ok {
		if top := len(hv.entries); top == len(ov.entries) {
			ovIndex := ov.valueIndex()
			for key, idx := range hv.valueIndex() {
				var ovIdx int
				if ovIdx, ok = ovIndex[key]; !(ok && hv.entries[idx].Equals(ov.entries[ovIdx], g)) {
					return false
				}
			}
			return true
		}
	}
	return false
}

func (hv *HashValue) Get(key PValue) (PValue, bool) {
	return hv.get(ToKey(key))
}

func (hv *HashValue) Get2(key PValue, dflt PValue) PValue {
	return hv.get2(ToKey(key), dflt)
}

func (hv *HashValue) Get3(key PValue, dflt Producer) PValue {
	return hv.get3(ToKey(key), dflt)
}

func (hv *HashValue) Get4(key string) (PValue, bool) {
	return hv.get(HashKey(key))
}

func (hv *HashValue) Get5(key string, dflt PValue) PValue {
	return hv.get2(HashKey(key), dflt)
}

func (hv *HashValue) Get6(key string, dflt Producer) PValue {
	return hv.get3(HashKey(key), dflt)
}

func (hv *HashValue) get(key HashKey) (PValue, bool) {
	if pos, ok := hv.valueIndex()[key]; ok {
		return hv.entries[pos].value, true
	}
	return _UNDEF, false
}

func (hv *HashValue) get2(key HashKey, dflt PValue) PValue {
	if pos, ok := hv.valueIndex()[key]; ok {
		return hv.entries[pos].value
	}
	return dflt
}

func (hv *HashValue) get3(key HashKey, dflt Producer) PValue {
	if pos, ok := hv.valueIndex()[key]; ok {
		return hv.entries[pos].value
	}
	return dflt()
}

func (hv *HashValue) IncludesKey(o PValue) bool {
	_, ok := hv.valueIndex()[ToKey(o)]
	return ok
}

func (hv *HashValue) IncludesKey2(key string) bool {
	_, ok := hv.valueIndex()[HashKey(key)]
	return ok
}

func (hv *HashValue) IsEmpty() bool {
	return len(hv.entries) == 0
}

func (hv *HashValue) IsHashStyle() bool {
	return true
}

func (hv *HashValue) Iterator() Iterator {
	t := hv.Type().(*HashType)
	et := NewTupleType([]PType{t.KeyType(), t.ValueType()}, NewIntegerType(2, 2))
	return &indexedIterator{et, -1, hv}
}

func (hv *HashValue) Keys() IndexedValue {
	keys := make([]PValue, len(hv.entries))
	for idx, entry := range hv.entries {
		keys[idx] = entry.key
	}
	return WrapArray(keys)
}

func (hv *HashValue) Len() int {
	return len(hv.entries)
}

func (hv *HashValue) Merge(o KeyedValue) KeyedValue {
	return WrapHash(hv.mergeEntries(o))
}

func (hv *HashValue) mergeEntries(o KeyedValue) []*HashEntry {
	oh := o.(*HashValue)
	index := hv.valueIndex()
	selfLen := len(hv.entries)
	all := make([]*HashEntry, selfLen, selfLen+len(oh.entries))
	copy(all, hv.entries)
	for _, entry := range oh.entries {
		if idx, ok := index[ToKey(entry.key)]; ok {
			all[idx] = entry
		} else {
			all = append(all, entry)
		}
	}
	return all
}

func (hv *HashValue) Slice(i int, j int) IndexedValue {
	return WrapHash(hv.entries[i:j])
}

func (hv *HashValue) String() string {
	return ToString2(hv, NONE)
}

func (hv *HashValue) ToString(b Writer, s FormatContext, g RDetect) {
	hv.ToString2(b, s, GetFormat(s.FormatMap(), hv.Type()), '{', g)
}

func (hv *HashValue) ToString2(b Writer, s FormatContext, f Format, delim byte, g RDetect) {
	switch f.FormatChar() {
	case 'a':
		WrapArray(hv.Elements()).ToString(b, s, g)
		return
	case 'h', 's', 'p':
		indent := s.Indentation()
		indent = indent.Indenting(f.IsAlt() || indent.IsIndenting())

		if indent.Breaks() {
			WriteString(b, "\n")
			WriteString(b, indent.Padding())
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

		if f.IsAlt() {
			WriteString(b, "\n")
		}

		top := len(hv.entries)
		if top > 0 {
			sep := f.Separator(`,`)
			assoc := f.Separator2(` => `)
			cf := f.ContainerFormats()
			if cf == nil {
				cf = DEFAULT_CONTAINER_FORMATS
			}
			if f.IsAlt() {
				sep += "\n"
			} else {
				sep += ` `
			}

			childrenIndent := indent.Increase(f.IsAlt())
			padding := ``
			if f.IsAlt() {
				padding = childrenIndent.Padding()
			}

			last := top - 1
			for idx, entry := range hv.entries {
				k := entry.Key()
				WriteString(b, padding)
				if isContainer(k) {
					k.ToString(b, NewFormatContext2(childrenIndent, s.FormatMap()), g)
				} else {
					k.ToString(b, NewFormatContext2(childrenIndent, cf), g)
				}
				v := entry.Value()
				WriteString(b, assoc)
				if isContainer(v) {
					v.ToString(b, NewFormatContext2(childrenIndent, s.FormatMap()), g)
				} else {
					if v == nil {
						panic(`not good`)
					}
					v.ToString(b, NewFormatContext2(childrenIndent, cf), g)
				}
				if idx < last {
					WriteString(b, sep)
				}
			}
		}

		if f.IsAlt() {
			WriteString(b, "\n")
			WriteString(b, indent.Padding())
		}
		if delims[1] != 0 {
			b.Write(delims[1:])
		}
	default:
		panic(s.UnsupportedFormat(hv.Type(), `hasp`, f))
	}
}

func (hv *HashValue) Type() PType {
	hv.lock.Lock()
	t := hv.prtvReducedType()
	hv.lock.Unlock()
	return t
}

func (hv *HashValue) Values() IndexedValue {
	values := make([]PValue, len(hv.entries))
	for idx, entry := range hv.entries {
		values[idx] = entry.value
	}
	return WrapArray(values)
}

func (hv *HashValue) prtvDetailedType() PType {
	if hv.detailedType == nil {
		top := len(hv.entries)
		if top == 0 {
			hv.detailedType = hv.prtvReducedType()
			return hv.detailedType
		}

		for _, entry := range hv.entries {
			if sv, ok := entry.key.(*StringValue); !ok || len(sv.String()) == 0 {
				firstEntry := hv.entries[0]
				commonKeyType := DetailedValueType(firstEntry.key)
				commonValueType := DetailedValueType(firstEntry.value)
				for idx := 1; idx < top; idx++ {
					entry := hv.entries[idx]
					commonKeyType = commonType(commonKeyType, DetailedValueType(entry.key))
					commonValueType = commonType(commonValueType, DetailedValueType(entry.value))
				}
				sz := int64(len(hv.entries))
				hv.detailedType = NewHashType(commonKeyType, commonValueType, NewIntegerType(sz, sz))
				return hv.detailedType
			}
		}

		structEntries := make([]*StructElement, top)
		for idx, entry := range hv.entries {
			structEntries[idx] = NewStructElement(entry.key, DetailedValueType(entry.value))
		}
		hv.detailedType = NewStructType(structEntries)
	}
	return hv.detailedType
}

func (hv *HashValue) prtvReducedType() PType {
	if hv.reducedType == nil {
		top := len(hv.entries)
		if top == 0 {
			hv.reducedType = EmptyHashType()
		} else {
			firstEntry := hv.entries[0]
			commonKeyType := firstEntry.key.Type()
			commonValueType := firstEntry.value.Type()
			for idx := 1; idx < top; idx++ {
				entry := hv.entries[idx]
				commonKeyType = commonType(commonKeyType, entry.key.Type())
				commonValueType = commonType(commonValueType, entry.value.Type())
			}
			sz := int64(len(hv.entries))
			hv.reducedType = NewHashType(commonKeyType, commonValueType, NewIntegerType(sz, sz))
		}
	}
	return hv.reducedType
}

func (hv *HashValue) valueIndex() map[HashKey]int {
	hv.lock.Lock()
	if hv.index == nil {
		result := make(map[HashKey]int, len(hv.entries))
		for idx, entry := range hv.entries {
			result[ToKey(entry.key)] = idx
		}
		hv.index = result
	}
	hv.lock.Unlock()
	return hv.index
}

// PutAll merges the given hash into this hash (mutates the value). The method
// is not thread safe
func (hv *MutableHashValue) PutAll(o *HashValue) {
	hv.entries = hv.mergeEntries(o)
	hv.detailedType = nil
	hv.index = nil
}

// MergeSelf merges the given hash into this hash (mutates the value)
func (hv *MutableHashValue) Put(k, v PValue) {
	hv.PutAll(WrapHash([]*HashEntry{{k, v}}))
}
