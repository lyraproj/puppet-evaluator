package types

import (
	"io"

	"github.com/lyraproj/puppet-evaluator/errors"
	"github.com/lyraproj/puppet-evaluator/eval"
)

type TypeType struct {
	typ eval.Type
}

var typeType_DEFAULT = &TypeType{typ: anyType_DEFAULT}

var Type_Type eval.ObjectType

func init() {
	Type_Type = newObjectType(`Pcore::TypeType`,
		`Pcore::AnyType {
	attributes => {
		type => {
			type => Optional[Type],
			value => Any
		},
	}
}`, func(ctx eval.Context, args []eval.Value) eval.Value {
			return NewTypeType2(args...)
		})

	newGoConstructor(`Type`,
		func(d eval.Dispatch) {
			d.Param(`String`)
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				return c.ParseType(args[0])
			})
		},
		func(d eval.Dispatch) {
			d.Param2(TYPE_OBJECT_INIT_HASH)
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				return NewObjectType(``, nil, args[0]).Resolve(c)
			})
		})
}

func DefaultTypeType() *TypeType {
	return typeType_DEFAULT
}

func NewTypeType(containedType eval.Type) *TypeType {
	if containedType == nil || containedType == anyType_DEFAULT {
		return DefaultTypeType()
	}
	return &TypeType{containedType}
}

func NewTypeType2(args ...eval.Value) *TypeType {
	switch len(args) {
	case 0:
		return DefaultTypeType()
	case 1:
		if containedType, ok := args[0].(eval.Type); ok {
			return NewTypeType(containedType)
		}
		panic(NewIllegalArgumentType2(`Type[]`, 0, `Type`, args[0]))
	default:
		panic(errors.NewIllegalArgumentCount(`Type[]`, `0 or 1`, len(args)))
	}
}

func (t *TypeType) ContainedType() eval.Type {
	return t.typ
}

func (t *TypeType) Accept(v eval.Visitor, g eval.Guard) {
	v(t)
	t.typ.Accept(v, g)
}

func (t *TypeType) Default() eval.Type {
	return typeType_DEFAULT
}

func (t *TypeType) Equals(o interface{}, g eval.Guard) bool {
	if ot, ok := o.(*TypeType); ok {
		return t.typ.Equals(ot.typ, g)
	}
	return false
}

func (t *TypeType) Generic() eval.Type {
	return NewTypeType(eval.GenericType(t.typ))
}

func (t *TypeType) Get(key string) (value eval.Value, ok bool) {
	switch key {
	case `type`:
		return t.typ, true
	}
	return nil, false
}

func (t *TypeType) IsAssignable(o eval.Type, g eval.Guard) bool {
	if ot, ok := o.(*TypeType); ok {
		return GuardedIsAssignable(t.typ, ot.typ, g)
	}
	return false
}

func (t *TypeType) IsInstance(o eval.Value, g eval.Guard) bool {
	if ot, ok := o.(eval.Type); ok {
		return GuardedIsAssignable(t.typ, ot, g)
	}
	return false
}

func (t *TypeType) MetaType() eval.ObjectType {
	return Type_Type
}

func (t *TypeType) Name() string {
	return `Type`
}

func (t *TypeType) Parameters() []eval.Value {
	if t.typ == DefaultAnyType() {
		return eval.EMPTY_VALUES
	}
	return []eval.Value{t.typ}
}

func (t *TypeType) Resolve(c eval.Context) eval.Type {
	t.typ = resolve(c, t.typ)
	return t
}

func (t *TypeType) CanSerializeAsString() bool {
	ts, ok := t.typ.(eval.SerializeAsString)
	return ok && ts.CanSerializeAsString()
}

func (t *TypeType) SerializationString() string {
	return t.String()
}

func (t *TypeType) String() string {
	return eval.ToString2(t, NONE)
}

func (t *TypeType) PType() eval.Type {
	return &TypeType{t}
}

func (t *TypeType) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	TypeToString(t, b, s, g)
}
