package serialization

import (
	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/types"
	"reflect"
)

type dsContext struct {
	collector
	allowUnresolved bool
	context         eval.Context
	newTypes        []eval.Type
	value           eval.Value
	ntUnique        map[uintptr]bool
	converted       map[uintptr]eval.Value
}

// NewDeserializer creates a new Collector that consumes input and creates a RichData Value
func NewDeserializer(ctx eval.Context, options eval.OrderedMap) Collector {
	ds := &dsContext{
		context:         ctx,
		newTypes:        make([]eval.Type, 0, 11),
		ntUnique:        make(map[uintptr]bool, 11),
		converted:       make(map[uintptr]eval.Value, 11),
		allowUnresolved: options.Get5(`allow_unresolved`, types.Boolean_FALSE).(*types.BooleanValue).Bool()}
	ds.Init()
	return ds
}

func (ds *dsContext) Value() eval.Value {
	if ds.value == nil {
		ds.value = ds.convert(ds.collector.Value())
		ds.context.AddTypes(ds.newTypes...)
	}
	return ds.value
}

func (ds *dsContext) convert(value eval.Value) eval.Value {
	key := reflect.ValueOf(value).Pointer()
	if cv, ok := ds.converted[key]; ok {
		return cv
	}

	if hash, ok := value.(*types.HashValue); ok {
		if hash.AllKeysAreStrings() {
			if pcoreType, ok := hash.Get4(PcoreTypeKey); ok {
				switch pcoreType.String() {
				case PcoreTypeHash:
					return ds.convertHash(hash, key)
				case PcoreTypeSensitive:
					return ds.convertSensitive(hash, key)
				case PcoreTypeDefault:
					return types.WrapDefault()
				default:
					v := ds.convertOther(hash, key, pcoreType)
					switch v.(type) {
					case eval.ObjectType, eval.TypeSet, *types.TypeAliasType:
						// Ensure that type is made known to current loader
						rt := v.(eval.ResolvableType)
						tn := eval.NewTypedName(eval.NsType, rt.Name())
						if lt, ok := eval.Load(ds.context, tn); ok && lt != rt {
							t := rt.Resolve(ds.context)
							if t.Equals(lt, nil) {
								return lt.(eval.Value)
							}
							panic(eval.Error(eval.EVAL_ATTEMPT_TO_REDEFINE, issue.H{`name`: tn}))
						}
						ds.newTypes = append(ds.newTypes, rt)
					}
					return v
				}
			}
		}

		return types.BuildHash(hash.Len(), func(h *types.HashValue, entries []*types.HashEntry) []*types.HashEntry {
			ds.converted[key] = h
			hash.EachPair(func(k, v eval.Value) {
				entries = append(entries, types.WrapHashEntry(ds.convert(k), ds.convert(v)))
			})
			return entries
		})
	}

	if array, ok := value.(*types.ArrayValue); ok {
		return types.BuildArray(array.Len(), func(a *types.ArrayValue, elements []eval.Value) []eval.Value {
			ds.converted[key] = a
			array.Each(func(v eval.Value) { elements = append(elements, ds.convert(v)) })
			return elements
		})
	}
	return value
}

func (ds *dsContext) convertHash(hash eval.OrderedMap, key uintptr) eval.Value {
	value := hash.Get5(PcoreValueKey, eval.EMPTY_ARRAY).(eval.List)
	return types.BuildHash(value.Len(), func(hash *types.HashValue, entries []*types.HashEntry) []*types.HashEntry {
		ds.converted[key] = hash
		for idx := 0; idx < value.Len(); idx += 2 {
			entries = append(entries, types.WrapHashEntry(ds.convert(value.At(idx)), ds.convert(value.At(idx+1))))
		}
		return entries
	})
}

func (ds *dsContext) convertSensitive(hash eval.OrderedMap, key uintptr) eval.Value {
	cv := types.WrapSensitive(ds.convert(hash.Get5(PcoreValueKey, eval.UNDEF)))
	ds.converted[key] = cv
	return cv
}

func (ds *dsContext) convertOther(hash eval.OrderedMap, key uintptr, typeValue eval.Value) eval.Value {
	value := hash.Get6(PcoreValueKey, func() eval.Value {
		return hash.RejectPairs(func(k, v eval.Value) bool {
			if s, ok := k.(*types.StringValue); ok {
				return s.String() == PcoreTypeKey
			}
			return false
		})
	})
	if typeHash, ok := typeValue.(*types.HashValue); ok {
		typ := ds.convert(typeHash)
		if _, ok := typ.(*types.HashValue); ok {
			if !ds.allowUnresolved {
				panic(eval.Error(eval.EVAL_UNABLE_TO_DESERIALIZE_TYPE, issue.H{`hash`: typ.String()}))
			}
			return hash
		}
		return ds.pcoreTypeHashToValue(typ.(eval.Type), key, value)
	}
	typ := ds.context.ParseType(typeValue)
	if tr, ok := typ.(*types.TypeReferenceType); ok {
		if !ds.allowUnresolved {
			panic(eval.Error(eval.EVAL_UNRESOLVED_TYPE, issue.H{`typeString`: tr.String()}))
		}
		return hash
	}
	return ds.pcoreTypeHashToValue(typ.(eval.Type), key, value)
}

func (ds *dsContext) pcoreTypeHashToValue(typ eval.Type, key uintptr, value eval.Value) eval.Value {
	var ov eval.Value

	if hash, ok := value.(*types.HashValue); ok {
		if ov, ok = ds.allocate(typ); ok {
			ds.converted[key] = ov
			ov.(eval.Object).InitFromHash(ds.context, ds.convert(hash).(*types.HashValue))
			return ov
		}

		hash = ds.convert(hash).(*types.HashValue)
		if ot, ok := typ.(eval.ObjectType); ok {
			if ot.HasHashConstructor() {
				ov = eval.New(ds.context, typ, hash)
			} else {
				ov = eval.New(ds.context, typ, ot.AttributesInfo().PositionalFromHash(hash)...)
			}
		} else {
			ov = eval.New(ds.context, typ, hash)
		}
	} else {
		if str, ok := value.(*types.StringValue); ok {
			ov = eval.New(ds.context, typ, str)
		} else {
			panic(eval.Error(eval.EVAL_UNABLE_TO_DESERIALIZE_VALUE, issue.H{`type`: typ.Name(), `arg_type`: value.PType().Name()}))
		}
	}
	ds.converted[key] = ov
	return ov
}

func (ds *dsContext) allocate(typ eval.Type) (eval.Object, bool) {
	if allocator, ok := eval.Load(ds.context, eval.NewTypedName(eval.NsAllocator, typ.Name())); ok {
		return allocator.(eval.Lambda).Call(nil, nil).(eval.Object), true
	}
	if ot, ok := typ.(eval.ObjectType); ok && ot.Name() == `Pcore::ObjectType` {
		return types.AllocObjectType(), true
	}
	return nil, false
}
