package types

import (
	"io"

	"github.com/lyraproj/puppet-evaluator/eval"
)

type VariantType struct {
	types []eval.Type
}

var VariantMetaType eval.ObjectType

func init() {
	VariantMetaType = newObjectType(`Pcore::VariantType`,
		`Pcore::AnyType {
	attributes => {
		types => Array[Type]
	}
}`, func(ctx eval.Context, args []eval.Value) eval.Value {
			return newVariantType2(args...)
		})
}

func DefaultVariantType() *VariantType {
	return variantTypeDefault
}

func NewVariantType(types ...eval.Type) eval.Type {
	switch len(types) {
	case 0:
		return DefaultVariantType()
	case 1:
		return types[0]
	default:
		return &VariantType{types}
	}
}

func newVariantType2(args ...eval.Value) eval.Type {
	return newVariantType3(WrapValues(args))
}

func newVariantType3(args eval.List) eval.Type {
	var variants []eval.Type
	var failIdx int

	switch args.Len() {
	case 0:
		return DefaultVariantType()
	case 1:
		first := args.At(0)
		switch first := first.(type) {
		case eval.Type:
			return first
		case *ArrayValue:
			return newVariantType3(first)
		default:
			panic(NewIllegalArgumentType(`Variant[]`, 0, `Type or Array[Type]`, args.At(0)))
		}
	default:
		variants, failIdx = toTypes(args)
		if failIdx >= 0 {
			panic(NewIllegalArgumentType(`Variant[]`, failIdx, `Type`, args.At(failIdx)))
		}
	}
	return &VariantType{variants}
}

func (t *VariantType) Accept(v eval.Visitor, g eval.Guard) {
	v(t)
	for _, c := range t.types {
		c.Accept(v, g)
	}
}

func (t *VariantType) Equals(o interface{}, g eval.Guard) bool {
	ot, ok := o.(*VariantType)
	return ok && len(t.types) == len(ot.types) && eval.GuardedIncludesAll(eval.EqSlice(t.types), eval.EqSlice(ot.types), g)
}

func (t *VariantType) Generic() eval.Type {
	return &VariantType{UniqueTypes(alterTypes(t.types, generalize))}
}

func (t *VariantType) Default() eval.Type {
	return variantTypeDefault
}

func (t *VariantType) IsAssignable(o eval.Type, g eval.Guard) bool {
	for _, v := range t.types {
		if GuardedIsAssignable(v, o, g) {
			return true
		}
	}
	return false
}

func (t *VariantType) IsInstance(o eval.Value, g eval.Guard) bool {
	for _, v := range t.types {
		if GuardedIsInstance(v, o, g) {
			return true
		}
	}
	return false
}

func (t *VariantType) MetaType() eval.ObjectType {
	return VariantMetaType
}

func (t *VariantType) Name() string {
	return `Variant`
}

func (t *VariantType) Parameters() []eval.Value {
	if len(t.types) == 0 {
		return eval.EmptyValues
	}
	ps := make([]eval.Value, len(t.types))
	for idx, t := range t.types {
		ps[idx] = t
	}
	return ps
}

func (t *VariantType) Resolve(c eval.Context) eval.Type {
	rts := make([]eval.Type, len(t.types))
	for i, ts := range t.types {
		rts[i] = resolve(c, ts)
	}
	t.types = rts
	return t
}

func (t *VariantType) CanSerializeAsString() bool {
	for _, v := range t.types {
		if !canSerializeAsString(v) {
			return false
		}
	}
	return true
}

func (t *VariantType) SerializationString() string {
	return t.String()
}

func (t *VariantType) String() string {
	return eval.ToString2(t, None)
}

func (t *VariantType) Types() []eval.Type {
	return t.types
}

func (t *VariantType) allAssignableTo(o eval.Type, g eval.Guard) bool {
	return allAssignableTo(t.types, o, g)
}

func (t *VariantType) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *VariantType) PType() eval.Type {
	return &TypeType{t}
}

var variantTypeDefault = &VariantType{types: []eval.Type{}}

func allAssignableTo(types []eval.Type, o eval.Type, g eval.Guard) bool {
	for _, v := range types {
		if !GuardedIsAssignable(o, v, g) {
			return false
		}
	}
	return true
}
