package types

import (
	"io"

	"github.com/puppetlabs/go-evaluator/eval"
)

type UnitType struct{}

var Unit_Type eval.ObjectType

func init() {
	Unit_Type = newObjectType(`Pcore::UnitType`, `Pcore::AnyType{}`, func(ctx eval.Context, args []eval.Value) eval.Value {
		return DefaultUnitType()
	})

	newGoConstructor(`Unit`,
		func(d eval.Dispatch) {
			d.Param(`Any`)
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
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

func (t *UnitType) IsAssignable(o eval.Type, g eval.Guard) bool {
	return true
}

func (t *UnitType) IsInstance(o eval.Value, g eval.Guard) bool {
	return true
}

func (t *UnitType) MetaType() eval.ObjectType {
	return Unit_Type
}

func (t *UnitType) Name() string {
	return `Unit`
}

func (t *UnitType)  CanSerializeAsString() bool {
  return true
}

func (t *UnitType)  SerializationString() string {
	return t.String()
}


func (t *UnitType) String() string {
	return `Unit`
}

func (t *UnitType) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *UnitType) PType() eval.Type {
	return &TypeType{t}
}

var unitType_DEFAULT = &UnitType{}
