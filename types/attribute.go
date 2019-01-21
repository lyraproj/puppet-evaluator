package types

import (
	"fmt"
	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/hash"
)

const KEY_GONAME = `go_name`

var TYPE_ATTRIBUTE_KIND = NewEnumType([]string{string(CONSTANT), string(DERIVED), string(GIVEN_OR_DERIVED), string(REFERENCE)}, false)

var TYPE_ATTRIBUTE = NewStructType([]*StructElement{
	NewStructElement2(KEY_TYPE, NewVariantType(DefaultTypeType(), TYPE_TYPE_NAME)),
	NewStructElement(NewOptionalType3(KEY_FINAL), DefaultBooleanType()),
	NewStructElement(NewOptionalType3(KEY_OVERRIDE), DefaultBooleanType()),
	NewStructElement(NewOptionalType3(KEY_KIND), TYPE_ATTRIBUTE_KIND),
	NewStructElement(NewOptionalType3(KEY_VALUE), DefaultAnyType()),
	NewStructElement(NewOptionalType3(KEY_ANNOTATIONS), TYPE_ANNOTATIONS),
	NewStructElement(NewOptionalType3(KEY_GONAME), DefaultStringType()),
})

var TYPE_ATTRIBUTE_CALLABLE = NewCallableType2(NewIntegerType(0, 0))

type attribute struct {
	annotatedMember
	kind   eval.AttributeKind
	value  eval.Value
	goName string
}

func newAttribute(c eval.Context, name string, container *objectType, initHash *HashValue) eval.Attribute {
	a := &attribute{}
	a.initialize(c, name, container, initHash)
	return a
}

func (a *attribute) initialize(c eval.Context, name string, container *objectType, initHash *HashValue) {
	eval.AssertInstance(func() string { return fmt.Sprintf(`initializer for attribute %s[%s]`, container.Label(), name) }, TYPE_ATTRIBUTE, initHash)
	a.annotatedMember.initialize(c, `attribute`, name, container, initHash)
	a.kind = eval.AttributeKind(stringArg(initHash, KEY_KIND, ``))
	if a.kind == CONSTANT { // final is implied
		if initHash.IncludesKey2(KEY_FINAL) && !a.final {
			panic(eval.Error(eval.EVAL_CONSTANT_WITH_FINAL, issue.H{`label`: a.Label()}))
		}
		a.final = true
	}
	v := initHash.Get5(KEY_VALUE, nil)
	if v != nil {
		if a.kind == DERIVED || a.kind == GIVEN_OR_DERIVED {
			panic(eval.Error(eval.EVAL_ILLEGAL_KIND_VALUE_COMBINATION, issue.H{`label`: a.Label(), `kind`: a.kind}))
		}
		if _, ok := v.(*DefaultValue); ok || eval.IsInstance(a.typ, v) {
			a.value = v
		} else {
			panic(eval.Error(eval.EVAL_TYPE_MISMATCH, issue.H{`detail`: eval.DescribeMismatch(a.Label(), a.typ, eval.DetailedValueType(v))}))
		}
	} else {
		if a.kind == CONSTANT {
			panic(eval.Error(eval.EVAL_CONSTANT_REQUIRES_VALUE, issue.H{`label`: a.Label()}))
		}
		if a.kind == GIVEN_OR_DERIVED {
			// Type is always optional
			if !eval.IsInstance(a.typ, _UNDEF) {
				a.typ = NewOptionalType(a.typ)
			}
		}
		a.value = nil // Not to be confused with undef
	}
	if gn, ok := initHash.Get4(KEY_GONAME); ok {
		a.goName = gn.String()
	}
}

func (a *attribute) Call(c eval.Context, receiver eval.Value, block eval.Lambda, args []eval.Value) eval.Value {
	if block == nil && len(args) == 0 {
		return a.Get(receiver)
	}
	types := make([]eval.Value, len(args))
	for i, a := range args {
		types[i] = a.PType()
	}
	panic(eval.Error(eval.EVAL_TYPE_MISMATCH, issue.H{`detail`: eval.DescribeSignatures(
		[]eval.Signature{a.CallableType().(*CallableType)}, NewTupleType2(types...), block)}))
}

func (a *attribute) Default(value eval.Value) bool {
	return eval.Equals(a.value, value)
}

func (a *attribute) GoName() string {
	return a.goName
}

func (a *attribute) Kind() eval.AttributeKind {
	return a.kind
}

func (a *attribute) HasValue() bool {
	return a.value != nil
}

func (a *attribute) initHash() *hash.StringHash {
	hash := a.annotatedMember.initHash()
	if a.kind != DEFAULT_KIND {
		hash.Put(KEY_KIND, stringValue(string(a.kind)))
	}
	if a.value != nil {
		hash.Put(KEY_VALUE, a.value)
	}
	return hash
}

func (a *attribute) InitHash() eval.OrderedMap {
	return WrapStringPValue(a.initHash())
}

func (a *attribute) Value() eval.Value {
	if a.value == nil {
		panic(eval.Error(eval.EVAL_ATTRIBUTE_HAS_NO_VALUE, issue.H{`label`: a.Label()}))
	}
	return a.value
}

func (a *attribute) FeatureType() string {
	return `attribute`
}

func (a *attribute) Get(instance eval.Value) eval.Value {
	if a.kind == CONSTANT {
		return a.value
	}
	if v, ok := a.container.GetValue(a.name, instance); ok {
		return v
	}
	panic(eval.Error(eval.EVAL_NO_ATTRIBUTE_READER, issue.H{`label`: a.Label()}))
}

func (a *attribute) Label() string {
	return fmt.Sprintf(`attribute %s[%s]`, a.container.Label(), a.Name())
}

func (a *attribute) Equals(other interface{}, g eval.Guard) bool {
	if oa, ok := other.(*attribute); ok {
		return a.kind == oa.kind && a.override == oa.override && a.name == oa.name && a.final == oa.final && a.typ.Equals(oa.typ, g)
	}
	return false
}

func (a *attribute) CallableType() eval.Type {
	return TYPE_ATTRIBUTE_CALLABLE
}

func newTypeParameter(c eval.Context, name string, container *objectType, initHash *HashValue) eval.Attribute {
	t := &typeParameter{}
	t.initialize(c, name, container, initHash)
	return t
}
