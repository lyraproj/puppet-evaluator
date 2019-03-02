package serialization

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/types"
)

const NoDedup = 0
const NoKeyDedup = 1
const MaxDedup = 2

// Serializer is a re-entrant fully configured serializer that streams the given
// value to the given consumer.
type Serializer interface {
	// Convert the given RichData value to a series of Data values streamed to the
	// given consumer.
	Convert(value eval.Value, consumer eval.ValueConsumer)
}

type rdSerializer struct {
	context       eval.Context
	richData      bool
	messagePrefix string
	dedupLevel    int
}

type context struct {
	config     *rdSerializer
	values     map[eval.Value]int
	path       []eval.Value
	refIndex   int
	dedupLevel int
	consumer   eval.ValueConsumer
}

// NewSerializer returns a new Serializer
func NewSerializer(ctx eval.Context, options eval.OrderedMap) Serializer {
	t := &rdSerializer{context: ctx}
	t.richData = options.Get5(`rich_data`, types.BooleanTrue).(eval.BooleanValue).Bool()
	t.messagePrefix = options.Get5(`message_prefix`, eval.EmptyString).String()
	if !options.Get5(`local_reference`, types.BooleanTrue).(eval.BooleanValue).Bool() {
		// local_reference explicitly set to false
		t.dedupLevel = NoDedup
	} else {
		t.dedupLevel = int(options.Get5(`dedup_level`, types.WrapInteger(MaxDedup)).(eval.IntegerValue).Int())
	}
	return t
}

var typeKey = types.WrapString(PcoreTypeKey)
var valueKey = types.WrapString(PcoreValueKey)
var defaultType = types.WrapString(PcoreTypeDefault)
var binaryType = types.WrapString(PcoreTypeBinary)
var sensitiveType = types.WrapString(PcoreTypeSensitive)
var hashKey = types.WrapString(PcoreTypeHash)

func (t *rdSerializer) Convert(value eval.Value, consumer eval.ValueConsumer) {
	c := context{config: t, values: make(map[eval.Value]int, 63), refIndex: 0, consumer: consumer, path: make([]eval.Value, 0, 16), dedupLevel: t.dedupLevel}
	if c.dedupLevel >= MaxDedup && !consumer.CanDoComplexKeys() {
		c.dedupLevel = NoKeyDedup
	}
	c.toData(1, value)
}

func (sc *context) pathToString() string {
	s := bytes.NewBufferString(sc.config.messagePrefix)
	for _, v := range sc.path {
		if s.Len() > 0 {
			s.WriteByte('/')
		}
		if v == nil {
			s.WriteString(`null`)
		} else if eval.IsInstance(types.DefaultScalarType(), v) {
			v.ToString(s, types.Program, nil)
		} else {
			s.WriteString(issue.Label(s))
		}
	}
	return s.String()
}

