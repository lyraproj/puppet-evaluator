package types

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"fmt"
	"github.com/puppetlabs/go-issues/issue"
)

var TYPE_ATTRIBUTE_KIND = NewEnumType([]string{string(CONSTANT), string(DERIVED), string(GIVEN_OR_DERIVED), string(REFERENCE)}, false)

var TYPE_ATTRIBUTE = NewStructType([]*StructElement{
	NewStructElement2(KEY_TYPE, DefaultTypeType()),
	NewStructElement(NewOptionalType3(KEY_FINAL), DefaultBooleanType()),
	NewStructElement(NewOptionalType3(KEY_OVERRIDE), DefaultBooleanType()),
	NewStructElement(NewOptionalType3(KEY_KIND), TYPE_ATTRIBUTE_KIND),
	NewStructElement(NewOptionalType3(KEY_VALUE), DefaultAnyType()),
	NewStructElement(NewOptionalType3(KEY_ANNOTATIONS), TYPE_ANNOTATIONS),
})

var TYPE_ATTRIBUTE_CALLABLE = NewCallableType2(NewIntegerType(0, 0))

type attribute struct {
	annotatedMember
	kind  eval.AttributeKind
	value eval.PValue
}

func newAttribute(c eval.Context, name string, container *objectType, initHash *HashValue) eval.Attribute {
	a := &attribute{}
	a.initialize(c, name, container, initHash)
	return a
}

func (a *attribute) initialize(c eval.Context, name string, container *objectType, initHash *HashValue) {
	a.annotatedMember.initialize(name, container, initHash)
	eval.AssertInstance(c, func() string { return fmt.Sprintf(`initializer for %s`, a.Label()) }, TYPE_ATTRIBUTE, initHash)
	a.kind = eval.AttributeKind(stringArg(initHash, KEY_KIND, ``))
	if a.kind == CONSTANT { // final is implied
		if initHash.IncludesKey2(KEY_FINAL) && !a.final {
			panic(eval.Error(c, eval.EVAL_CONSTANT_WITH_FINAL, issue.H{`label`: a.Label()}))
		}
		a.final = true
	}
	v := initHash.Get5(KEY_VALUE, nil)
	if v != nil {
		if a.kind == DERIVED || a.kind == GIVEN_OR_DERIVED {
			panic(eval.Error(c, eval.EVAL_ILLEGAL_KIND_VALUE_COMBINATION, issue.H{`label`: a.Label(), `kind`: a.kind}))
		}
		if _, ok := v.(*DefaultValue); ok || eval.IsInstance(c, a.typ, v) {
			a.value = v
		} else {
			panic(eval.Error(c, eval.EVAL_TYPE_MISMATCH, issue.H{`detail`: eval.DescribeMismatch(a.Label(), a.typ, eval.DetailedValueType(v))}))
		}
	} else {
		if a.kind == CONSTANT {
			panic(eval.Error(c, eval.EVAL_CONSTANT_REQUIRES_VALUE, issue.H{`label`: a.Label()}))
		}
		a.value = nil // Not to be confused with undef
	}
}

func (a *attribute) Call(c eval.Context, receiver eval.PValue, block eval.Lambda, args []eval.PValue) eval.PValue {
	if block == nil && len(args) == 0 {
		return a.Get(c, receiver)
	}
	panic(eval.Error(c, eval.EVAL_TYPE_MISMATCH, issue.H{`detail`: eval.DescribeSignatures(
		[]eval.Signature{a.CallableType().(*CallableType)}, NewTupleType2(args...), block)}))
}

func (a *attribute) Default(value eval.PValue) bool {
	return eval.Equals(a.value, value)
}

func (a *attribute) Kind() eval.AttributeKind {
	return a.kind
}

func (a *attribute) HasValue() bool {
	return a.value != nil
}

func (a *attribute) initHash() map[string]eval.PValue {
	hash := a.annotatedMember.initHash()
	if a.kind != DEFAULT_KIND {
		hash[KEY_KIND] = WrapString(string(a.kind))
	}
	if a.value != nil {
		hash[KEY_VALUE] = a.value
	}
	return hash
}

func (a *attribute) InitHash() eval.KeyedValue {
	return WrapHash3(a.initHash())
}

func (a *attribute) Value(c eval.Context) eval.PValue {
	if a.value == nil {
		panic(eval.Error(c, eval.EVAL_ATTRIBUTE_HAS_NO_VALUE, issue.H{`label`: a.Label()}))
	}
	return a.value
}

func (a *attribute) FeatureType() string {
	return `attribute`
}

func (a *attribute) Get(c eval.Context, instance eval.PValue) eval.PValue {
	if a.kind == CONSTANT {
		return a.value
	}
	if v, ok := a.container.GetValue(c, a.name, instance); ok {
		return v
	}
	panic(eval.Error(c, eval.EVAL_NO_ATTRIBUTE_READER, issue.H{`label`: a.Label()}))
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

func (a *attribute) CallableType() eval.PType {
	return TYPE_ATTRIBUTE_CALLABLE
}

func newTypeParameter(c eval.Context, name string, container *objectType, initHash *HashValue) eval.Attribute {
	t := &typeParameter{}
	t.initialize(c, name, container, initHash)
	return t
}
