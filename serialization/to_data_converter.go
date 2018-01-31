package serialization

import (
	"bytes"
	"fmt"
	. "github.com/puppetlabs/go-evaluator/evaluator"
	. "github.com/puppetlabs/go-evaluator/types"
	. "github.com/puppetlabs/go-parser/issue"
	"strings"
)

type (
	ToDataConverter struct {
		typeByReference bool
		localReference  bool
		symbolAsString  bool
		richData        bool
		messagePrefix   string
		path            []PValue
		values          map[PValue]interface{}
		recursiveLock   map[PValue]bool
	}
)

func NewToDataConverter(options KeyedValue) *ToDataConverter {
	t := &ToDataConverter{}
	t.typeByReference = options.Get5(`type_by_reference`, Boolean_TRUE).(*BooleanValue).Bool()
	t.localReference = options.Get5(`local_reference`, Boolean_TRUE).(*BooleanValue).Bool()
	t.symbolAsString = options.Get5(`symbol_as_string`, Boolean_FALSE).(*BooleanValue).Bool()
	t.richData = options.Get5(`rich_data`, Boolean_FALSE).(*BooleanValue).Bool()
	t.messagePrefix = options.Get5(`message_prefix`, EMPTY_STRING).String()
	return t
}

func (t *ToDataConverter) Convert(value PValue) PValue {
	t.path = make([]PValue, 0, 16)
	t.values = make(map[PValue]interface{}, 63)
	return t.toData(value)
}

func toJsonPath(path []PValue) string {
	s := bytes.NewBufferString(`$`)
	for _, v := range path {
		if v == nil {
			s.WriteString(`[null]`)
		} else if IsInstance(DefaultScalarType(), v) {
			s.WriteByte('[')
			v.ToString(s, PROGRAM, nil)
			s.WriteByte(']')
		} else {
			return ``
		}
	}
	return s.String()
}

func (t *ToDataConverter) pathToString() string {
	return fmt.Sprintf(`%s%s`, t.messagePrefix, toJsonPath(t.path)[1:])
}

func (t *ToDataConverter) toData(value PValue) PValue {
	if value == UNDEF || IsInstance(DefaultDataType(), value) {
		return value
	}
	if value == WrapDefault() {
		if t.richData {
			return SingletonHash2(PCORE_TYPE_KEY, WrapString(string(PCORE_TYPE_DEFAULT)))
		}
		Warning(EVAL_SERIALIZATION_DEFAULT_CONVERTED_TO_STRING, H{`path`: t.pathToString()})
		return WrapString(`default`)
	}

	if h, ok := value.(KeyedValue); ok {
		return t.process(h, func() PValue {
			if h.AllPairs(func(k, v PValue) bool {
				if _, ok := k.(*StringValue); ok {
					return true
				}
				return false
			}) {
				result := make([]*HashEntry, 0, h.Len())
				h.EachPair(func(key, elem PValue) {
					t.with(key, func() PValue {
						elem = t.toData(elem)
						result = append(result, WrapHashEntry(key, elem))
						return elem
					})
				})
				return WrapHash(result)
			}
			return t.nonStringKeyedHashToData(h)
		})
	}

	if a, ok := value.(IndexedValue); ok {
		return t.process(a, func() PValue {
			result := make([]PValue, 0, a.Len())
			a.EachWithIndex(func(elem PValue, index int) {
				t.with(WrapInteger(int64(index)), func() PValue {
					elem = t.toData(elem)
					result = append(result, elem)
					return elem
				})
			})
			return WrapArray(result)
		})
	}

	if sv, ok := value.(*SensitiveValue); ok {
		return t.process(sv, func() PValue {
			return WrapHash3(map[string]PValue{PCORE_TYPE_KEY: WrapString(PCORE_TYPE_SENSITIVE), PCORE_VALUE_KEY: t.toData(sv.Unwrap())})
		})
	}
	return t.unknownToData(value)
}

func (t *ToDataConverter) withRecursiveGuard(value PValue, producer Producer) PValue {
	if t.recursiveLock == nil {
		t.recursiveLock = map[PValue]bool{value: true}
	} else {
		if _, ok := t.recursiveLock[value]; ok {
			panic(Error(EVAL_SERIALIZATION_ENDLESS_RECURSION, H{`type_name`: value.Type().Name()}))
		}
		t.recursiveLock[value] = true
	}
	v := producer()
	delete(t.recursiveLock, value)
	return v
}

func (t *ToDataConverter) unknownToData(value PValue) PValue {
	if t.richData {
		return t.valueToDataHash(value)
	}
	return t.unkownToStringWithWarning(value)
}

