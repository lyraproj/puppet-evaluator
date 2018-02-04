package types

import (
	"io"

	"github.com/puppetlabs/go-evaluator/eval"
)

type UnitType struct{}

func DefaultUnitType() *UnitType {
	return unitType_DEFAULT
}

func (t *UnitType) Accept(v eval.Visitor, g eval.Guard) {
	v(t)
}

func (t *UnitType) Equals(o interface{}, g eval.Guard) bool {
	_, ok := o.(*UnitType)
	return ok
}

func (t *UnitType) IsAssignable(o eval.PType, g eval.Guard) (ok bool) {
	return true
}

func (t *UnitType) IsInstance(o eval.PValue, g eval.Guard) bool {
	return true
}

func (t *UnitType) Name() string {
	return `Unit`
}

func (t *UnitType) String() string {
	return `Unit`
}

func (t *UnitType) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *UnitType) Type() eval.PType {
	return &TypeType{t}
}

var unitType_DEFAULT = &UnitType{}
