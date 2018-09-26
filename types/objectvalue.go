package types

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-issues/issue"
	"io"
)

func AllocObjectValue(typ eval.ObjectType) eval.ObjectValue {
	if typ.IsMetaType() {
		return AllocObjectType()
	}
	return &objectValue{typ, eval.EMPTY_VALUES}
}

func (ov *objectValue) Initialize(c eval.Context, values []eval.PValue) {
	if len(values) > 0 && ov.typ.IsParameterized() {
		ov.InitFromHash(c, makeValueHash(ov.typ.AttributesInfo(), values))
		return
	}
	fillValueSlice(values, ov.typ.AttributesInfo().Attributes())
	ov.values = values
}

func (ov *objectValue) InitFromHash(c eval.Context, hash eval.KeyedValue) {
	typ := ov.typ.(*objectType)
	va := typ.AttributesInfo().PositionalFromHash(hash)
	if len(va) > 0 && typ.IsParameterized() {
		params := make([]*HashEntry, 0)
		typ.typeParameters(true).EachPair(func(k string, v interface{}) {
			if pv, ok := hash.Get4(k); ok && eval.IsInstance(v.(*typeParameter).typ, pv) {
				params = append(params, WrapHashEntry2(k, pv))
			}
		})
		if len(params) > 0 {
			ov.typ = NewObjectTypeExtension(c, typ, []eval.PValue{WrapHash(params)})
		}
	}
	ov.values = va
}

func NewObjectValue(c eval.Context, typ eval.ObjectType, values []eval.PValue) eval.ObjectValue {
	ov := AllocObjectValue(typ)
	ov.Initialize(c, values)
	return ov
}

func NewObjectValue2(c eval.Context, typ eval.ObjectType, hash *HashValue) eval.ObjectValue {
	ov := AllocObjectValue(typ)
	ov.InitFromHash(c, hash)
	return ov
}

// Ensure that all entries in the value slice that are nil receive default values from the given attributes
func fillValueSlice(values []eval.PValue, attrs []eval.Attribute) {
	for ix, v := range values {
		if v == nil {
			at := attrs[ix]
			if !at.HasValue() {
				panic(eval.Error(eval.EVAL_MISSING_REQUIRED_ATTRIBUTE, issue.H{`label`: at.Label()}))
			}
			values[ix] = at.Value()
		}
	}
}

func (o *objectValue) Get(key string) (eval.PValue, bool) {
	pi := o.typ.AttributesInfo()
	if idx, ok := pi.NameToPos()[key]; ok {
		if idx < len(o.values) {
			return o.values[idx], ok
		}
		return pi.Attributes()[idx].Value(), ok
	}
	return nil, false
}

func (o *objectValue) Equals(other interface{}, g eval.Guard) bool {
	if ov, ok := other.(*objectValue); ok {
		return o.typ.Equals(ov.typ, g) && eval.GuardedEquals(o.values, ov.values, g)
	}
	return false
}

func (o *objectValue) String() string {
	return eval.ToString(o)
}

func (o *objectValue) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	ObjectToString(o, s, b, g)
}

func (o *objectValue) Type() eval.PType {
	return o.typ
}

func (o *objectValue) InitHash() eval.KeyedValue {
	return makeValueHash(o.typ.AttributesInfo(), o.values)
}

// Turn a positional argument list into a hash. The hash will exclude all values
// that are equal to the default value of the corresponding attribute
func makeValueHash(pi eval.AttributesInfo, values []eval.PValue) *HashValue {
	posToName := pi.PosToName()
	entries := make([]*HashEntry, 0, len(posToName))
	for i, v := range values {
		attr := pi.Attributes()[i]
		if !(attr.HasValue() && eval.Equals(v, attr.Value())) {
			entries = append(entries, WrapHashEntry2(attr.Name(), v))
		}
	}
	return WrapHash(entries)
}