func (t *ToDataConverter) unkownToStringWithWarning(value PValue) PValue {
	warn := true
	klass := ``
	s := ``
	if rt, ok := value.(*RuntimeValue); ok {
		if sym, ok := rt.Interface().(Symbol); ok {
			s = string(sym)
			warn = !t.symbolAsString
			klass = `Symbol`
		} else {
			s = fmt.Sprintf(`%v`, rt.Interface())
			klass = rt.Type().(*RuntimeType).Name()
		}
	} else {
		s = value.String()
		klass = value.Type().Name()
	}
	if warn {
		Warning(EVAL_SERIALIZATION_UNKNOWN_CONVERTED_TO_STRING, H{`path`: t.pathToString(), `klass`: klass, `value`: s})
	}
	return WrapString(s)
}

func (t *ToDataConverter) process(value PValue, producer Producer) PValue {
	if t.localReference {
		if ref, ok := t.values[value]; ok {
			if hash, ok := ref.(KeyedValue); ok {
				return hash
			}
			jsonRef := toJsonPath(ref.([]PValue))
			if jsonRef == `` {
				// Complex key and hence no way to reference the prior value. The value must therefore be
				// duplicated which in turn introduces a risk for endless recursion in case of self
				// referencing structures
				return t.withRecursiveGuard(value, producer)
			}
			v := WrapHash3(map[string]PValue{PCORE_TYPE_KEY: WrapString(PCORE_LOCAL_REF_SYMBOL), PCORE_VALUE_KEY: WrapString(jsonRef)})
			t.values[value] = v
			return v
		}
		t.values[value] = CopyValues(t.path)
		return producer()
	}
	return t.withRecursiveGuard(value, producer)
}

func (t *ToDataConverter) with(key PValue, producer Producer) PValue {
	t.path = append(t.path, key)
	value := producer()
	t.path = t.path[0 : len(t.path)-1]
	return value
}

func (t *ToDataConverter) nonStringKeyedHashToData(hash KeyedValue) PValue {
	if t.richData {
		return t.toKeyExtendedHash(hash)
	}
	result := make([]*HashEntry, 0, hash.Len())
	hash.EachPair(func(key, elem PValue) {
		t.with(key, func() PValue {
			if _, ok := key.(*StringValue); !ok {
				key = t.unkownToStringWithWarning(key)
			}
			result = append(result, WrapHashEntry(key, t.toData(elem)))
			return UNDEF
		})
	})
	return WrapHash(result)
}

func (t *ToDataConverter) valueToDataHash(value PValue) PValue {
	if rt, ok := value.(*RuntimeValue); ok {
		if !t.symbolAsString {
			if sym, ok := rt.Interface().(Symbol); ok {
				return WrapHash3(map[string]PValue{PCORE_TYPE_KEY: WrapString(PCORE_TYPE_SYMBOL), PCORE_VALUE_KEY: WrapString(string(sym))})
			}
		}
		return t.unkownToStringWithWarning(value)
	}
	pcoreTv := t.pcoreTypeToData(value.Type())
	if ss, ok := value.(SerializeAsString); ok {
		return WrapHash3(map[string]PValue{PCORE_TYPE_KEY: pcoreTv, PCORE_VALUE_KEY: WrapString(ss.SerializationString())})
	}

	if po, ok := value.(PuppetObject); ok {
		ih := t.toData(po.InitHash()).(KeyedValue)
		entries := make([]*HashEntry, 0, ih.Len())
		entries = append(entries, WrapHashEntry2(PCORE_TYPE_KEY, pcoreTv))
		ih.Each(func(v PValue) { entries = append(entries, v.(*HashEntry)) })
		return WrapHash(entries)
	}

	return t.unkownToStringWithWarning(value)
}

func (t *ToDataConverter) pcoreTypeToData(pcoreType PType) PValue {
	typeName := pcoreType.Name()
	if t.typeByReference || strings.HasPrefix(typeName, `Pcore::`) {
		return WrapString(typeName)
	}
	return t.with(WrapString(PCORE_TYPE_KEY), func() PValue { return t.toData(pcoreType) })
}

func (t *ToDataConverter) toKeyExtendedHash(hash KeyedValue) PValue {
	pairs := make([]PValue, 0, 2*hash.Len())
	hash.EachPair(func(key, value PValue) {
		key = t.toData(key)
		pairs = append(pairs, key, t.with(key, func() PValue { return t.toData(value) }))
	})
	return WrapHash3(map[string]PValue{PCORE_TYPE_KEY: WrapString(PCORE_TYPE_HASH), PCORE_VALUE_KEY: WrapArray(pairs)})
}