func (sc *context) toData(level int, value eval.Value) {
	if value == nil {
		sc.addData(eval.Undef)
		return
	}

	switch value := value.(type) {
	case *types.UndefValue, eval.IntegerValue, eval.FloatValue, eval.BooleanValue:
		// Never dedup
		sc.addData(value)
	case eval.StringValue:
		// Dedup only if length exceeds stringThreshold
		key := value.String()
		if sc.dedupLevel >= level && len(key) >= sc.consumer.StringDedupThreshold() {
			sc.process(value, func() {
				sc.addData(value)
			})
		} else {
			sc.addData(value)
		}
	case *types.DefaultValue:
		if sc.config.richData {
			sc.addHash(1, func() {
				sc.toData(2, typeKey)
				sc.toData(1, defaultType)
			})
		} else {
			eval.LogWarning(eval.SerializationDefaultConvertedToString, issue.H{`path`: sc.pathToString()})
			sc.toData(1, types.WrapString(`default`))
		}
	case *types.HashValue:
		if sc.consumer.CanDoComplexKeys() || value.AllKeysAreStrings() {
			sc.process(value, func() {
				sc.addHash(value.Len(), func() {
					value.EachPair(func(key, elem eval.Value) {
						sc.toData(2, key)
						sc.withPath(key, func() { sc.toData(1, elem) })
					})
				})
			})
		} else {
			sc.nonStringKeyedHashToData(value)
		}
	case *types.ArrayValue:
		sc.process(value, func() {
			sc.addArray(value.Len(), func() {
				value.EachWithIndex(func(elem eval.Value, idx int) {
					sc.withPath(types.WrapInteger(int64(idx)), func() { sc.toData(1, elem) })
				})
			})
		})
	case *types.SensitiveValue:
		sc.process(value, func() {
			if sc.config.richData {
				sc.addHash(2, func() {
					sc.toData(2, typeKey)
					sc.toData(1, sensitiveType)
					sc.toData(2, valueKey)
					sc.withPath(valueKey, func() { sc.toData(1, value.Unwrap()) })
				})
			} else {
				sc.unknownToStringWithWarning(level, value)
			}
		})
	case *types.BinaryValue:
		sc.process(value, func() {
			if sc.consumer.CanDoBinary() {
				sc.addData(value)
			} else {
				if sc.config.richData {
					sc.addHash(2, func() {
						sc.toData(2, typeKey)
						sc.toData(1, binaryType)
						sc.toData(2, valueKey)
						sc.toData(1, types.WrapString(value.SerializationString()))
					})
				} else {
					sc.unknownToStringWithWarning(level, value)
				}
			}
		})
	default:
		if sc.config.richData {
			sc.valueToDataHash(value)
		} else {
			sc.unknownToStringWithWarning(1, value)
		}
	}
}

func (sc *context) unknownToStringWithWarning(level int, value eval.Value) {
	var klass string
	var s string
	if rt, ok := value.(*types.RuntimeValue); ok {
		s = fmt.Sprintf(`%v`, rt.Interface())
		klass = rt.PType().(*types.RuntimeType).Name()
	} else {
		s = value.String()
		klass = value.PType().Name()
	}
	eval.LogWarning(eval.SerializationUnknownConvertedToString, issue.H{`path`: sc.pathToString(), `klass`: klass, `value`: s})
	sc.toData(level, types.WrapString(s))
}

func (sc *context) withPath(p eval.Value, doer eval.Doer) {
	sc.path = append(sc.path, p)
	doer()
	sc.path = sc.path[0 : len(sc.path)-1]
}

func (sc *context) process(value eval.Value, doer eval.Doer) {
	if sc.dedupLevel == NoDedup {
		doer()
		return
	}

	if ref, ok := sc.values[value]; ok {
		sc.consumer.AddRef(ref)
	} else {
		sc.values[value] = sc.refIndex
		doer()
	}
}

func (sc *context) nonStringKeyedHashToData(hash eval.OrderedMap) {
	if sc.config.richData {
		sc.toKeyExtendedHash(hash)
		return
	}
	sc.process(hash, func() {
		sc.addHash(hash.Len(), func() {
			hash.EachPair(func(key, elem eval.Value) {
				if s, ok := key.(eval.StringValue); ok {
					sc.toData(2, s)
				} else {
					sc.unknownToStringWithWarning(2, key)
				}
				sc.withPath(key, func() { sc.toData(1, elem) })
			})
		})
	})
}

func (sc *context) addArray(len int, doer eval.Doer) {
	sc.refIndex++
	sc.consumer.AddArray(len, doer)
}

func (sc *context) addHash(len int, doer eval.Doer) {
	sc.refIndex++
	sc.consumer.AddHash(len, doer)
}

func (sc *context) addData(v eval.Value) {
	sc.refIndex++
	sc.consumer.Add(v)
}

