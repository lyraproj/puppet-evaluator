package types

import (
	"io"

	"github.com/lyraproj/puppet-evaluator/eval"
)

type ScalarType struct{}

var Scalar_Type eval.ObjectType

func init() {
	Scalar_Type = newObjectType(`Pcore::ScalarType`, `Pcore::AnyType{}`, func(ctx eval.Context, args []eval.Value) eval.Value {
		return DefaultScalarType()
	})
}

func DefaultScalarType() *ScalarType {
	return scalarType_DEFAULT
}

func (t *ScalarType) Accept(v eval.Visitor, g eval.Guard) {
	v(t)
}

func (t *ScalarType) Equals(o interface{}, g eval.Guard) bool {
	_, ok := o.(*ScalarType)
	return ok
}

func (t *ScalarType) IsAssignable(o eval.Type, g eval.Guard) bool {
	switch o.(type) {
	case *ScalarType, *ScalarDataType:
		return true
	default:
		return GuardedIsAssignable(stringType_DEFAULT, o, g) ||
			GuardedIsAssignable(numericType_DEFAULT, o, g) ||
			GuardedIsAssignable(booleanType_DEFAULT, o, g) ||
			GuardedIsAssignable(regexpType_DEFAULT, o, g)
	}
}

func (t *ScalarType) IsInstance(o eval.Value, g eval.Guard) bool {
	switch o.(type) {
	// TODO: Add TimeSpanValue, TimestampValue, and VersionValue here
	case *BooleanValue, *FloatValue, *IntegerValue, *StringValue, *RegexpValue:
		return true
	}
	return false
}

func (t *ScalarType) MetaType() eval.ObjectType {
	return Scalar_Type
}

func (t *ScalarType) Name() string {
	return `Scalar`
}

func (t *ScalarType)  CanSerializeAsString() bool {
  return true
}

func (t *ScalarType)  SerializationString() string {
	return t.String()
}


func (t *ScalarType) String() string {
	return `Scalar`
}

func (t *ScalarType) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *ScalarType) PType() eval.Type {
	return &TypeType{t}
}

var scalarType_DEFAULT = &ScalarType{}
