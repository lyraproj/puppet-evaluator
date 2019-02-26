package types

import (
	"io"

	"github.com/lyraproj/puppet-evaluator/eval"
)

type ScalarDataType struct{}

var ScalarDataMetaType eval.ObjectType

func init() {
	ScalarDataMetaType = newObjectType(`Pcore::ScalarDataType`, `Pcore::ScalarType{}`, func(ctx eval.Context, args []eval.Value) eval.Value {
		return DefaultScalarDataType()
	})
}

func DefaultScalarDataType() *ScalarDataType {
	return scalarDataType_DEFAULT
}

func (t *ScalarDataType) Accept(v eval.Visitor, g eval.Guard) {
	v(t)
}

func (t *ScalarDataType) Equals(o interface{}, g eval.Guard) bool {
	_, ok := o.(*ScalarDataType)
	return ok
}

func (t *ScalarDataType) IsAssignable(o eval.Type, g eval.Guard) bool {
	switch o.(type) {
	case *ScalarDataType:
		return true
	default:
		return GuardedIsAssignable(stringTypeDefault, o, g) ||
			GuardedIsAssignable(integerTypeDefault, o, g) ||
			GuardedIsAssignable(booleanTypeDefault, o, g) ||
			GuardedIsAssignable(floatTypeDefault, o, g)
	}
}

func (t *ScalarDataType) IsInstance(o eval.Value, g eval.Guard) bool {
	switch o.(type) {
	case booleanValue, floatValue, integerValue, stringValue:
		return true
	}
	return false
}

func (t *ScalarDataType) MetaType() eval.ObjectType {
	return ScalarDataMetaType
}

func (t *ScalarDataType) Name() string {
	return `ScalarData`
}

func (t *ScalarDataType) CanSerializeAsString() bool {
	return true
}

func (t *ScalarDataType) SerializationString() string {
	return t.String()
}

func (t *ScalarDataType) String() string {
	return `ScalarData`
}

func (t *ScalarDataType) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *ScalarDataType) PType() eval.Type {
	return &TypeType{t}
}

var scalarDataType_DEFAULT = &ScalarDataType{}
