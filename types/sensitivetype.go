package types

import (
	"io"

	"github.com/lyraproj/puppet-evaluator/utils"

	"github.com/lyraproj/puppet-evaluator/errors"
	"github.com/lyraproj/puppet-evaluator/eval"
)

var sensitiveTypeDefault = &SensitiveType{typ: anyTypeDefault}

type (
	SensitiveType struct {
		typ eval.Type
	}

	SensitiveValue struct {
		value eval.Value
	}
)

var SensitiveMetaType eval.ObjectType

func init() {
	SensitiveMetaType = newObjectType(`Pcore::SensitiveType`,
		`Pcore::AnyType {
	attributes => {
		type => {
			type => Optional[Type],
			value => Any
		},
	}
}`, func(ctx eval.Context, args []eval.Value) eval.Value {
			return newSensitiveType2(args...)
		})

	newGoConstructor(`Sensitive`,
		func(d eval.Dispatch) {
			d.Param(`Any`)
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				return WrapSensitive(args[0])
			})
		})
}

func DefaultSensitiveType() *SensitiveType {
	return sensitiveTypeDefault
}

func NewSensitiveType(containedType eval.Type) *SensitiveType {
	if containedType == nil || containedType == anyTypeDefault {
		return DefaultSensitiveType()
	}
	return &SensitiveType{containedType}
}

func newSensitiveType2(args ...eval.Value) *SensitiveType {
	switch len(args) {
	case 0:
		return DefaultSensitiveType()
	case 1:
		if containedType, ok := args[0].(eval.Type); ok {
			return NewSensitiveType(containedType)
		}
		panic(NewIllegalArgumentType(`Sensitive[]`, 0, `Type`, args[0]))
	default:
		panic(errors.NewIllegalArgumentCount(`Sensitive[]`, `0 or 1`, len(args)))
	}
}

func (t *SensitiveType) ContainedType() eval.Type {
	return t.typ
}

func (t *SensitiveType) Accept(v eval.Visitor, g eval.Guard) {
	v(t)
	t.typ.Accept(v, g)
}

func (t *SensitiveType) Default() eval.Type {
	return DefaultSensitiveType()
}

func (t *SensitiveType) Equals(o interface{}, g eval.Guard) bool {
	if ot, ok := o.(*SensitiveType); ok {
		return t.typ.Equals(ot.typ, g)
	}
	return false
}

func (t *SensitiveType) Generic() eval.Type {
	return NewSensitiveType(eval.GenericType(t.typ))
}

func (t *SensitiveType) IsAssignable(o eval.Type, g eval.Guard) bool {
	if ot, ok := o.(*SensitiveType); ok {
		return GuardedIsAssignable(t.typ, ot.typ, g)
	}
	return false
}

func (t *SensitiveType) IsInstance(o eval.Value, g eval.Guard) bool {
	if sv, ok := o.(*SensitiveValue); ok {
		return GuardedIsInstance(t.typ, sv.Unwrap(), g)
	}
	return false
}

func (t *SensitiveType) MetaType() eval.ObjectType {
	return SensitiveMetaType
}

func (t *SensitiveType) Name() string {
	return `Sensitive`
}

func (t *SensitiveType) Parameters() []eval.Value {
	if t.typ == DefaultAnyType() {
		return eval.EmptyValues
	}
	return []eval.Value{t.typ}
}

func (t *SensitiveType) Resolve(c eval.Context) eval.Type {
	t.typ = resolve(c, t.typ)
	return t
}

func (t *SensitiveType) CanSerializeAsString() bool {
	return canSerializeAsString(t.typ)
}

func (t *SensitiveType) SerializationString() string {
	return t.String()
}

func (t *SensitiveType) String() string {
	return eval.ToString2(t, None)
}

func (t *SensitiveType) PType() eval.Type {
	return &TypeType{t}
}

func (t *SensitiveType) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	TypeToString(t, b, s, g)
}

func WrapSensitive(val eval.Value) *SensitiveValue {
	return &SensitiveValue{val}
}

func (s *SensitiveValue) Equals(o interface{}, g eval.Guard) bool {
	return false
}

func (s *SensitiveValue) String() string {
	return eval.ToString2(s, None)
}

func (s *SensitiveValue) ToString(b io.Writer, f eval.FormatContext, g eval.RDetect) {
	utils.WriteString(b, `Sensitive [value redacted]`)
}

func (s *SensitiveValue) PType() eval.Type {
	return NewSensitiveType(s.Unwrap().PType())
}

func (s *SensitiveValue) Unwrap() eval.Value {
	return s.value
}
