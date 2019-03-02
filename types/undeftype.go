package types

import (
	"io"

	"github.com/lyraproj/puppet-evaluator/utils"

	"reflect"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/puppet-evaluator/eval"
)

type (
	UndefType struct{}

	// UndefValue is an empty struct because both type and value are known
	UndefValue struct{}
)

var undefTypeDefault = &UndefType{}

var UndefMetaType eval.ObjectType

func init() {
	UndefMetaType = newObjectType(`Pcore::UndefType`, `Pcore::AnyType{}`,
		func(ctx eval.Context, args []eval.Value) eval.Value {
			return DefaultUndefType()
		})
}

func DefaultUndefType() *UndefType {
	return undefTypeDefault
}

func (t *UndefType) Accept(v eval.Visitor, g eval.Guard) {
	v(t)
}

func (t *UndefType) Equals(o interface{}, g eval.Guard) bool {
	_, ok := o.(*UndefType)
	return ok
}

func (t *UndefType) IsAssignable(o eval.Type, g eval.Guard) bool {
	_, ok := o.(*UndefType)
	return ok
}

func (t *UndefType) IsInstance(o eval.Value, g eval.Guard) bool {
	return o == undef
}

func (t *UndefType) MetaType() eval.ObjectType {
	return UndefMetaType
}

func (t *UndefType) Name() string {
	return `Undef`
}

func (t *UndefType) ReflectType(c eval.Context) (reflect.Type, bool) {
	return reflect.Value{}.Type(), true
}

func (t *UndefType) CanSerializeAsString() bool {
	return true
}

func (t *UndefType) SerializationString() string {
	return t.String()
}

func (t *UndefType) String() string {
	return `Undef`
}

func (t *UndefType) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *UndefType) PType() eval.Type {
	return &TypeType{t}
}

func WrapUndef() *UndefValue {
	return &UndefValue{}
}

func (uv *UndefValue) Equals(o interface{}, g eval.Guard) bool {
	_, ok := o.(*UndefValue)
	return ok
}

func (uv *UndefValue) Reflect(c eval.Context) reflect.Value {
	return reflect.Value{}
}

func (uv *UndefValue) ReflectTo(c eval.Context, value reflect.Value) {
	if !value.CanSet() {
		panic(eval.Error(eval.AttemptToSetUnsettable, issue.H{`kind`: value.Kind().String()}))
	}
	value.Set(reflect.Zero(value.Type()))
}

func (uv *UndefValue) String() string {
	return `undef`
}

func (uv *UndefValue) ToKey() eval.HashKey {
	return eval.HashKey([]byte{1, HkUndef})
}

func (uv *UndefValue) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	utils.WriteString(b, `undef`)
}

func (uv *UndefValue) PType() eval.Type {
	return DefaultUndefType()
}
