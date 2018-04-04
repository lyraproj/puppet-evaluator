package types

import (
	"io"

	"github.com/puppetlabs/go-evaluator/errors"
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-parser/issue"
)

type TypeReferenceType struct {
	typeString string
}

var TypeReference_Type eval.ObjectType

func init() {
	TypeReference_Type = newObjectType(`Pcore::TypeReference`,
		`Pcore::AnyType {
	attributes => {
		type_string => String[1]
	}
}`, func(ctx eval.EvalContext, args []eval.PValue) eval.PValue {
			return NewTypeReferenceType2(args...)
		})
}

func DefaultTypeReferenceType() *TypeReferenceType {
	return typeReferenceType_DEFAULT
}

func NewTypeReferenceType(typeString string) *TypeReferenceType {
	return &TypeReferenceType{typeString}
}

func NewTypeReferenceType2(args ...eval.PValue) *TypeReferenceType {
	switch len(args) {
	case 0:
		return DefaultTypeReferenceType()
	case 1:
		if str, ok := args[0].(*StringValue); ok {
			return &TypeReferenceType{str.String()}
		}
		panic(NewIllegalArgumentType2(`TypeReference[]`, 0, `String`, args[0]))
	default:
		panic(errors.NewIllegalArgumentCount(`TypeReference[]`, `0 - 1`, len(args)))
	}
}

func (t *TypeReferenceType) Accept(v eval.Visitor, g eval.Guard) {
	v(t)
}

func (t *TypeReferenceType) Default() eval.PType {
	return typeReferenceType_DEFAULT
}

func (t *TypeReferenceType) Equals(o interface{}, g eval.Guard) bool {
	if ot, ok := o.(*TypeReferenceType); ok {
		return t.typeString == ot.typeString
	}
	return false
}

func (t *TypeReferenceType) Get(key string) (eval.PValue, bool) {
	switch key {
	case `type_string`:
		return WrapString(t.typeString), true
	default:
		return nil, false
	}
}

func (t *TypeReferenceType) IsAssignable(o eval.PType, g eval.Guard) bool {
	tr, ok := o.(*TypeReferenceType)
	return ok && t.typeString == tr.typeString
}

func (t *TypeReferenceType) IsInstance(o eval.PValue, g eval.Guard) bool {
	return false
}

func (t *TypeReferenceType) MetaType() eval.ObjectType {
	return TypeReference_Type
}

func (t *TypeReferenceType) Name() string {
	return `TypeReference`
}

func (t *TypeReferenceType) String() string {
	return eval.ToString2(t, NONE)
}

func (t *TypeReferenceType) Parameters() []eval.PValue {
	if *t == *typeReferenceType_DEFAULT {
		return eval.EMPTY_VALUES
	}
	return []eval.PValue{WrapString(t.typeString)}
}

func (t *TypeReferenceType) Resolve(c eval.EvalContext) eval.PType {
	r := c.ParseType2(t.typeString)
	if rt, ok := r.(eval.ResolvableType); ok {
		if tr, ok := rt.(*TypeReferenceType); ok && t.typeString == tr.typeString {
			panic(eval.Error(c, eval.EVAL_UNRESOLVED_TYPE, issue.H{`typeString`: t.typeString}))
		}
		r = rt.Resolve(c)
	}
	return r
}

func (t *TypeReferenceType) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *TypeReferenceType) Type() eval.PType {
	return &TypeType{t}
}

func (t *TypeReferenceType) TypeString() string {
	return t.typeString
}

var typeReferenceType_DEFAULT = &TypeReferenceType{`UnresolvedReference`}
