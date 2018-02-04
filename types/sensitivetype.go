package types

import (
	"io"

	"github.com/puppetlabs/go-evaluator/errors"
	"github.com/puppetlabs/go-evaluator/eval"
)

var sensitiveType_DEFAULT = &SensitiveType{typ: anyType_DEFAULT}

type (
	SensitiveType struct {
		typ eval.PType
	}

	SensitiveValue struct {
		value eval.PValue
	}
)

var Sensitive_Type eval.ObjectType

func init() {
	Sensitive_Type = newObjectType(`Pcore::SensitiveType`, `Pcore::AnyType{}`, func(ctx eval.EvalContext, args []eval.PValue) eval.PValue {
		return DefaultSensitiveType()
	})
}

func DefaultSensitiveType() *SensitiveType {
	return sensitiveType_DEFAULT
}

func NewSensitiveType(containedType eval.PType) *SensitiveType {
	if containedType == nil || containedType == anyType_DEFAULT {
		return DefaultSensitiveType()
	}
	return &SensitiveType{containedType}
}

func NewSensitiveType2(args ...eval.PValue) *SensitiveType {
	switch len(args) {
	case 0:
		return DefaultSensitiveType()
	case 1:
		if containedType, ok := args[0].(eval.PType); ok {
			return NewSensitiveType(containedType)
		}
		panic(NewIllegalArgumentType2(`Sensitive[]`, 0, `Type`, args[0]))
	default:
		panic(errors.NewIllegalArgumentCount(`Sensitive[]`, `0 or 1`, len(args)))
	}
}

func (t *SensitiveType) ContainedType() eval.PType {
	return t.typ
}

func (t *SensitiveType) Accept(v eval.Visitor, g eval.Guard) {
	v(t)
	t.typ.Accept(v, g)
}

func (t *SensitiveType) Default() eval.PType {
	return DefaultSensitiveType()
}

func (t *SensitiveType) Equals(o interface{}, g eval.Guard) bool {
	if ot, ok := o.(*SensitiveType); ok {
		return t.typ.Equals(ot.typ, g)
	}
	return false
}

func (t *SensitiveType) Generic() eval.PType {
	return NewSensitiveType(eval.GenericType(t.typ))
}

func (t *SensitiveType) IsAssignable(o eval.PType, g eval.Guard) bool {
	if ot, ok := o.(*SensitiveType); ok {
		return GuardedIsAssignable(t.typ, ot.typ, g)
	}
	return false
}

func (t *SensitiveType) IsInstance(o eval.PValue, g eval.Guard) bool {
	if sv, ok := o.(*SensitiveValue); ok {
		return GuardedIsInstance(t.typ, sv.Unwrap(), g)
	}
	return false
}

func (t *SensitiveType) MetaType() eval.ObjectType {
	return Sensitive_Type
}

func (t *SensitiveType) Name() string {
	return `Sensitive`
}

func (t *SensitiveType) Parameters() []eval.PValue {
	if t.typ == DefaultAnyType() {
		return eval.EMPTY_VALUES
	}
	return []eval.PValue{t.typ}
}

func (t *SensitiveType) String() string {
	return eval.ToString2(t, NONE)
}

func (t *SensitiveType) Type() eval.PType {
	return &TypeType{t}
}

func (t *SensitiveType) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	TypeToString(t, b, s, g)
}

func WrapSensitive(val eval.PValue) *SensitiveValue {
	return &SensitiveValue{val}
}

func (s *SensitiveValue) Equals(o interface{}, g eval.Guard) bool {
	return false
}

func (s *SensitiveValue) String() string {
	return eval.ToString2(s, NONE)
}

func (s *SensitiveValue) ToString(b io.Writer, f eval.FormatContext, g eval.RDetect) {
	io.WriteString(b, `Sensitive [value redacted]`)
}

func (s *SensitiveValue) Type() eval.PType {
	return NewSensitiveType(s.Unwrap().Type())
}

func (s *SensitiveValue) Unwrap() eval.PValue {
	return s.value
}
