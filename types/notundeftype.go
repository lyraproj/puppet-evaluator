package types

import (
	"io"

	"reflect"

	"github.com/lyraproj/puppet-evaluator/errors"
	"github.com/lyraproj/puppet-evaluator/eval"
)

type NotUndefType struct {
	typ eval.Type
}

var NotUndefMetaType eval.ObjectType

func init() {
	NotUndefMetaType = newObjectType(`Pcore::NotUndefType`,
		`Pcore::AnyType {
			attributes => {
				type => {
					type => Optional[Type],
					value => Any
				},
			}
		}`, func(ctx eval.Context, args []eval.Value) eval.Value {
			return newNotUndefType2(args...)
		})
}

func DefaultNotUndefType() *NotUndefType {
	return notUndefTypeDefault
}

func NewNotUndefType(containedType eval.Type) *NotUndefType {
	if containedType == nil || containedType == anyTypeDefault {
		return DefaultNotUndefType()
	}
	return &NotUndefType{containedType}
}

func newNotUndefType2(args ...eval.Value) *NotUndefType {
	switch len(args) {
	case 0:
		return DefaultNotUndefType()
	case 1:
		if containedType, ok := args[0].(eval.Type); ok {
			return NewNotUndefType(containedType)
		}
		if containedType, ok := args[0].(stringValue); ok {
			return newNotUndefType3(string(containedType))
		}
		panic(NewIllegalArgumentType(`NotUndef[]`, 0, `Variant[Type,String]`, args[0]))
	default:
		panic(errors.NewIllegalArgumentCount(`NotUndef[]`, `0 - 1`, len(args)))
	}
}

func newNotUndefType3(str string) *NotUndefType {
	return &NotUndefType{NewStringType(nil, str)}
}

func (t *NotUndefType) Accept(v eval.Visitor, g eval.Guard) {
	v(t)
	t.typ.Accept(v, g)
}

func (t *NotUndefType) ContainedType() eval.Type {
	return t.typ
}

func (t *NotUndefType) Default() eval.Type {
	return notUndefTypeDefault
}

func (t *NotUndefType) Equals(o interface{}, g eval.Guard) bool {
	if ot, ok := o.(*NotUndefType); ok {
		return t.typ.Equals(ot.typ, g)
	}
	return false
}

func (t *NotUndefType) Generic() eval.Type {
	return NewNotUndefType(eval.GenericType(t.typ))
}

func (t *NotUndefType) Get(key string) (value eval.Value, ok bool) {
	switch key {
	case `type`:
		return t.typ, true
	}
	return nil, false
}

func (t *NotUndefType) IsAssignable(o eval.Type, g eval.Guard) bool {
	return !GuardedIsAssignable(o, undefTypeDefault, g) && GuardedIsAssignable(t.typ, o, g)
}

func (t *NotUndefType) IsInstance(o eval.Value, g eval.Guard) bool {
	return o != undef && GuardedIsInstance(t.typ, o, g)
}

func (t *NotUndefType) MetaType() eval.ObjectType {
	return NotUndefMetaType
}

func (t *NotUndefType) Name() string {
	return `NotUndef`
}

func (t *NotUndefType) Parameters() []eval.Value {
	if t.typ == DefaultAnyType() {
		return eval.EmptyValues
	}
	if str, ok := t.typ.(*vcStringType); ok && str.value != `` {
		return []eval.Value{stringValue(str.value)}
	}
	return []eval.Value{t.typ}
}

func (t *NotUndefType) Resolve(c eval.Context) eval.Type {
	t.typ = resolve(c, t.typ)
	return t
}

func (t *NotUndefType) ReflectType(c eval.Context) (reflect.Type, bool) {
	return ReflectType(c, t.typ)
}

func (t *NotUndefType) CanSerializeAsString() bool {
	return canSerializeAsString(t.typ)
}

func (t *NotUndefType) SerializationString() string {
	return t.String()
}

func (t *NotUndefType) String() string {
	return eval.ToString2(t, None)
}

func (t *NotUndefType) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *NotUndefType) PType() eval.Type {
	return &TypeType{t}
}

var notUndefTypeDefault = &NotUndefType{typ: anyTypeDefault}
