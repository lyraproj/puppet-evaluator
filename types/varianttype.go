package types

import (
	"io"

	"github.com/puppetlabs/go-evaluator/eval"
)

type VariantType struct {
	types []eval.PType
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

func NewVariantType(types ...eval.PType) eval.PType {
	switch len(types) {
	case 0:
		return DefaultVariantType()
	case 1:
		return types[0]
	default:
		return &VariantType{types}
	}
}

func NewVariantType2(args ...eval.PValue) eval.PType {
	return NewVariantType3(WrapArray(args))
}

func NewVariantType3(args eval.IndexedValue) eval.PType {
	var variants []eval.PType
	var failIdx int

	switch args.Len() {
	case 0:
		return DefaultVariantType()
	case 1:
		first := args.At(0)
		switch first.(type) {
		case eval.PType:
			return first.(eval.PType)
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

func (t *VariantType) Generic() eval.PType {
	return &VariantType{UniqueTypes(alterTypes(t.types, generalize))}
}

func (t *VariantType) Default() eval.PType {
	return variantType_DEFAULT
}

func (t *VariantType) IsAssignable(o eval.PType, g eval.Guard) bool {
	for _, v := range t.types {
		if GuardedIsAssignable(v, o, g) {
			return true
		}
	}
	return false
}

func (t *VariantType) IsInstance(o eval.PValue, g eval.Guard) bool {
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

func (t *VariantType) Parameters() []eval.PValue {
	if len(t.types) == 0 {
		return eval.EMPTY_VALUES
	}
	ps := make([]eval.PValue, len(t.types))
	for idx, t := range t.types {
		ps[idx] = t
	}
	return ps
}

func (t *VariantType) Resolve(c eval.Context) eval.PType {
	rts := make([]eval.PType, len(t.types))
	for i, ts := range t.types {
		rts[i] = resolve(c, ts)
	}
	t.types = rts
	return t
}

func (t *VariantType) String() string {
	return eval.ToString2(t, NONE)
}

func (t *VariantType) Types() []eval.PType {
	return t.types
}

func (t *VariantType) allAssignableTo(o eval.PType, g eval.Guard) bool {
	return allAssignableTo(t.types, o, g)
}

func (t *VariantType) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *VariantType) Type() eval.PType {
	return &TypeType{t}
}

var variantType_DEFAULT = &VariantType{types: []eval.PType{}}

func allAssignableTo(types []eval.PType, o eval.PType, g eval.Guard) bool {
	for _, v := range types {
		if !GuardedIsAssignable(o, v, g) {
			return false
		}
	}
	return true
}
