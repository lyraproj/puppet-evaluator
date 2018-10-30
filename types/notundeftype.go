package types

import (
	"io"

	"github.com/puppetlabs/go-evaluator/errors"
	"github.com/puppetlabs/go-evaluator/eval"
	"reflect"
)

type NotUndefType struct {
	typ eval.Type
}

var NotUndef_Type eval.ObjectType

func init() {
	NotUndef_Type = newObjectType(`Pcore::NotUndefType`,
		`Pcore::AnyType {
			attributes => {
				type => {
					type => Optional[Type],
					value => Any
				},
			}
		}`, func(ctx eval.Context, args []eval.Value) eval.Value {
			return NewNotUndefType2(args...)
		})
}

func DefaultNotUndefType() *NotUndefType {
	return notUndefType_DEFAULT
}

func NewNotUndefType(containedType eval.Type) *NotUndefType {
	if containedType == nil || containedType == anyType_DEFAULT {
		return DefaultNotUndefType()
	}
	return &NotUndefType{containedType}
}

func NewNotUndefType2(args ...eval.Value) *NotUndefType {
	switch len(args) {
	case 0:
		return DefaultNotUndefType()
	case 1:
		if containedType, ok := args[0].(eval.Type); ok {
			return NewNotUndefType(containedType)
		}
		if containedType, ok := args[0].(*StringValue); ok {
			return NewNotUndefType3(containedType.String())
		}
		panic(NewIllegalArgumentType2(`NotUndef[]`, 0, `Variant[Type,String]`, args[0]))
	default:
		panic(errors.NewIllegalArgumentCount(`NotUndef[]`, `0 - 1`, len(args)))
	}
}

func NewNotUndefType3(str string) *NotUndefType {
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
	return notUndefType_DEFAULT
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
	return !GuardedIsAssignable(o, undefType_DEFAULT, g) && GuardedIsAssignable(t.typ, o, g)
}

func (t *NotUndefType) IsInstance(o eval.Value, g eval.Guard) bool {
	return o != _UNDEF && GuardedIsInstance(t.typ, o, g)
}

func (t *NotUndefType) MetaType() eval.ObjectType {
	return NotUndef_Type
}

func (t *NotUndefType) Name() string {
	return `NotUndef`
}

func (t *NotUndefType) Parameters() []eval.Value {
	if t.typ == DefaultAnyType() {
		return eval.EMPTY_VALUES
	}
	if str, ok := t.typ.(*StringType); ok && str.value != `` {
		return []eval.Value{WrapString(str.value)}
	}
	return []eval.Value{t.typ}
}

func (t *NotUndefType) Resolve(c eval.Context) eval.Type {
	t.typ = resolve(c, t.typ)
	return t
}

func (t *NotUndefType) ReflectType() (reflect.Type, bool) {
	return ReflectType(t.typ)
}

func (t *NotUndefType) String() string {
	return eval.ToString2(t, NONE)
}

func (t *NotUndefType) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *NotUndefType) Type() eval.Type {
	return &TypeType{t}
}

var notUndefType_DEFAULT = &NotUndefType{typ: anyType_DEFAULT}
