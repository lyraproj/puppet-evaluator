package serialization

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/types"
	"github.com/puppetlabs/go-parser/issue"
)

type (
	ToDataConverter struct {
		typeByReference bool
		localReference  bool
		symbolAsString  bool
		richData        bool
		messagePrefix   string
		context         eval.Context
		path            []eval.PValue
		values          map[eval.PValue]interface{}
		recursiveLock   map[eval.PValue]bool
	}
)

func NewToDataConverter(ctx eval.Context, options eval.KeyedValue) *ToDataConverter {
	t := &ToDataConverter{}
	t.typeByReference = options.Get5(`type_by_reference`, types.Boolean_TRUE).(*types.BooleanValue).Bool()
	t.localReference = options.Get5(`local_reference`, types.Boolean_TRUE).(*types.BooleanValue).Bool()
	t.symbolAsString = options.Get5(`symbol_as_string`, types.Boolean_FALSE).(*types.BooleanValue).Bool()
	t.richData = options.Get5(`rich_data`, types.Boolean_FALSE).(*types.BooleanValue).Bool()
	t.messagePrefix = options.Get5(`message_prefix`, eval.EMPTY_STRING).String()
	t.context = ctx
	return t
}

func (t *ToDataConverter) Convert(value eval.PValue) eval.PValue {
	t.path = make([]eval.PValue, 0, 16)
	t.values = make(map[eval.PValue]interface{}, 63)
	return t.toData(value)
}

