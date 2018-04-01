package types

import (
	"fmt"
	"io"
	"math"
	"sort"
	"sync"

	"github.com/puppetlabs/go-evaluator/errors"
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/hash"
)

type (
	HashType struct {
		size      *IntegerType
		keyType   eval.PType
		valueType eval.PType
	}

	HashEntry struct {
		key   eval.PValue
		value eval.PValue
	}

	HashValue struct {
		lock         sync.Mutex
		reducedType  eval.PType
		detailedType eval.PType
		entries      []*HashEntry
		index        map[eval.HashKey]int
	}

	MutableHashValue struct {
		HashValue
	}
)

var hashType_EMPTY = &HashType{integerType_ZERO, unitType_DEFAULT, unitType_DEFAULT}
var hashType_DEFAULT = &HashType{integerType_POSITIVE, anyType_DEFAULT, anyType_DEFAULT}

var Hash_Type eval.ObjectType

func init() {
	Hash_Type = newObjectType(`Pcore::HashType`,
`Pcore::CollectionType {
	attributes => {
		key_type => {
			type => Optional[Type],
			value => Any
		},
		value_type => {
			type => Optional[Type],
			value => Any
		},
	}
}`, func(ctx eval.EvalContext, args []eval.PValue) eval.PValue {
			return NewHashType2(args...)
		})

	newGoConstructor2(`Hash`,
		func(t eval.LocalTypes) {
			t.Type(`KeyValueArray`, `Array[Tuple[Any,Any],1]`)
			t.Type(`TreeArray`,     `Array[Tuple[Array,Any],1]`)
			t.Type(`NewHashOption`, `Enum[tree, hash_tree]`)
		},

		func(d eval.Dispatch) {
			d.Param(`TreeArray`)
			d.OptionalParam(`NewHashOption`)
			d.Function(func(c eval.EvalContext, args []eval.PValue) eval.PValue {
				if len(args) < 2 {
					return WrapHashFromArray(args[0].(*ArrayValue))
				}
				allHashes := args[1].String() == `hash_tree`
				result := NewMutableHash()
				args[0].(*ArrayValue).Each(func(entry eval.PValue) {
					tpl := entry.(*ArrayValue)
					path := tpl.At(0).(*ArrayValue)
					value := tpl.At(1)
					if path.IsEmpty() {
						// root node (index [] was included - values merge into the result)
						//  An array must be changed to a hash first as this is the root
						// (Cannot return an array from a Hash.new)
						if av, ok := value.(*ArrayValue); ok {
							result.PutAll(IndexedFromArray(av))
						} else {
							if hv, ok := value.(eval.KeyedValue); ok {
								result.PutAll(hv)
							}
						}
					} else {
						r := path.Slice(0, path.Len() - 1).Reduce2(result, func(memo, idx eval.PValue) eval.PValue {
							if hv, ok := memo.(*MutableHashValue); ok {
								return hv.Get3(idx, func() eval.PValue {
									x := NewMutableHash()
									hv.Put(idx, x)
									return x
								})
							}
							if av, ok := memo.(eval.IndexedValue); ok {
								if ix, ok := idx.(*IntegerValue); ok {
									return av.At(int(ix.Int()))
								}
							}
							return _UNDEF
						})
						if hr, ok := r.(*MutableHashValue); ok {
							if allHashes {
								if av, ok := value.(*ArrayValue); ok {
									value = IndexedFromArray(av)
								}
							}
							hr.Put(path.At(path.Len()-1), value)
						}
					}
				})
				return &result.HashValue
			})
		},

		func(d eval.Dispatch) {
			d.Param(`KeyValueArray`)
			d.Function(func(c eval.EvalContext, args []eval.PValue) eval.PValue {
				return WrapHashFromArray(args[0].(*ArrayValue))
			})
		},

		func(d eval.Dispatch) {
			d.Param(`Iterable`)
			d.Function(func(c eval.EvalContext, args []eval.PValue) eval.PValue {
				arg := args[0]
				switch arg.(type) {
				case *ArrayValue:
					return WrapHashFromArray(arg.(*ArrayValue))
				case *HashValue:
					return arg
				default:
					return WrapHashFromArray(arg.(eval.IterableValue).Iterator().AsArray().(*ArrayValue))
				}
			})
		},
	)
}

func DefaultHashType() *HashType {
	return hashType_DEFAULT
}

