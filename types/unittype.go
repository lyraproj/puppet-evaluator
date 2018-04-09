package types

import (
	"io"

	"github.com/puppetlabs/go-evaluator/eval"
)

type UnitType struct{}

var Unit_Type eval.ObjectType

func init() {
	Unit_Type = newObjectType(`Pcore::UnitType`, `Pcore::AnyType{}`, func(ctx eval.Context, args []eval.PValue) eval.PValue {
		return DefaultUnitType()
	})

	newGoConstructor(`Unit`,
		func(d eval.Dispatch) {
			d.Param(`Any`)
			d.Function(func(c eval.Context, args []eval.PValue) eval.PValue {
				return args[0]
			})
		},
	)
}

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

func (t *UnitType) IsAssignable(o eval.PType, g eval.Guard) bool {
	return true
}

func (t *UnitType) IsInstance(c eval.Context, o eval.PValue, g eval.Guard) bool {
	return true
}

func (t *UnitType) MetaType() eval.ObjectType {
	return Unit_Type
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