func toJsonPath(path []eval.PValue) string {
	s := bytes.NewBufferString(`$`)
	for _, v := range path {
		if v == nil {
			s.WriteString(`[null]`)
		} else if eval.IsInstance(types.DefaultScalarType(), v) {
			s.WriteByte('[')
			v.ToString(s, types.PROGRAM, nil)
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

func (t *ToDataConverter) toData(value eval.PValue) eval.PValue {
	if value == eval.UNDEF || eval.IsInstance(types.DefaultDataType(), value) {
		return value
	}
	if value == types.WrapDefault() {
		if t.richData {
			return types.SingletonHash2(PCORE_TYPE_KEY, types.WrapString(string(PCORE_TYPE_DEFAULT)))
		}
		eval.Warning(t.context, eval.EVAL_SERIALIZATION_DEFAULT_CONVERTED_TO_STRING, issue.H{`path`: t.pathToString()})
		return types.WrapString(`default`)
	}

	if h, ok := value.(eval.KeyedValue); ok {
		return t.process(h, func() eval.PValue {
			if h.AllPairs(func(k, v eval.PValue) bool {
				if _, ok := k.(*types.StringValue); ok {
					return true
				}
				return false
			}) {
				result := make([]*types.HashEntry, 0, h.Len())
				h.EachPair(func(key, elem eval.PValue) {
					t.with(key, func() eval.PValue {
						elem = t.toData(elem)
						result = append(result, types.WrapHashEntry(key, elem))
						return elem
					})
				})
				return types.WrapHash(result)
			}
			return t.nonStringKeyedHashToData(h)
		})
	}

	if a, ok := value.(eval.IndexedValue); ok {
		return t.process(a, func() eval.PValue {
			result := make([]eval.PValue, 0, a.Len())
			a.EachWithIndex(func(elem eval.PValue, index int) {
				t.with(types.WrapInteger(int64(index)), func() eval.PValue {
					elem = t.toData(elem)
					result = append(result, elem)
					return elem
				})
			})
			return types.WrapArray(result)
		})
	}

	if sv, ok := value.(*types.SensitiveValue); ok {
		return t.process(sv, func() eval.PValue {
			return types.WrapHash3(map[string]eval.PValue{PCORE_TYPE_KEY: types.WrapString(PCORE_TYPE_SENSITIVE), PCORE_VALUE_KEY: t.toData(sv.Unwrap())})
		})
	}
	return t.unknownToData(value)
}

func (t *ToDataConverter) withRecursiveGuard(value eval.PValue, producer eval.Producer) eval.PValue {
	if t.recursiveLock == nil {
		t.recursiveLock = map[eval.PValue]bool{value: true}
	} else {
		if _, ok := t.recursiveLock[value]; ok {
			panic(eval.Error(t.context, eval.EVAL_SERIALIZATION_ENDLESS_RECURSION, issue.H{`type_name`: value.Type().Name()}))
		}
		t.recursiveLock[value] = true
	}
	v := producer()
	delete(t.recursiveLock, value)
	return v
}

func (t *ToDataConverter) unknownToData(value eval.PValue) eval.PValue {
	if t.richData {
		return t.valueToDataHash(value)
	}
	return t.unkownToStringWithWarning(value)
}

func (t *ToDataConverter) unkownToStringWithWarning(value eval.PValue) eval.PValue {
	warn := true
	klass := ``
	s := ``
	if rt, ok := value.(*types.RuntimeValue); ok {
		if sym, ok := rt.Interface().(Symbol); ok {
			s = string(sym)
			warn = !t.symbolAsString
			klass = `Symbol`
		} else {
			s = fmt.Sprintf(`%v`, rt.Interface())
			klass = rt.Type().(*types.RuntimeType).Name()
		}
	} else {
		s = value.String()
		klass = value.Type().Name()
	}
	if warn {
		eval.Warning(t.context, eval.EVAL_SERIALIZATION_UNKNOWN_CONVERTED_TO_STRING, issue.H{`path`: t.pathToString(), `klass`: klass, `value`: s})
	}
	return types.WrapString(s)
}

func (t *ToDataConverter) process(value eval.PValue, producer eval.Producer) eval.PValue {
	if t.localReference {
		if ref, ok := t.values[value]; ok {
			if hash, ok := ref.(eval.KeyedValue); ok {
				return hash
			}
			jsonRef := toJsonPath(ref.([]eval.PValue))
			if jsonRef == `` {
				// Complex key and hence no way to reference the prior value. The value must therefore be
				// duplicated which in turn introduces a risk for endless recursion in case of self
				// referencing structures
				return t.withRecursiveGuard(value, producer)
			}
			v := types.WrapHash3(map[string]eval.PValue{PCORE_TYPE_KEY: types.WrapString(PCORE_LOCAL_REF_SYMBOL), PCORE_VALUE_KEY: types.WrapString(jsonRef)})
			t.values[value] = v
			return v
		}
		t.values[value] = eval.CopyValues(t.path)
		return producer()
	}
	return t.withRecursiveGuard(value, producer)
}

func (t *ToDataConverter) with(key eval.PValue, producer eval.Producer) eval.PValue {
	t.path = append(t.path, key)
	value := producer()
	t.path = t.path[0 : len(t.path)-1]
	return value
}

func (t *ToDataConverter) nonStringKeyedHashToData(hash eval.KeyedValue) eval.PValue {
	if t.richData {
		return t.toKeyExtendedHash(hash)
	}
	result := make([]*types.HashEntry, 0, hash.Len())
	hash.EachPair(func(key, elem eval.PValue) {
		t.with(key, func() eval.PValue {
			if _, ok := key.(*types.StringValue); !ok {
				key = t.unkownToStringWithWarning(key)
			}
			result = append(result, types.WrapHashEntry(key, t.toData(elem)))
			return eval.UNDEF
		})
	})
	return types.WrapHash(result)
}

func (t *ToDataConverter) valueToDataHash(value eval.PValue) eval.PValue {
	if rt, ok := value.(*types.RuntimeValue); ok {
		if !t.symbolAsString {
			if sym, ok := rt.Interface().(Symbol); ok {
				return types.WrapHash3(map[string]eval.PValue{PCORE_TYPE_KEY: types.WrapString(PCORE_TYPE_SYMBOL), PCORE_VALUE_KEY: types.WrapString(string(sym))})
			}
		}
		return t.unkownToStringWithWarning(value)
	}

	var vt eval.PType
	if tp, ok := value.(eval.PType); ok {
		// The Type of Type X is a Type[X]. To create instances of X we need the MetaType (the Object
		// describing the type X)
		vt = tp.MetaType()
	} else {
		vt = value.Type()
	}

	pcoreTv := t.pcoreTypeToData(vt)
	if ss, ok := value.(eval.SerializeAsString); ok {
		return types.WrapHash3(map[string]eval.PValue{PCORE_TYPE_KEY: pcoreTv, PCORE_VALUE_KEY: types.WrapString(ss.SerializationString())})
	}

	if po, ok := value.(eval.PuppetObject); ok {
		ih := t.toData(po.InitHash()).(eval.KeyedValue)
		entries := make([]*types.HashEntry, 0, ih.Len())
		entries = append(entries, types.WrapHashEntry2(PCORE_TYPE_KEY, pcoreTv))
		ih.Each(func(v eval.PValue) { entries = append(entries, v.(*types.HashEntry)) })
		return types.WrapHash(entries)
	}

	if ot, ok := vt.(eval.ObjectType); ok {
		ai := ot.AttributesInfo()
		attrs := ai.Attributes()
		args := make([]eval.PValue, len(attrs))
		for i, a := range attrs {
			args[i] = a.Get(t.context, value)
		}

		for i := len(args) - 1; i >= ai.RequiredCount(); i-- {
			if !attrs[i].Default(args[i]) {
				break
			}
			args = args[:i]
		}
		entries := make([]*types.HashEntry, 0, len(attrs) + 1)
		entries = append(entries, types.WrapHashEntry2(PCORE_TYPE_KEY, pcoreTv))
		for i, a := range args {
			key := types.WrapString(attrs[i].Name())
			t.with(key, func() eval.PValue {
				a = t.toData(a)
				entries = append(entries, types.WrapHashEntry(key, a))
				return a
			})
		}
		return types.WrapHash(entries)
	}

	return t.unkownToStringWithWarning(value)
}

func (t *ToDataConverter) pcoreTypeToData(pcoreType eval.PType) eval.PValue {
	typeName := pcoreType.Name()
	if t.typeByReference || strings.HasPrefix(typeName, `Pcore::`) {
		return types.WrapString(typeName)
	}
	return t.with(types.WrapString(PCORE_TYPE_KEY), func() eval.PValue { return t.toData(pcoreType) })
}

func (t *ToDataConverter) toKeyExtendedHash(hash eval.KeyedValue) eval.PValue {
	pairs := make([]eval.PValue, 0, 2*hash.Len())
	hash.EachPair(func(key, value eval.PValue) {
		key = t.toData(key)
		pairs = append(pairs, key, t.with(key, func() eval.PValue { return t.toData(value) }))
	})
	return types.WrapHash3(map[string]eval.PValue{PCORE_TYPE_KEY: types.WrapString(PCORE_TYPE_HASH), PCORE_VALUE_KEY: types.WrapArray(pairs)})
}
