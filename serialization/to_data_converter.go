package serialization

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/types"
	"github.com/puppetlabs/go-issues/issue"
)

type (
	ToDataConverter struct {
		typeByReference bool
		localReference  bool
		symbolAsString  bool
		richData        bool
		messagePrefix   string
		path            []eval.Value
		values          map[eval.Value]interface{}
		recursiveLock   map[eval.Value]bool
	}
)

func NewToDataConverter(options eval.OrderedMap) *ToDataConverter {
	t := &ToDataConverter{}
	t.typeByReference = options.Get5(`type_by_reference`, types.Boolean_TRUE).(*types.BooleanValue).Bool()
	t.localReference = options.Get5(`local_reference`, types.Boolean_TRUE).(*types.BooleanValue).Bool()
	t.symbolAsString = options.Get5(`symbol_as_string`, types.Boolean_FALSE).(*types.BooleanValue).Bool()
	t.richData = options.Get5(`rich_data`, types.Boolean_TRUE).(*types.BooleanValue).Bool()
	t.messagePrefix = options.Get5(`message_prefix`, eval.EMPTY_STRING).String()
	return t
}

var typeKey = types.WrapString(PCORE_TYPE_KEY)
var valueKey = types.WrapString(PCORE_VALUE_KEY)
var defaultType = types.WrapString(string(PCORE_TYPE_DEFAULT))
var localRefSym = types.WrapString(PCORE_LOCAL_REF_SYMBOL)

func (t *ToDataConverter) Convert(value eval.Value) eval.Value {
	t.path = make([]eval.Value, 0, 16)
	t.values = make(map[eval.Value]interface{}, 63)
	return t.toData(value)
}

func (t *ToDataConverter) toJsonPath(path []eval.Value) string {
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
	return fmt.Sprintf(`%s%s`, t.messagePrefix, t.toJsonPath(t.path)[1:])
}