func (sc *context) valueToDataHash(value eval.Value) {
	if _, ok := value.(*types.RuntimeValue); ok {
		sc.unknownToStringWithWarning(1, value)
		return
	}

	switch value := value.(type) {
	case *types.TypeAliasType:
		if sc.isKnownType(value.Name()) {
			sc.process(value, func() {
				sc.addHash(2, func() {
					sc.toData(2, typeKey)
					sc.toData(2, types.WrapString(`Type`))
					sc.toData(2, valueKey)
					sc.toData(1, types.WrapString(value.Name()))
				})
			})
			return
		}
	case eval.ObjectType:
		tv := value.(eval.ObjectType)
		if sc.isKnownType(tv.Name()) {
			sc.process(value, func() {
				sc.addHash(2, func() {
					sc.toData(2, typeKey)
					sc.toData(2, types.WrapString(`Type`))
					sc.toData(2, valueKey)
					sc.toData(1, types.WrapString(tv.String()))
				})
			})
			return
		}
	}

	vt := value.PType()
	if tx, ok := value.(eval.Type); ok {
		if ss, ok := value.(eval.SerializeAsString); ok && ss.CanSerializeAsString() {
			sc.process(value, func() {
				sc.addHash(2, func() {
					sc.toData(2, typeKey)
					sc.withPath(typeKey, func() { sc.pcoreTypeToData(vt) })
					sc.toData(2, valueKey)
					sc.toData(1, types.WrapString(ss.SerializationString()))
				})
			})
			return
		}
		vt = tx.MetaType()
	}

	if ss, ok := value.(eval.SerializeAsString); ok && ss.CanSerializeAsString() {
		sc.process(value, func() {
			sc.addHash(2, func() {
				sc.toData(2, typeKey)
				sc.withPath(typeKey, func() { sc.pcoreTypeToData(vt) })
				sc.toData(2, valueKey)
				sc.toData(1, types.WrapString(ss.SerializationString()))
			})
		})
		return
	}

	if po, ok := value.(eval.PuppetObject); ok {
		sc.process(value, func() {
			sc.addHash(2, func() {
				sc.toData(2, typeKey)
				sc.withPath(typeKey, func() { sc.pcoreTypeToData(vt) })
				po.InitHash().EachPair(func(k, v eval.Value) {
					sc.toData(2, k) // No need to convert key. It's always a string
					sc.withPath(k, func() { sc.toData(1, v) })
				})
			})
		})
		return
	}

	if ot, ok := vt.(eval.ObjectType); ok {
		sc.process(value, func() {
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
			sc.addHash(1+len(args), func() {
				sc.toData(2, typeKey)
				sc.withPath(typeKey, func() { sc.pcoreTypeToData(vt) })
				for i, a := range args {
					k := types.WrapString(attrs[i].Name())
					sc.toData(2, k)
					sc.withPath(k, func() { sc.toData(1, a) })
				}
			})
		})
		return
	}
	sc.unknownToStringWithWarning(1, value)
}

func (sc *context) isKnownType(typeName string) bool {
	if strings.HasPrefix(typeName, `Pcore::`) {
		return true
	}
	_, found := eval.Load(sc.config.context, eval.NewTypedName(eval.NsType, typeName))
	return found
}

func (sc *context) pcoreTypeToData(pcoreType eval.Type) {
	typeName := pcoreType.Name()
	if sc.isKnownType(typeName) {
		sc.toData(1, types.WrapString(typeName))
	} else {
		sc.toData(1, pcoreType)
	}
}

func (sc *context) toKeyExtendedHash(hash eval.OrderedMap) {
	sc.process(hash, func() {
		sc.addHash(2, func() {
			sc.toData(2, typeKey)
			sc.toData(1, hashKey)
			sc.toData(2, valueKey)
			sc.addArray(hash.Len()*2, func() {
				hash.EachPair(func(key, value eval.Value) {
					sc.toData(1, key)
					sc.withPath(key, func() { sc.toData(1, value) })
				})
			})
		})
	})
}
