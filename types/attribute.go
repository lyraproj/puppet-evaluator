package types

import (
	"fmt"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/hash"
)

const KeyGoName = `go_name`

var typeAttributeKind = NewEnumType([]string{string(constant), string(derived), string(givenOrDerived), string(reference)}, false)

var typeAttribute = NewStructType([]*StructElement{
	newStructElement2(keyType, NewVariantType(DefaultTypeType(), TypeTypeName)),
	NewStructElement(newOptionalType3(keyFinal), DefaultBooleanType()),
	NewStructElement(newOptionalType3(keyOverride), DefaultBooleanType()),
	NewStructElement(newOptionalType3(keyKind), typeAttributeKind),
	NewStructElement(newOptionalType3(keyValue), DefaultAnyType()),
	NewStructElement(newOptionalType3(keyAnnotations), typeAnnotations),
	NewStructElement(newOptionalType3(KeyGoName), DefaultStringType()),
})

var typeAttributeCallable = newCallableType2(NewIntegerType(0, 0))

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
	eval.AssertInstance(func() string { return fmt.Sprintf(`initializer for attribute %s[%s]`, container.Label(), name) }, typeAttribute, initHash)
	a.annotatedMember.initialize(c, `attribute`, name, container, initHash)
	a.kind = eval.AttributeKind(stringArg(initHash, keyKind, ``))
	if a.kind == constant { // final is implied
		if initHash.IncludesKey2(keyFinal) && !a.final {
			panic(eval.Error(eval.ConstantWithFinal, issue.H{`label`: a.Label()}))
		}
		a.final = true
	}
	v := initHash.Get5(keyValue, nil)
	if v != nil {
		if a.kind == derived || a.kind == givenOrDerived {
			panic(eval.Error(eval.IllegalKindValueCombination, issue.H{`label`: a.Label(), `kind`: a.kind}))
		}
		if _, ok := v.(*DefaultValue); ok || eval.IsInstance(a.typ, v) {
			a.value = v
		} else {
			panic(eval.Error(eval.TypeMismatch, issue.H{`detail`: eval.DescribeMismatch(a.Label(), a.typ, eval.DetailedValueType(v))}))
		}
	} else {
		if a.kind == constant {
			panic(eval.Error(eval.ConstantRequiresValue, issue.H{`label`: a.Label()}))
		}
		if a.kind == givenOrDerived {
			// Type is always optional
			if !eval.IsInstance(a.typ, undef) {
				a.typ = NewOptionalType(a.typ)
			}
		}
		a.value = nil // Not to be confused with undef
	}
	if gn, ok := initHash.Get4(KeyGoName); ok {
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
	panic(eval.Error(eval.TypeMismatch, issue.H{`detail`: eval.DescribeSignatures(
		[]eval.Signature{a.CallableType().(*CallableType)}, newTupleType2(types...), block)}))
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
	h := a.annotatedMember.initHash()
	if a.kind != defaultKind {
		h.Put(keyKind, stringValue(string(a.kind)))
	}
	if a.value != nil {
		h.Put(keyValue, a.value)
	}
	return h
}

func (a *attribute) InitHash() eval.OrderedMap {
	return WrapStringPValue(a.initHash())
}

func (a *attribute) Value() eval.Value {
	if a.value == nil {
		panic(eval.Error(eval.AttributeHasNoValue, issue.H{`label`: a.Label()}))
	}
	return a.value
}

func (a *attribute) FeatureType() string {
	return `attribute`
}

func (a *attribute) Get(instance eval.Value) eval.Value {
	if a.kind == constant {
		return a.value
	}
	if v, ok := a.container.GetValue(a.name, instance); ok {
		return v
	}
	panic(eval.Error(eval.NoAttributeReader, issue.H{`label`: a.Label()}))
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
	return typeAttributeCallable
}

func newTypeParameter(c eval.Context, name string, container *objectType, initHash *HashValue) eval.Attribute {
	t := &typeParameter{}
	t.initialize(c, name, container, initHash)
	return t
}
