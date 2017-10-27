package types

import (
	. "io"

	. "github.com/puppetlabs/go-evaluator/evaluator"
)

type UnitType struct{}

func DefaultUnitType() *UnitType {
	return unitType_DEFAULT
}

func (t *UnitType) Equals(o interface{}, g Guard) bool {
	_, ok := o.(*UnitType)
	return ok
}

func (t *UnitType) IsAssignable(o PType, g Guard) (ok bool) {
	return true
}

func (t *UnitType) IsInstance(o PValue, g Guard) bool {
	return true
}

func (t *UnitType) Name() string {
	return `Unit`
}

func (t *UnitType) String() string {
	return `Unit`
}

func (t *UnitType) ToString(b Writer, s FormatContext, g RDetect) {
	TypeToString(t, b, s, g)
}

func (t *UnitType) Type() PType {
	return &TypeType{t}
}

var unitType_DEFAULT = &UnitType{}
