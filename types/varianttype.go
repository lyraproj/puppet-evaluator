package types

import (
	. "io"

	. "github.com/puppetlabs/go-evaluator/evaluator"
)

type VariantType struct {
	types []PType
}

func DefaultVariantType() *VariantType {
	return variantType_DEFAULT
}

func NewVariantType(types []PType) PType {
	switch len(types) {
	case 0:
		return DefaultVariantType()
	case 1:
		return types[0]
	default:
		return &VariantType{types}
	}
}

func NewVariantType2(args ...PValue) PType {
	var variants []PType
	var failIdx int

	switch len(args) {
	case 0:
		return DefaultVariantType()
	case 1:
		first := args[0]
		switch first.(type) {
		case PType:
			return first.(PType)
		case *ArrayValue:
			return NewVariantType2(first.(*ArrayValue).Elements()...)
		default:
			panic(NewIllegalArgumentType2(`Variant[]`, 0, `Type or Array[Type]`, args[0]))
		}
	default:
		variants, failIdx = toTypes(args...)
		if failIdx >= 0 {
			panic(NewIllegalArgumentType2(`Variant[]`, failIdx, `Type`, args[failIdx]))
		}
	}
	return &VariantType{variants}
}

func (t *VariantType) Equals(o interface{}, g Guard) bool {
	ot, ok := o.(*VariantType)
	return ok && len(t.types) == len(ot.types) && GuardedIncludesAll(EqSlice(t.types), EqSlice(ot.types), g)
}

func (t *VariantType) Generic() PType {
	return &VariantType{UniqueTypes(alterTypes(t.types, generalize))}
}

func (t *VariantType) IsAssignable(o PType, g Guard) bool {
	for _, v := range t.types {
		if GuardedIsAssignable(v, o, g) {
			return true
		}
	}
	return false
}

func (t *VariantType) IsInstance(o PValue, g Guard) bool {
	for _, v := range t.types {
		if GuardedIsInstance(v, o, g) {
			return true
		}
	}
	return false
}

func (t *VariantType) Name() string {
	return `Variant`
}

func (t *VariantType) Parameters() []PValue {
	if len(t.types) == 0 {
		return EMPTY_VALUES
	}
	ps := make([]PValue, len(t.types))
	for idx, t := range t.types {
		ps[idx] = t
	}
	return ps
}

func (t *VariantType) String() string {
	return ToString2(t, NONE)
}

func (t *VariantType) Types() []PType {
	return t.types
}

func (t *VariantType) allAssignableTo(o PType, g Guard) bool {
	return allAssignableTo(t.types, o, g)
}

func (t *VariantType) ToString(b Writer, s FormatContext, g RDetect) {
	TypeToString(t, b, s, g)
}

func (t *VariantType) Type() PType {
	return &TypeType{t}
}

var variantType_DEFAULT = &VariantType{types: []PType{}}

func allAssignableTo(types []PType, o PType, g Guard) bool {
	for _, v := range types {
		if !GuardedIsAssignable(o, v, g) {
			return false
		}
	}
	return true
}
