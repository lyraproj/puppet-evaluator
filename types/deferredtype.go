package types

import (
	"io"
	"reflect"

	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/utils"
)

var deferredMetaType eval.ObjectType

func init() {
	deferredMetaType = newGoObjectType(`DeferredType`, reflect.TypeOf(&DeferredType{}), `{}`)
}

type DeferredType struct {
	tn       string
	params   []eval.Value
	resolved eval.Type
}

func (dt *DeferredType) String() string {
	return eval.ToString(dt)
}

func (dt *DeferredType) Equals(other interface{}, guard eval.Guard) bool {
	if ot, ok := other.(*DeferredType); ok {
		return dt.tn == ot.tn
	}
	return false
}

func (dt *DeferredType) ToString(bld io.Writer, s eval.FormatContext, g eval.RDetect) {
	utils.WriteString(bld, `DeferredType(`)
	utils.WriteString(bld, dt.tn)
	if dt.params != nil {
		utils.WriteString(bld, `, `)
		WrapValues(dt.params).ToString(bld, s.Subsequent(), g)
	}
	utils.WriteByte(bld, ')')
}

func (dt *DeferredType) PType() eval.Type {
	return deferredMetaType
}

func (dt *DeferredType) Name() string {
	return dt.tn
}

func (dt *DeferredType) Resolve(c eval.Context) eval.Type {
	if dt.resolved == nil {
		if dt.params != nil {
			if dt.Name() == `TypeSet` && len(dt.params) == 1 {
				if ih, ok := dt.params[0].(eval.OrderedMap); ok {
					dt.resolved = newTypeSetType2(ih, c.Loader())
				}
			} else {
				ar := resolveValue(c, WrapValues(dt.params)).(*ArrayValue)
				dt.resolved = ResolveWithParams(c, dt.tn, ar.AppendTo(make([]eval.Value, 0, ar.Len())))
			}
		} else {
			dt.resolved = Resolve(c, dt.tn)
		}
	}
	return dt.resolved
}

func (dt *DeferredType) Parameters() []eval.Value {
	return dt.params
}

func resolveValue(c eval.Context, v eval.Value) (rv eval.Value) {
	switch v := v.(type) {
	case *DeferredType:
		rv = v.Resolve(c)
	case Deferred:
		rv = v.Resolve(c)
	case *ArrayValue:
		rv = v.Map(func(e eval.Value) eval.Value { return resolveValue(c, e) })
	case *HashEntry:
		rv = resolveEntry(c, v)
	case *HashValue:
		rv = v.MapEntries(func(he eval.MapEntry) eval.MapEntry { return resolveEntry(c, he) })
	default:
		rv = v
	}
	return
}

func resolveEntry(c eval.Context, he eval.MapEntry) eval.MapEntry {
	return WrapHashEntry(resolveValue(c, he.Key()), resolveValue(c, he.Value()))
}
