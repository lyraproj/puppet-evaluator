package types

import (
	"io"

	"github.com/puppetlabs/go-evaluator/eval"
)

type NumericType struct{}

var Numeric_Type eval.ObjectType

func init() {
	Numeric_Type = newObjectType(`Pcore::NumericType`, `Pcore::ScalarDataType {}`, func(ctx eval.EvalContext, args []eval.PValue) eval.PValue {
		return DefaultNumericType()
	})
}

func DefaultNumericType() *NumericType {
	return numericType_DEFAULT
}

func (t *NumericType) Accept(v eval.Visitor, g eval.Guard) {
	v(t)
}

func (t *NumericType) Equals(o interface{}, g eval.Guard) bool {
	_, ok := o.(*NumericType)
	return ok
}

func (t *NumericType) IsAssignable(o eval.PType, g eval.Guard) bool {
	switch o.(type) {
	case *IntegerType, *FloatType:
		return true
	default:
		return false
	}
}

func (t *NumericType) IsInstance(o eval.PValue, g eval.Guard) bool {
	switch o.Type().(type) {
	case *FloatType, *IntegerType:
		return true
	default:
		return false
	}
}

func (t *NumericType) MetaType() eval.ObjectType {
	return Numeric_Type
}

func (t *NumericType) Name() string {
	return `Numeric`
}

func (t *NumericType) String() string {
	return eval.ToString2(t, NONE)
}

func (t *NumericType) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *NumericType) Type() eval.PType {
	return &TypeType{t}
}

var numericType_DEFAULT = &NumericType{}