func EmptyHashType() *HashType {
	return hashType_EMPTY
}

func NewHashType(keyType eval.PType, valueType eval.PType, rng *IntegerType) *HashType {
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

func NewHashType2(args ...eval.PValue) *HashType {
	argc := len(args)
	if argc == 0 {
		return hashType_DEFAULT
	}

	if argc == 1 || argc > 4 {
		panic(errors.NewIllegalArgumentCount(`Hash[]`, `0, 2, or 3`, argc))
	}

	offset := 0
	var valueType eval.PType
	keyType, ok := args[0].(eval.PType)
	if ok {
		valueType, ok = args[1].(eval.PType)
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

func (t *HashType) Accept(v eval.Visitor, g eval.Guard) {
	v(t)
	t.size.Accept(v, g)
	t.keyType.Accept(v, g)
	t.valueType.Accept(v, g)
}

func (t *HashType) Default() eval.PType {
	return hashType_DEFAULT
}

func (t *HashType) EntryType() eval.PType {
	return NewTupleType([]eval.PType{t.keyType, t.valueType}, nil)
}

func (t *HashType) Equals(o interface{}, g eval.Guard) bool {
	if ot, ok := o.(*HashType); ok {
		return t.size.Equals(ot.size, g) && t.keyType.Equals(ot.keyType, g) && t.valueType.Equals(ot.valueType, g)
	}
	return false
}

func (t *HashType) Generic() eval.PType {
	return NewHashType(eval.GenericType(t.keyType), eval.GenericType(t.valueType), nil)
}

func (t *HashType) Get(key string) (value eval.PValue, ok bool) {
	switch key {
	case `key_type`:
		return t.keyType, true
	case `value_type`:
		return t.valueType, true
	case `size_type`:
		return t.size, true
	}
	return nil, false
}

func (t *HashType) IsAssignable(o eval.PType, g eval.Guard) bool {
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

func (t *HashType) IsInstance(o eval.PValue, g eval.Guard) bool {
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

func (t *HashType) KeyType() eval.PType {
	return t.keyType
}

func (t *HashType) MetaType() eval.ObjectType {
	return Hash_Type
}

func (t *HashType) Name() string {
	return `Hash`
}

func (t *HashType) Parameters() []eval.PValue {
	if *t == *hashType_DEFAULT {
		return eval.EMPTY_VALUES
	}
	if *t == *hashType_EMPTY {
		return []eval.PValue{ZERO, ZERO}
	}
	params := make([]eval.PValue, 0, 4)
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
	return eval.ToString2(t, NONE)
}

func (t *HashType) ValueType() eval.PType {
	return t.valueType
}

func (t *HashType) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *HashType) Type() eval.PType {
	return &TypeType{t}
}

func WrapHashEntry(key eval.PValue, value eval.PValue) *HashEntry {
	return &HashEntry{key, value}
}

func WrapHashEntry2(key string, value eval.PValue) *HashEntry {
	return &HashEntry{WrapString(key), value}
}

func (he *HashEntry) Add(v eval.PValue) eval.IndexedValue {
	panic(`Operation not supported`)
}

func (he *HashEntry) AddAll(v eval.IndexedValue) eval.IndexedValue {
	panic(`Operation not supported`)
}

func (he *HashEntry) All(predicate eval.Predicate) bool {
	return predicate(he.key) && predicate(he.value)
}

func (he *HashEntry) Any(predicate eval.Predicate) bool {
	return predicate(he.key) || predicate(he.value)
}

func (he *HashEntry) AppendTo(slice []eval.PValue) []eval.PValue {
	return append(slice, he.key, he.value)
}

func (he *HashEntry) At(i int) eval.PValue {
	switch i {
	case 0:
		return he.key
	case 1:
		return he.value
	default:
		return _UNDEF
	}
}

func (he *HashEntry) Delete(v eval.PValue) eval.IndexedValue {
	panic(`Operation not supported`)
}

func (he *HashEntry) DeleteAll(v eval.IndexedValue) eval.IndexedValue {
	panic(`Operation not supported`)
}

func (he *HashEntry) DetailedType() eval.PType {
	return NewTupleType([]eval.PType{eval.DetailedValueType(he.key), eval.DetailedValueType(he.value)}, NewIntegerType(2, 2))
}

func (he *HashEntry) Each(consumer eval.Consumer) {
	consumer(he.key)
	consumer(he.value)
}

func (he *HashEntry) EachSlice(n int, consumer eval.SliceConsumer) {
	if n == 1 {
		consumer(SingletonArray(he.key))
		consumer(SingletonArray(he.value))
	} else if n >= 2 {
		consumer(he)
	}
}

func (he *HashEntry) EachWithIndex(consumer eval.IndexedConsumer) {
	consumer(he.key, 0)
	consumer(he.value, 1)
}

func (he *HashEntry) ElementType() eval.PType {
	return commonType(he.key.Type(), he.value.Type())
}

func (he *HashEntry) Equals(o interface{}, g eval.Guard) bool {
	if ov, ok := o.(*HashEntry); ok {
		return he.key.Equals(ov.key, g) && he.value.Equals(ov.value, g)
	}
	if iv, ok := o.(*ArrayValue); ok && iv.Len() == 2 {
		return he.key.Equals(iv.At(0), g) && he.value.Equals(iv.At(1), g)
	}
	return false
}

func (he *HashEntry) Find(predicate eval.Predicate) (eval.PValue, bool) {
	if predicate(he.key) {
		return he.key, true
	}
	if predicate(he.value) {
		return he.value, true
	}
	return nil, false
}

func (he *HashEntry) Flatten() eval.IndexedValue {
	return WrapArray([]eval.PValue{he.key, he.value}).Flatten()
}

func (he *HashEntry) IsEmpty() bool {
	return false
}

func (he *HashEntry) IsHashStyle() bool {
	return false
}

func (he *HashEntry) Iterator() eval.Iterator {
	return &indexedIterator{he.ElementType(), -1, he}
}

func (he *HashEntry) Key() eval.PValue {
	return he.key
}

func (he *HashEntry) Len() int {
	return 2
}

func (he *HashEntry) Map(mapper eval.Mapper) eval.IndexedValue {
	return WrapArray([]eval.PValue{mapper(he.key), mapper(he.value)})
}

func (he *HashEntry) Select(predicate eval.Predicate) eval.IndexedValue {
	if predicate(he.key) {
		if predicate(he.value) {
			return he
		}
		return SingletonArray(he.key)
	}
	if predicate(he.value) {
		return SingletonArray(he.value)
	}
	return eval.EMPTY_ARRAY
}

func (he *HashEntry) Slice(i int, j int) eval.IndexedValue {
	if i > 1 || i >= j {
		return eval.EMPTY_ARRAY
	}
	if i == 1 {
		return SingletonArray(he.value)
	}
	if j == 1 {
		return SingletonArray(he.key)
	}
	return he
}

func (he *HashEntry) Reduce(redactor eval.BiMapper) eval.PValue {
	return redactor(he.key, he.value)
}

func (he *HashEntry) Reduce2(initialValue eval.PValue, redactor eval.BiMapper) eval.PValue {
	return redactor(redactor(initialValue, he.key), he.value)
}

func (he *HashEntry) Reject(predicate eval.Predicate) eval.IndexedValue {
	if predicate(he.key) {
		if predicate(he.value) {
			return eval.EMPTY_ARRAY
		}
		return SingletonArray(he.value)
	}
	if predicate(he.value) {
		return SingletonArray(he.key)
	}
	return he
}

func (he *HashEntry) Type() eval.PType {
	return NewArrayType(commonType(he.key.Type(), he.value.Type()), NewIntegerType(2, 2))
}

func (he *HashEntry) Unique() eval.IndexedValue {
	if he.key.Equals(he.value, nil) {
		return SingletonArray(he.key)
	}
	return he
}

func (he *HashEntry) Value() eval.PValue {
	return he.value
}

func (he *HashEntry) String() string {
	return eval.ToString2(he, NONE)
}

func (he *HashEntry) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	WrapArray([]eval.PValue{he.key, he.value}).ToString(b, s, g)
}

func WrapHash(entries []*HashEntry) *HashValue {
	return &HashValue{entries: entries}
}

func WrapHash2(entries eval.IndexedValue) *HashValue {
	hvEntries := make([]*HashEntry, entries.Len())
	entries.EachWithIndex(func(entry eval.PValue, idx int) {
		hvEntries[idx] = entry.(*HashEntry)
	})
	return &HashValue{entries: hvEntries}
}

// WrapHash3 does not preserve order since order is undefined in a Go map
func WrapHash3(hash map[string]eval.PValue) *HashValue {
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

// WrapHash4 does not preserve order since order is undefined in a Go map
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

func WrapHash5(hash *hash.StringHash) *HashValue {
	hvEntries := make([]*HashEntry, hash.Len())
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
		a.EachWithIndex(func(pair eval.PValue, idx int) {
			pairArr := pair.(eval.IndexedValue)
			if pairArr.Len() != 2 {
				panic(errors.NewArgumentsError(`Hash`, fmt.Sprintf(`hash entry array must have 2 elements, got %d`, pairArr.Len())))
			}
			entries[idx] = WrapHashEntry(pairArr.At(0), pairArr.At(1))
		})
		return WrapHash(entries)
	default:
		if (top % 2) != 0 {
			panic(errors.NewArgumentsError(`Hash`, `odd number of arguments in Array`))
		}
		entries := make([]*HashEntry, top/2)
		idx := 0
		a.EachSlice(2, func(slice eval.IndexedValue) {
			entries[idx] = WrapHashEntry(slice.At(0), slice.At(1))
			idx++
		})
		return WrapHash(entries)
	}
}

func IndexedFromArray(a *ArrayValue) *HashValue {
	top := a.Len()
	entries := make([]*HashEntry, top)
	a.EachWithIndex(func(v eval.PValue, idx int) {
		entries[idx] = WrapHashEntry(WrapInteger(int64(idx)), v)
	})
	return WrapHash(entries)
}

func SingletonHash(key, value eval.PValue) *HashValue {
	return &HashValue{entries: []*HashEntry{WrapHashEntry(key, value)}}
}

func SingletonHash2(key string, value eval.PValue) *HashValue {
	return &HashValue{entries: []*HashEntry{WrapHashEntry2(key, value)}}
}

func (hv *HashValue) Add(v eval.PValue) eval.IndexedValue {
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

func (hv *HashValue) AddAll(v eval.IndexedValue) eval.IndexedValue {
	switch v.(type) {
	case *HashValue:
		return hv.Merge(v.(*HashValue))
	case *ArrayValue:
		return hv.Merge(WrapHashFromArray(v.(*ArrayValue)))
	}
	panic(`Operation not supported`)
}

func (hv *HashValue) All(predicate eval.Predicate) bool {
	for _, e := range hv.entries {
		if !predicate(e) {
			return false
		}
	}
	return true
}

func (hv *HashValue) AllPairs(predicate eval.BiPredicate) bool {
	for _, e := range hv.entries {
		if !predicate(e.key, e.value) {
			return false
		}
	}
	return true
}

func (hv *HashValue) Any(predicate eval.Predicate) bool {
	for _, e := range hv.entries {
		if predicate(e) {
			return true
		}
	}
	return false
}

func (hv *HashValue) AnyPair(predicate eval.BiPredicate) bool {
	for _, e := range hv.entries {
		if predicate(e.key, e.value) {
			return true
		}
	}
	return false
}

func (hv *HashValue) AppendEntriesTo(entries []*HashEntry) []*HashEntry {
	return append(entries, hv.entries...)
}

func (hv *HashValue) AppendTo(slice []eval.PValue) []eval.PValue {
	for _, e := range hv.entries {
		slice = append(slice, e)
	}
	return slice
}

func (hv *HashValue) At(i int) eval.PValue {
	if i >= 0 && i < len(hv.entries) {
		return hv.entries[i]
	}
	return _UNDEF
}

func (hv *HashValue) Delete(key eval.PValue) eval.IndexedValue {
	if idx, ok := hv.valueIndex()[eval.ToKey(key)]; ok {
		return WrapHash(append(hv.entries[:idx], hv.entries[idx+1:]...))
	}
	return hv
}

func (hv *HashValue) DeleteAll(keys eval.IndexedValue) eval.IndexedValue {
	entries := hv.entries
	valueIndex := hv.valueIndex()
	keys.Each(func(key eval.PValue) {
		if idx, ok := valueIndex[eval.ToKey(key)]; ok {
			entries = append(hv.entries[:idx], hv.entries[idx+1:]...)
		}
	})
	if len(hv.entries) == len(entries) {
		return hv
	}
	return WrapHash(entries)
}

func (hv *HashValue) DetailedType() eval.PType {
	hv.lock.Lock()
	t := hv.prtvDetailedType()
	hv.lock.Unlock()
	return t
}

func (hv *HashValue) ElementType() eval.PType {
	return hv.Type().(*HashType).EntryType()
}

func (hv *HashValue) Entries() eval.IndexedValue {
	return hv
}

func (hv *HashValue) Each(consumer eval.Consumer) {
	for _, e := range hv.entries {
		consumer(e)
	}
}

func (hv *HashValue) EachSlice(n int, consumer eval.SliceConsumer) {
	top := len(hv.entries)
	for i := 0; i < top; i += n {
		e := i + n
		if e > top {
			e = top
		}
		consumer(WrapArray(ValueSlice(hv.entries[i:e])))
	}
}

func (hv *HashValue) EachKey(consumer eval.Consumer) {
	for _, e := range hv.entries {
		consumer(e.key)
	}
}

func (hv *HashValue) Find(predicate eval.Predicate) (eval.PValue, bool) {
	for _, e := range hv.entries {
		if predicate(e) {
			return e, true
		}
	}
	return nil, false
}

func (hv *HashValue) Flatten() eval.IndexedValue {
	els := make([]eval.PValue, 0, len(hv.entries) * 2)
	for _, he := range hv.entries {
		els = append(els, he.key, he.value)
	}
	return WrapArray(els).Flatten()
}

func (hv *HashValue) Map(mapper eval.Mapper) eval.IndexedValue {
	mapped := make([]eval.PValue, len(hv.entries))
	for i, e := range hv.entries {
		mapped[i] = mapper(e)
	}
	return WrapArray(mapped)
}

func (hv *HashValue) MapValues(mapper eval.Mapper) eval.KeyedValue {
	mapped := make([]*HashEntry, len(hv.entries))
	for i, e := range hv.entries {
		mapped[i] = WrapHashEntry(e.key, mapper(e.value))
	}
	return WrapHash(mapped)
}

func (hv *HashValue) Select(predicate eval.Predicate) eval.IndexedValue {
	selected := make([]*HashEntry, 0)
	for _, e := range hv.entries {
		if predicate(e) {
			selected = append(selected, e)
		}
	}
	return WrapHash(selected)
}

func (hv *HashValue) SelectPairs(predicate eval.BiPredicate) eval.KeyedValue {
	selected := make([]*HashEntry, 0)
	for _, e := range hv.entries {
		if predicate(e.key, e.value) {
			selected = append(selected, e)
		}
	}
	return WrapHash(selected)
}

func (hv *HashValue) Reduce(redactor eval.BiMapper) eval.PValue {
	if hv.IsEmpty() {
		return _UNDEF
	}
	return reduceEntries(hv.entries[1:], hv.At(0), redactor)
}

func (hv *HashValue) Reduce2(initialValue eval.PValue, redactor eval.BiMapper) eval.PValue {
	return reduceEntries(hv.entries, initialValue, redactor)
}

func (hv *HashValue) Reject(predicate eval.Predicate) eval.IndexedValue {
	selected := make([]*HashEntry, 0)
	for _, e := range hv.entries {
		if !predicate(e) {
			selected = append(selected, e)
		}
	}
	return WrapHash(selected)
}

func (hv *HashValue) RejectPairs(predicate eval.BiPredicate) eval.KeyedValue {
	selected := make([]*HashEntry, 0)
	for _, e := range hv.entries {
		if !predicate(e.key, e.value) {
			selected = append(selected, e)
		}
	}
	return WrapHash(selected)
}

func (hv *HashValue) EachPair(consumer eval.BiConsumer) {
	for _, e := range hv.entries {
		consumer(e.key, e.value)
	}
}

func (hv *HashValue) EachValue(consumer eval.Consumer) {
	for _, e := range hv.entries {
		consumer(e.value)
	}
}

func (hv *HashValue) EachWithIndex(consumer eval.IndexedConsumer) {
	for i, e := range hv.entries {
		consumer(e, i)
	}
}

func (hv *HashValue) Equals(o interface{}, g eval.Guard) bool {
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

func (hv *HashValue) Get(key eval.PValue) (eval.PValue, bool) {
	return hv.get(eval.ToKey(key))
}

func (hv *HashValue) Get2(key eval.PValue, dflt eval.PValue) eval.PValue {
	return hv.get2(eval.ToKey(key), dflt)
}

func (hv *HashValue) Get3(key eval.PValue, dflt eval.Producer) eval.PValue {
	return hv.get3(eval.ToKey(key), dflt)
}

func (hv *HashValue) Get4(key string) (eval.PValue, bool) {
	return hv.get(eval.HashKey(key))
}

func (hv *HashValue) Get5(key string, dflt eval.PValue) eval.PValue {
	return hv.get2(eval.HashKey(key), dflt)
}

func (hv *HashValue) Get6(key string, dflt eval.Producer) eval.PValue {
	return hv.get3(eval.HashKey(key), dflt)
}

func (hv *HashValue) get(key eval.HashKey) (eval.PValue, bool) {
	if pos, ok := hv.valueIndex()[key]; ok {
		return hv.entries[pos].value, true
	}
	return _UNDEF, false
}

func (hv *HashValue) get2(key eval.HashKey, dflt eval.PValue) eval.PValue {
	if pos, ok := hv.valueIndex()[key]; ok {
		return hv.entries[pos].value
	}
	return dflt
}

func (hv *HashValue) get3(key eval.HashKey, dflt eval.Producer) eval.PValue {
	if pos, ok := hv.valueIndex()[key]; ok {
		return hv.entries[pos].value
	}
	return dflt()
}

func (hv *HashValue) IncludesKey(o eval.PValue) bool {
	_, ok := hv.valueIndex()[eval.ToKey(o)]
	return ok
}

func (hv *HashValue) IncludesKey2(key string) bool {
	_, ok := hv.valueIndex()[eval.HashKey(key)]
	return ok
}

func (hv *HashValue) IsEmpty() bool {
	return len(hv.entries) == 0
}

func (hv *HashValue) IsHashStyle() bool {
	return true
}

func (hv *HashValue) Iterator() eval.Iterator {
	t := hv.Type().(*HashType)
	et := NewTupleType([]eval.PType{t.KeyType(), t.ValueType()}, NewIntegerType(2, 2))
	return &indexedIterator{et, -1, hv}
}

func (hv *HashValue) Keys() eval.IndexedValue {
	keys := make([]eval.PValue, len(hv.entries))
	for idx, entry := range hv.entries {
		keys[idx] = entry.key
	}
	return WrapArray(keys)
}

func (hv *HashValue) Len() int {
	return len(hv.entries)
}

func (hv *HashValue) Merge(o eval.KeyedValue) eval.KeyedValue {
	return WrapHash(hv.mergeEntries(o))
}

func (hv *HashValue) mergeEntries(o eval.KeyedValue) []*HashEntry {
	oh := o.(*HashValue)
	index := hv.valueIndex()
	selfLen := len(hv.entries)
	all := make([]*HashEntry, selfLen, selfLen+len(oh.entries))
	copy(all, hv.entries)
	for _, entry := range oh.entries {
		if idx, ok := index[eval.ToKey(entry.key)]; ok {
			all[idx] = entry
		} else {
			all = append(all, entry)
		}
	}
	return all
}

func (hv *HashValue) Slice(i int, j int) eval.IndexedValue {
	return WrapHash(hv.entries[i:j])
}

type hashSorter struct {
	entries    []*HashEntry
	comparator eval.Comparator
}

func (s *hashSorter) Len() int {
	return len(s.entries)
}

func (s *hashSorter) Less(i, j int) bool {
	vs := s.entries
	return s.comparator(vs[i].key, vs[j].key)
}

func (s *hashSorter) Swap(i, j int) {
	vs := s.entries
	v := vs[i]
	vs[i] = vs[j]
	vs[j] = v
}

// Sort reorders the associations of this hash by applying the comparator
// to the keys
func (hv *HashValue) Sort(comparator eval.Comparator) eval.IndexedValue {
	s := &hashSorter{make([]*HashEntry, len(hv.entries)), comparator}
	copy(s.entries, hv.entries)
	sort.Sort(s)
	return WrapHash(s.entries)
}

func (hv *HashValue) String() string {
	return eval.ToString2(hv, NONE)
}

func (hv *HashValue) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	hv.ToString2(b, s, eval.GetFormat(s.FormatMap(), hv.Type()), '{', g)
}

func (hv *HashValue) ToString2(b io.Writer, s eval.FormatContext, f eval.Format, delim byte, g eval.RDetect) {
	switch f.FormatChar() {
	case 'a':
		WrapArray3(hv).ToString(b, s, g)
		return
	case 'h', 's', 'p':
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

		if f.IsAlt() {
			io.WriteString(b, "\n")
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
				io.WriteString(b, padding)
				if isContainer(k) {
					k.ToString(b, eval.NewFormatContext2(childrenIndent, s.FormatMap()), g)
				} else {
					k.ToString(b, eval.NewFormatContext2(childrenIndent, cf), g)
				}
				v := entry.Value()
				io.WriteString(b, assoc)
				if isContainer(v) {
					v.ToString(b, eval.NewFormatContext2(childrenIndent, s.FormatMap()), g)
				} else {
					if v == nil {
						panic(`not good`)
					}
					v.ToString(b, eval.NewFormatContext2(childrenIndent, cf), g)
				}
				if idx < last {
					io.WriteString(b, sep)
				}
			}
		}

		if f.IsAlt() {
			io.WriteString(b, "\n")
			io.WriteString(b, indent.Padding())
		}
		if delims[1] != 0 {
			b.Write(delims[1:])
		}
	default:
		panic(s.UnsupportedFormat(hv.Type(), `hasp`, f))
	}
}

func (hv *HashValue) Type() eval.PType {
	hv.lock.Lock()
	t := hv.prtvReducedType()
	hv.lock.Unlock()
	return t
}

// Unique on a HashValue will always return self since the keys of a hash are unique
func (hv *HashValue) Unique() eval.IndexedValue {
	return hv
}

func (hv *HashValue) Values() eval.IndexedValue {
	values := make([]eval.PValue, len(hv.entries))
	for idx, entry := range hv.entries {
		values[idx] = entry.value
	}
	return WrapArray(values)
}

func (hv *HashValue) prtvDetailedType() eval.PType {
	if hv.detailedType == nil {
		top := len(hv.entries)
		if top == 0 {
			hv.detailedType = hv.prtvReducedType()
			return hv.detailedType
		}

		for _, entry := range hv.entries {
			if sv, ok := entry.key.(*StringValue); !ok || len(sv.String()) == 0 {
				firstEntry := hv.entries[0]
				commonKeyType := eval.DetailedValueType(firstEntry.key)
				commonValueType := eval.DetailedValueType(firstEntry.value)
				for idx := 1; idx < top; idx++ {
					entry := hv.entries[idx]
					commonKeyType = commonType(commonKeyType, eval.DetailedValueType(entry.key))
					commonValueType = commonType(commonValueType, eval.DetailedValueType(entry.value))
				}
				sz := int64(len(hv.entries))
				hv.detailedType = NewHashType(commonKeyType, commonValueType, NewIntegerType(sz, sz))
				return hv.detailedType
			}
		}

		structEntries := make([]*StructElement, top)
		for idx, entry := range hv.entries {
			structEntries[idx] = NewStructElement(entry.key, eval.DetailedValueType(entry.value))
		}
		hv.detailedType = NewStructType(structEntries)
	}
	return hv.detailedType
}

func (hv *HashValue) prtvReducedType() eval.PType {
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

func (hv *HashValue) valueIndex() map[eval.HashKey]int {
	hv.lock.Lock()
	if hv.index == nil {
		result := make(map[eval.HashKey]int, len(hv.entries))
		for idx, entry := range hv.entries {
			result[eval.ToKey(entry.key)] = idx
		}
		hv.index = result
	}
	hv.lock.Unlock()
	return hv.index
}

func NewMutableHash() *MutableHashValue {
	return &MutableHashValue{HashValue{entries:make([]*HashEntry, 0, 7)}}
}

// PutAll merges the given hash into this hash (mutates the hash). The method
// is not thread safe
func (hv *MutableHashValue) PutAll(o eval.KeyedValue) {
	hv.entries = hv.mergeEntries(o)
	hv.detailedType = nil
	hv.index = nil
}

// Put adds or replaces the given key/value association in this hash
func (hv *MutableHashValue) Put(key, value eval.PValue) {
	hv.PutAll(WrapHash([]*HashEntry{{key, value}}))
}

func reduceEntries(slice []*HashEntry, initialValue eval.PValue, redactor eval.BiMapper) eval.PValue {
	memo := initialValue
	for _, v := range	slice {
		memo = redactor(memo, v)
	}
	return memo
}
