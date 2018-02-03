package types

import (
	"io"

	"github.com/puppetlabs/go-evaluator/eval"
)

type ScalarType struct{}

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

func (t *ScalarType) IsAssignable(o eval.PType, g eval.Guard) bool {
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

func (t *ScalarType) IsInstance(o eval.PValue, g eval.Guard) bool {
	switch o.(type) {
	// TODO: Add TimeSpanValue, TimestampValue, and VersionValue here
	case *BooleanValue, *FloatValue, *IntegerValue, *StringValue, *RegexpValue:
		return true
	}
	return false
}

func (t *ScalarType) Name() string {
	return `Scalar`
}

func (t *ScalarType) String() string {
	return `Scalar`
}

func (t *ScalarType) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *ScalarType) Type() eval.PType {
	return &TypeType{t}
}

var scalarType_DEFAULT = &ScalarType{}
