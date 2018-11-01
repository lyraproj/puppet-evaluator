package types

import (
	"io"

	"github.com/puppetlabs/go-evaluator/eval"
)

type VariantType struct {
	types []eval.Type
}

var Variant_Type eval.ObjectType

func init() {
	Variant_Type = newObjectType(`Pcore::VariantType`,
		`Pcore::AnyType {
	attributes => {
		types => Array[Type]
	}
}`)
}

func DefaultVariantType() *VariantType {
	return variantType_DEFAULT
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

func NewVariantType2(args ...eval.Value) eval.Type {
	return NewVariantType3(WrapArray(args))
}

func NewVariantType3(args eval.List) eval.Type {
	var variants []eval.Type
	var failIdx int

	switch args.Len() {
	case 0:
		return DefaultVariantType()
	case 1:
		first := args.At(0)
		switch first.(type) {
		case eval.Type:
			return first.(eval.Type)
		case *ArrayValue:
			return NewVariantType3(first.(*ArrayValue))
		default:
			panic(NewIllegalArgumentType2(`Variant[]`, 0, `Type or Array[Type]`, args.At(0)))
		}
	default:
		variants, failIdx = toTypes(args)
		if failIdx >= 0 {
			panic(NewIllegalArgumentType2(`Variant[]`, failIdx, `Type`, args.At(failIdx)))
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
	return variantType_DEFAULT
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
	return Variant_Type
}

func (t *VariantType) Name() string {
	return `Variant`
}

func (t *VariantType) Parameters() []eval.Value {
	if len(t.types) == 0 {
		return eval.EMPTY_VALUES
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

func (t *VariantType) String() string {
	return eval.ToString2(t, NONE)
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

var variantType_DEFAULT = &VariantType{types: []eval.Type{}}

func allAssignableTo(types []eval.Type, o eval.Type, g eval.Guard) bool {
	for _, v := range types {
		if !GuardedIsAssignable(o, v, g) {
			return false
		}
	}
	return true
}