func (t *ToDataConverter) toData(value eval.Value) eval.Value {
	if value == eval.UNDEF || eval.IsInstance(types.DefaultDataType(), value) {
		return value
	}
	if value == types.WrapDefault() {
		if t.richData {
			return types.SingletonHash(typeKey, defaultType)
		}
		eval.Warning(eval.EVAL_SERIALIZATION_DEFAULT_CONVERTED_TO_STRING, issue.H{`path`: t.pathToString()})
		return types.WrapString(`default`)
	}

	if h, ok := value.(eval.OrderedMap); ok {
		return t.process(h, func() eval.Value {
			if h.AllPairs(func(k, v eval.Value) bool {
				if _, ok := k.(*types.StringValue); ok {
					return true
				}
				return false
			}) {
				result := make([]*types.HashEntry, 0, h.Len())
				h.EachPair(func(key, elem eval.Value) {
					t.with(key, func() eval.Value {
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

	if a, ok := value.(eval.List); ok {
		return t.process(a, func() eval.Value {
			result := make([]eval.Value, 0, a.Len())
			a.EachWithIndex(func(elem eval.Value, index int) {
				t.with(types.WrapInteger(int64(index)), func() eval.Value {
					elem = t.toData(elem)
					result = append(result, elem)
					return elem
				})
			})
			return types.WrapValues(result)
		})
	}

	if sv, ok := value.(*types.SensitiveValue); ok {
		return t.process(sv, func() eval.Value {
			return typeAndValueHash(types.WrapString(PCORE_TYPE_SENSITIVE), t.toData(sv.Unwrap()))
		})
	}
	return t.unknownToData(value)
}

func (t *ToDataConverter) withRecursiveGuard(value eval.Value, producer eval.Producer) eval.Value {
	if t.recursiveLock == nil {
		t.recursiveLock = map[eval.Value]bool{value: true}
	} else {
		if _, ok := t.recursiveLock[value]; ok {
			panic(eval.Error(eval.EVAL_SERIALIZATION_ENDLESS_RECURSION, issue.H{`type_name`: value.PType().Name()}))
		}
		t.recursiveLock[value] = true
	}
	v := producer()
	delete(t.recursiveLock, value)
	return v
}

func (t *ToDataConverter) unknownToData(value eval.Value) eval.Value {
	if t.richData {
		return t.valueToDataHash(value)
	}
	return t.unkownToStringWithWarning(value)
}

func (t *ToDataConverter) unkownToStringWithWarning(value eval.Value) eval.Value {
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
			klass = rt.PType().(*types.RuntimeType).Name()
		}
	} else {
		s = value.String()
		klass = value.PType().Name()
	}
	if warn {
		eval.Warning(eval.EVAL_SERIALIZATION_UNKNOWN_CONVERTED_TO_STRING, issue.H{`path`: t.pathToString(), `klass`: klass, `value`: s})
	}
	return types.WrapString(s)
}

func (t *ToDataConverter) process(value eval.Value, producer eval.Producer) eval.Value {
	if t.localReference {
		if ref, ok := t.values[value]; ok {
			if hash, ok := ref.(eval.OrderedMap); ok {
				return hash
			}
			jsonRef := t.toJsonPath(ref.([]eval.Value))
			if jsonRef == `` {
				// Complex key and hence no way to reference the prior value. The value must therefore be
				// duplicated which in turn introduces a risk for endless recursion in case of self
				// referencing structures
				return t.withRecursiveGuard(value, producer)
			}
			v := typeAndValueHash(localRefSym, types.WrapString(jsonRef))
			t.values[value] = v
			return v
		}
		t.values[value] = eval.CopyValues(t.path)
		return producer()
	}
	return t.withRecursiveGuard(value, producer)
}

func (t *ToDataConverter) with(key eval.Value, producer eval.Producer) eval.Value {
	t.path = append(t.path, key)
	value := producer()
	t.path = t.path[0 : len(t.path)-1]
	return value
}

func (t *ToDataConverter) nonStringKeyedHashToData(hash eval.OrderedMap) eval.Value {
	if t.richData {
		return t.toKeyExtendedHash(hash)
	}
	result := make([]*types.HashEntry, 0, hash.Len())
	hash.EachPair(func(key, elem eval.Value) {
		t.with(key, func() eval.Value {
			if _, ok := key.(*types.StringValue); !ok {
				key = t.unkownToStringWithWarning(key)
			}
			result = append(result, types.WrapHashEntry(key, t.toData(elem)))
			return eval.UNDEF
		})
	})
	return types.WrapHash(result)
}

func typeAndValueHash(t, v eval.Value) eval.OrderedMap {
	return types.WrapHash([]*types.HashEntry{types.WrapHashEntry(typeKey, t), types.WrapHashEntry(valueKey, v)})
}

func (t *ToDataConverter) valueToDataHash(value eval.Value) eval.Value {
	if rt, ok := value.(*types.RuntimeValue); ok {
		if !t.symbolAsString {
			if sym, ok := rt.Interface().(Symbol); ok {
				return typeAndValueHash(types.WrapString(PCORE_TYPE_SYMBOL), types.WrapString(string(sym)))
			}
		}
		return t.unkownToStringWithWarning(value)
	}

	vt := value.PType()
	if tx, ok := value.(eval.Type); ok {
		if ss, ok := value.(eval.SerializeAsString); ok {
			return typeAndValueHash(t.pcoreTypeToData(vt), types.WrapString(ss.SerializationString()))
		}
		if t.typeByReference {
			return typeAndValueHash(t.pcoreTypeToData(vt), types.WrapString(value.String()))
		}
		vt = tx.MetaType()
	}

	pcoreTv := t.pcoreTypeToData(vt)
	if ss, ok := value.(eval.SerializeAsString); ok {
		return typeAndValueHash(pcoreTv, types.WrapString(ss.SerializationString()))
	}


	if po, ok := value.(eval.PuppetObject); ok {
		ih := t.toData(po.InitHash()).(eval.OrderedMap)
		entries := make([]*types.HashEntry, 0, ih.Len())
		entries = append(entries, types.WrapHashEntry(typeKey, pcoreTv))
		ih.Each(func(v eval.Value) { entries = append(entries, v.(*types.HashEntry)) })
		return types.WrapHash(entries)
	}

	if ot, ok := vt.(eval.ObjectType); ok {
		ai := ot.AttributesInfo()
		attrs := ai.Attributes()
		args := make([]eval.Value, len(attrs))
		for i, a := range attrs {
			args[i] = a.Get(value)
		}

		for i := len(args) - 1; i >= ai.RequiredCount(); i-- {
			if !attrs[i].Default(args[i]) {
				break
			}
			args = args[:i]
		}
		entries := make([]*types.HashEntry, 0, len(attrs)+1)
		entries = append(entries, types.WrapHashEntry(typeKey, pcoreTv))
		for i, a := range args {
			key := types.WrapString(attrs[i].Name())
			t.with(key, func() eval.Value {
				a = t.toData(a)
				entries = append(entries, types.WrapHashEntry(key, a))
				return a
			})
		}
		return types.WrapHash(entries)
	}

	return t.unkownToStringWithWarning(value)
}

func (t *ToDataConverter) pcoreTypeToData(pcoreType eval.Type) eval.Value {
	typeName := pcoreType.Name()
	if t.typeByReference || strings.HasPrefix(typeName, `Pcore::`) {
		return types.WrapString(typeName)
	}
	return t.with(typeKey, func() eval.Value { return t.toData(pcoreType) })
}

func (t *ToDataConverter) toKeyExtendedHash(hash eval.OrderedMap) eval.Value {
	pairs := make([]eval.Value, 0, 2*hash.Len())
	hash.EachPair(func(key, value eval.Value) {
		key = t.toData(key)
		pairs = append(pairs, key, t.with(key, func() eval.Value { return t.toData(value) }))
	})
	return typeAndValueHash(types.WrapString(PCORE_TYPE_HASH), types.WrapValues(pairs))
}
