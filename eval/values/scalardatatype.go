package values

import (
	. "io"

	. "github.com/puppetlabs/go-evaluator/eval/values/api"
)

type ScalarDataType struct{}

func DefaultScalarDataType() *ScalarDataType {
	return scalarDataType_DEFAULT
}

func (t *ScalarDataType) Equals(o interface{}, g Guard) bool {
	_, ok := o.(*ScalarDataType)
	return ok
}

func (t *ScalarDataType) IsAssignable(o PType, g Guard) bool {
	switch o.(type) {
	case *ScalarDataType:
		return true
	default:
		return GuardedIsAssignable(stringType_DEFAULT, o, g) ||
			GuardedIsAssignable(integerType_DEFAULT, o, g) ||
			GuardedIsAssignable(booleanType_DEFAULT, o, g) ||
			GuardedIsAssignable(floatType_DEFAULT, o, g)
	}
}

func (t *ScalarDataType) IsInstance(o PValue, g Guard) bool {
	switch o.(type) {
	case *BooleanValue, *FloatValue, *IntegerValue, *StringValue:
		return true
	}
	return false
}

func (t *ScalarDataType) Name() string {
	return `ScalarData`
}

func (t *ScalarDataType) String() string {
	return `ScalarData`
}

func (t *ScalarDataType) ToString(bld Writer, format FormatContext, g RDetect) {
	WriteString(bld, `ScalarData`)
}

func (t *ScalarDataType) Type() PType {
	return &TypeType{t}
}

var scalarDataType_DEFAULT = &ScalarDataType{}
