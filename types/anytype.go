package types

import (
	"io"

	"github.com/lyraproj/puppet-evaluator/eval"
)

type AnyType struct{}

var Any_Type eval.ObjectType

func init() {
	eval.NewTypedName = NewTypedName
	eval.NewTypedName2 = newTypedName2
	eval.TypedNameFromMapKey = typedNameFromMapKey

	Any_Type = newObjectType(`Pcore::AnyType`, `{}`, func(ctx eval.Context, args []eval.Value) eval.Value {
		return DefaultAnyType()
	})
}

func DefaultAnyType() *AnyType {
	return anyType_DEFAULT
}

func (t *AnyType) Accept(v eval.Visitor, g eval.Guard) {
	v(t)
}

func (t *AnyType) Equals(o interface{}, g eval.Guard) bool {
	_, ok := o.(*AnyType)
	return ok
}

func (t *AnyType) IsAssignable(o eval.Type, g eval.Guard) bool {
	return true
}

func (t *AnyType) IsInstance(v eval.Value, g eval.Guard) bool {
	return true
}

func (t *AnyType) MetaType() eval.ObjectType {
	return Any_Type
}

func (t *AnyType) Name() string {
	return `Any`
}

func (t *AnyType)  CanSerializeAsString() bool {
  return true
}

func (t *AnyType)  SerializationString() string {
	return `Any`
}


func (t *AnyType) String() string {
	return `Any`
}

func (t *AnyType) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *AnyType) PType() eval.Type {
	return &TypeType{t}
}

var anyType_DEFAULT = &AnyType{}
