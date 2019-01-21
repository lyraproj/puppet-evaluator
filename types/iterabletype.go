package types

import (
	"io"

	"github.com/lyraproj/puppet-evaluator/errors"
	"github.com/lyraproj/puppet-evaluator/eval"
)

type IterableType struct {
	typ eval.Type
}

var Iterable_Type eval.ObjectType

func init() {
	Iterable_Type = newObjectType(`Pcore::IterableType`,
		`Pcore::AnyType {
			attributes => {
				type => {
					type => Optional[Type],
					value => Any
				},
			}
		}`, func(ctx eval.Context, args []eval.Value) eval.Value {
			return NewIterableType2(args...)
		})
}

func DefaultIterableType() *IterableType {
	return iterableType_DEFAULT
}

func NewIterableType(elementType eval.Type) *IterableType {
	if elementType == nil || elementType == anyType_DEFAULT {
		return DefaultIterableType()
	}
	return &IterableType{elementType}
}

func NewIterableType2(args ...eval.Value) *IterableType {
	switch len(args) {
	case 0:
		return DefaultIterableType()
	case 1:
		containedType, ok := args[0].(eval.Type)
		if !ok {
			panic(NewIllegalArgumentType2(`Iterable[]`, 0, `Type`, args[0]))
		}
		return NewIterableType(containedType)
	default:
		panic(errors.NewIllegalArgumentCount(`Iterable[]`, `0 - 1`, len(args)))
	}
}

func (t *IterableType) Accept(v eval.Visitor, g eval.Guard) {
	v(t)
	t.typ.Accept(v, g)
}

func (t *IterableType) Default() eval.Type {
	return iterableType_DEFAULT
}

func (t *IterableType) Equals(o interface{}, g eval.Guard) bool {
	if ot, ok := o.(*IterableType); ok {
		return t.typ.Equals(ot.typ, g)
	}
	return false
}

func (t *IterableType) Generic() eval.Type {
	return NewIterableType(eval.GenericType(t.typ))
}

func (t *IterableType) Get(key string) (value eval.Value, ok bool) {
	switch key {
	case `type`:
		return t.typ, true
	}
	return nil, false
}

func (t *IterableType) IsAssignable(o eval.Type, g eval.Guard) bool {
	var et eval.Type
	switch o.(type) {
	case *ArrayType:
		et = o.(*ArrayType).ElementType()
	case *BinaryType:
		et = NewIntegerType(0, 255)
	case *HashType:
		et = o.(*HashType).EntryType()
	case *stringType:
		et = OneCharStringType
	case *TupleType:
		return allAssignableTo(o.(*TupleType).types, t.typ, g)
	default:
		return false
	}
	return GuardedIsAssignable(t.typ, et, g)
}

func (t *IterableType) IsInstance(o eval.Value, g eval.Guard) bool {
	if iv, ok := o.(eval.IterableValue); ok {
		return GuardedIsAssignable(t.typ, iv.ElementType(), g)
	}
	return false
}

func (t *IterableType) MetaType() eval.ObjectType {
	return Iterable_Type
}

func (t *IterableType) Name() string {
	return `Iterable`
}

func (t *IterableType) Parameters() []eval.Value {
	if t.typ == DefaultAnyType() {
		return eval.EMPTY_VALUES
	}
	return []eval.Value{t.typ}
}

func (t *IterableType) Resolve(c eval.Context) eval.Type {
	t.typ = resolve(c, t.typ)
	return t
}

func (t *IterableType) CanSerializeAsString() bool {
	return canSerializeAsString(t.typ)
}

func (t *IterableType) SerializationString() string {
	return t.String()
}

func (t *IterableType) String() string {
	return eval.ToString2(t, NONE)
}

func (t *IterableType) ElementType() eval.Type {
	return t.typ
}

func (t *IterableType) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *IterableType) PType() eval.Type {
	return &TypeType{t}
}

var iterableType_DEFAULT = &IterableType{typ: DefaultAnyType()}
