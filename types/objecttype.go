package types

import (
	"io"
	"regexp"

	"fmt"
	"github.com/puppetlabs/go-evaluator/errors"
	. "github.com/puppetlabs/go-evaluator/evaluator"
	. "github.com/puppetlabs/go-evaluator/hash"
	. "github.com/puppetlabs/go-parser/issue"
	. "github.com/puppetlabs/go-parser/parser"
	"github.com/puppetlabs/go-parser/validator"
	"sync/atomic"
	"github.com/puppetlabs/go-evaluator/utils"
)

const (
	KEY_ANNOTATIONS           = `annotations`
	KEY_ATTRIBUTES            = `attributes`
	KEY_CHECKS                = `checks`
	KEY_CONSTANTS             = `constants`
	KEY_EQUALITY              = `equality`
	KEY_EQUALITY_INCLUDE_TYPE = `equality_include_type`
	KEY_FINAL                 = `final`
	KEY_FUNCTIONS             = `functions`
	KEY_KIND                  = `kind`
	KEY_NAME                  = `name`
	KEY_OVERRIDE              = `override`
	KEY_PARENT                = `parent`
	KEY_TYPE                  = `type`
	KEY_TYPE_PARAMETERS       = `type_parameters`
	KEY_VALUE                 = `value`

	DEFAULT_KIND     = AttributeKind(``)
	CONSTANT         = AttributeKind(`constant`)
	DERIVED          = AttributeKind(`derived`)
	GIVEN_OR_DERIVED = AttributeKind(`given_or_derived`)
	REFERENCE        = AttributeKind(`reference`)
)

var QREF_PATTERN = regexp.MustCompile(`\A[A-Z][\w]*(?:::[A-Z][\w]*)*\z`)

var annotationType_DEFAULT = &ObjectType{
	annotatable: annotatable{annotations:_EMPTY_MAP},
	hashKey:     HashKey("\x00tAnnotation"),
	name:        `Annotation`,
	parameters:  EMPTY_STRINGHASH,
	attributes:  EMPTY_STRINGHASH,
	functions:   EMPTY_STRINGHASH,
	equality:    nil}

var TYPE_ATTRIBUTE_KIND = NewEnumType([]string{string(CONSTANT), string(DERIVED), string(GIVEN_OR_DERIVED), string(REFERENCE)})
var TYPE_OBJECT_NAME = NewPatternType([]*RegexpType{NewRegexpTypeR(QREF_PATTERN)})

var TYPE_TYPE_PARAMETER = NewStructType([]*StructElement{
	NewStructElement2(KEY_TYPE, DefaultTypeType()),
	NewStructElement(NewOptionalType3(KEY_ANNOTATIONS), annotationType_DEFAULT),
})

var TYPE_ATTRIBUTE = NewStructType([]*StructElement{
	NewStructElement2(KEY_TYPE, DefaultTypeType()),
	NewStructElement(NewOptionalType3(KEY_FINAL), DefaultBooleanType()),
	NewStructElement(NewOptionalType3(KEY_OVERRIDE), DefaultBooleanType()),
	NewStructElement(NewOptionalType3(KEY_KIND), TYPE_ATTRIBUTE_KIND),
	NewStructElement(NewOptionalType3(KEY_VALUE), DefaultAnyType()),
	NewStructElement(NewOptionalType3(KEY_ANNOTATIONS), annotationType_DEFAULT),
})

var TYPE_MEMBER_NAME = NewPatternType2(NewRegexpTypeR(validator.PARAM_NAME))
var TYPE_ATTRIBUTE_CALLABLE = NewCallableType2(NewIntegerType(0, 0))
var TYPE_FUNCTION_TYPE = NewTypeType(DefaultCallableType())

var TYPE_FUNCTION = NewStructType([]*StructElement{
	NewStructElement2(KEY_TYPE, TYPE_FUNCTION_TYPE),
	NewStructElement(NewOptionalType3(KEY_FINAL), DefaultBooleanType()),
	NewStructElement(NewOptionalType3(KEY_OVERRIDE), DefaultBooleanType()),
	NewStructElement(NewOptionalType3(KEY_ANNOTATIONS), annotationType_DEFAULT),
})

var TYPE_PARAMETERS = NewHashType(TYPE_MEMBER_NAME, DefaultNotUndefType(), nil)
var TYPE_ATTRIBUTES = NewHashType(TYPE_MEMBER_NAME, DefaultNotUndefType(), nil)
var TYPE_CONSTANTS = NewHashType(TYPE_MEMBER_NAME, DefaultAnyType(), nil)
var TYPE_FUNCTIONS = NewHashType(TYPE_MEMBER_NAME, DefaultNotUndefType(), nil)
var TYPE_EQUALITY = NewVariantType2(TYPE_MEMBER_NAME, NewArrayType2(TYPE_MEMBER_NAME))
var TYPE_CHECKS = DefaultAnyType()

var TYPE_OBJECT_INIT_HASH = NewStructType([]*StructElement{
	NewStructElement(NewOptionalType3(KEY_NAME), TYPE_OBJECT_NAME),
	NewStructElement(NewOptionalType3(KEY_PARENT), DefaultTypeType()),
	NewStructElement(NewOptionalType3(KEY_TYPE_PARAMETERS), TYPE_PARAMETERS),
	NewStructElement(NewOptionalType3(KEY_ATTRIBUTES), TYPE_ATTRIBUTES),
	NewStructElement(NewOptionalType3(KEY_CONSTANTS), TYPE_CONSTANTS),
	NewStructElement(NewOptionalType3(KEY_FUNCTIONS), TYPE_FUNCTIONS),
	NewStructElement(NewOptionalType3(KEY_EQUALITY), TYPE_EQUALITY),
	NewStructElement(NewOptionalType3(KEY_EQUALITY_INCLUDE_TYPE), DefaultBooleanType()),
	NewStructElement(NewOptionalType3(KEY_CHECKS), TYPE_CHECKS),
	NewStructElement(NewOptionalType3(KEY_ANNOTATIONS), annotationType_DEFAULT),
})

type (
	Annotatable interface {
		Annotations() *HashValue
	}

	AnnotatedMember interface {
		Equality

		Name() string

		Label() string

		FeatureType() string

		Container() *ObjectType

		Type() PType

		Override() bool

		Final() bool

		InitHash() *HashValue

		accept(v Visitor, g Guard)

		callableType() *CallableType
	}

	AttributeKind string

	Attribute interface {
		AnnotatedMember
		Kind() AttributeKind

		// return true if a value has been defined for this attribute.
		HasValue() bool

		// return true if the given value equals the default value for this attribute
		Default(value PValue) bool

		// Returns the value of this attribute, or raises an error if no value has been defined.
		Value() PValue
	}

	ObjFunc interface {
		AnnotatedMember
	}

	PuppetObject interface {
		PValue

		InitHash() *HashValue

		Get(key string) (value PValue, ok bool)
	}

	annotatable struct {
		annotations *HashValue
	}

	annotatedMember struct {
		annotatable
		name        string
		container   *ObjectType
		typ         PType
		override    bool
		final       bool
	}

	attribute struct {
		annotatedMember
		kind  AttributeKind
		value PValue
	}

	typeParameter struct {
		attribute
	}

	function struct {
		annotatedMember
	}

	ObjectType struct {
		annotatable
		hashKey             HashKey
		name                string
		parent              PType
		parameters          *StringHash // map doesn't preserve order
		attributes          *StringHash
		functions           *StringHash
		equality            []string
		equalityIncludeType bool
		loader              Loader
		initHashExpression  interface{} // Expression or *HashValue
	}

	ObjectValue struct {
		typ PType
		values *HashValue
	}
)

func argError(e PType, a PValue) errors.InstantiationError {
	return errors.NewArgumentsError(``, DescribeMismatch(`assert`, e, a.Type()))
}

func typeArg(hash *HashValue, key string, d PType) PType {
	v := hash.Get2(key, nil)
	if v == nil {
		return d
	}
	if t, ok := v.(PType); ok {
		return t
	}
	panic(argError(DefaultTypeType(), v))
}

func hashArg(hash *HashValue, key string) *HashValue {
	v := hash.Get2(key, nil)
	if v == nil {
		return _EMPTY_MAP
	}
	if t, ok := v.(*HashValue); ok {
		return t
	}
	panic(argError(DefaultHashType(), v))
}

func boolArg(hash *HashValue, key string, d bool) bool {
	v := hash.Get2(key, nil)
	if v == nil {
		return d
	}
	if t, ok := v.(*BooleanValue); ok {
		return t.Bool()
	}
	panic(argError(DefaultBooleanType(), v))
}

func stringArg(hash *HashValue, key string, d string) string {
	v := hash.Get2(key, nil)
	if v == nil {
		return d
	}
	if t, ok := v.(*StringValue); ok {
		return t.String()
	}
	panic(argError(DefaultStringType(), v))
}

// Visit the keys of an annotations map. All keys are known to be types
func visitAnnotations(a *HashValue, v Visitor, g Guard) {
	if a != nil {
		for _, e := range a.EntriesSlice() {
			e.key.(PType).Accept(v, g)
		}
	}
}

func (a *annotatable) initialize(initHash *HashValue) {
	a.annotations = hashArg(initHash, KEY_ANNOTATIONS)
}

func (a *annotatable) initHash() map[string]PValue {
	h := make(map[string]PValue, 5)
	if a.annotations != nil {
		h[KEY_ANNOTATIONS] = a.annotations
	}
	return h
}

func (a *annotatedMember) initialize(name string, container *ObjectType, initHash *HashValue) {
	a.annotatable.initialize(initHash)
	a.name = name
	a.container = container
	a.typ = typeArg(initHash, KEY_TYPE, DefaultTypeType())
	a.override = boolArg(initHash, KEY_OVERRIDE, false)
	a.final = boolArg(initHash, KEY_FINAL, false)
}

func (a *annotatedMember) accept(v Visitor, g Guard) {
	a.typ.Accept(v, g)
	visitAnnotations(a.annotations, v, g)
}

func (a *annotatedMember) Annotations() *HashValue {
	return a.annotations
}

// Checks if the this _member_ overrides an inherited member, and if so, that this member is declared with
// override = true and that the inherited member accepts to be overridden by this member.
func assertOverride(a AnnotatedMember, parentMembers *StringHash) {
	parentMember, _ := parentMembers.Get(a.Name(), nil).(AnnotatedMember)
	if parentMember == nil {
		if a.Override() {
			panic(Error(EVAL_OVERRIDDEN_NOT_FOUND, H{`label`: a.Label(), `feature_type`: a.FeatureType()}))
		}
	} else {
		assertCanBeOverridden(parentMember, a)
	}
}

func assertCanBeOverridden(a AnnotatedMember, member AnnotatedMember) {
	if a.FeatureType() != member.FeatureType() {
		panic(Error(EVAL_OVERRIDE_MEMBER_MISMATCH, H{`member`: member.Label(), `label`: a.Label()}))
	}
	if a.Final() {
		aa, ok := a.(Attribute)
		if !(ok && aa.Kind() == CONSTANT && member.(Attribute).Kind() == CONSTANT) {
			panic(Error(EVAL_OVERRIDE_OF_FINAL, H{`member`: member.Label(), `label`: a.Label()}))
		}
	}
	if !member.Override() {
		panic(Error(EVAL_OVERRIDE_IS_MISSING, H{`member`: member.Label(), `label`: a.Label()}))
	}
	if !IsAssignable(a.Type(), member.Type()) {
		panic(Error(EVAL_OVERRIDE_TYPE_MISMATCH, H{`member`: member.Label(), `label`: a.Label()}))
	}
}

func (a *annotatedMember) Name() string {
	return a.name
}

func (a *annotatedMember) Container() *ObjectType {
	return a.container
}

func (a *annotatedMember) Type() PType {
	return a.typ
}

func (a *annotatedMember) Override() bool {
	return a.override
}

func (a *annotatedMember) initHash() map[string]PValue {
	h := a.annotatable.initHash()
	h[KEY_TYPE] = a.typ
	if a.final {
		h[KEY_FINAL] = WrapBoolean(true)
	}
	if a.override {
		h[KEY_OVERRIDE] = WrapBoolean(true)
	}
	return h
}

func (a *annotatedMember) Final() bool {
	return a.final
}

func newAttribute(name string, container *ObjectType, initHash *HashValue) Attribute {
	a := &attribute{}
	a.initialize(name, container, initHash)
	return a
}

func (a *attribute) initialize(name string, container *ObjectType, initHash *HashValue) {
	a.annotatedMember.initialize(name, container, initHash)
	AssertInstance(func() string { return fmt.Sprintf(`initializer for %s`, a.Label()) }, TYPE_ATTRIBUTE, initHash)
	a.kind = AttributeKind(stringArg(initHash, KEY_KIND, ``))
	if a.kind == CONSTANT { // final is implied
		if initHash.IncludesKey2(KEY_FINAL) && !a.final {
			panic(Error(EVAL_CONSTANT_WITH_FINAL, H{`label`: a.Label()}))
		}
		a.final = true
	}
	v := initHash.Get2(KEY_VALUE, nil)
	if v != nil {
		if a.kind == DERIVED || a.kind == GIVEN_OR_DERIVED {
			panic(Error(EVAL_ILLEGAL_KIND_VALUE_COMBINATION, H{`label`: a.Label(), `kind`: a.kind}))
		}
		if _, ok := v.(*DefaultValue); ok || IsInstance(a.typ, v) {
			a.value = v
		} else {
			panic(Error(EVAL_TYPE_MISMATCH, H{`detail`: DescribeMismatch(a.Label(), a.typ, DetailedValueType(v))}))
		}
	} else {
		if a.kind == CONSTANT {
			panic(Error(EVAL_CONSTANT_REQUIRES_VALUE, H{`label`: a.Label()}))
		}
		a.value = _UNDEF // Not to be confused with nil or :default
	}
}

func (a *attribute) Default(value PValue) bool {
	return Equals(a.value, value)
}

func (a *attribute) Kind() AttributeKind {
	return a.kind
}

func (a *attribute) HasValue() bool {
	return a.value != nil
}

func (a *attribute) initHash() map[string]PValue {
	hash := a.annotatedMember.initHash()
	if a.kind != DEFAULT_KIND {
		hash[KEY_KIND] = WrapString(string(a.kind))
	}
	if a.value != nil {
		hash[KEY_VALUE] = a.value
	}
	return hash
}

func (a *attribute) InitHash() *HashValue {
	return WrapHash3(a.initHash())
}

func (a *attribute) Value() PValue {
	if a.value == nil {
		panic(Error(EVAL_ATTRIBUTE_HAS_NO_VALUE, H{`label`: a.Label()}))
	}
	return a.value
}

func (a *attribute) FeatureType() string {
	return `attribute`
}

func (a *attribute) Label() string {
	return fmt.Sprintf(`attribute %s[%s]`, a.container.Label(), a.Name())
}

func (a *attribute) Equals(other interface{}, g Guard) bool {
	if oa, ok := other.(*attribute); ok {
		return a.kind == oa.kind && a.override == oa.override && a.name == oa.name && a.final == oa.final && a.typ.Equals(oa.typ, g)
	}
	return false
}

func (a *attribute) callableType() *CallableType {
	return TYPE_ATTRIBUTE_CALLABLE
}

func newTypeParameter(name string, container *ObjectType, initHash *HashValue) Attribute {
	t := &typeParameter{}
	t.initialize(name, container, initHash)
	return t
}

func (t *typeParameter) initHash() map[string]PValue {
	hash := t.attribute.initHash()
	hash[KEY_TYPE] = hash[KEY_TYPE].(*TypeType).Type()
	if v, ok := hash[KEY_VALUE]; ok && Equals(v, _UNDEF) {
		delete(hash, KEY_VALUE)
	}
	return hash
}

func (t *typeParameter) InitHash() *HashValue {
	return WrapHash3(t.initHash())
}

func (t *typeParameter) FeatureType() string {
	return `type_parameter`
}

func newFunction(name string, container *ObjectType, initHash *HashValue) ObjFunc {
	f := &function{}
	f.initialize(name, container, initHash)
	return f
}

func (f *function) initialize(name string, container *ObjectType, initHash *HashValue) {
	f.annotatedMember.initialize(name, container, initHash)
	AssertInstance(func() string { return fmt.Sprintf(`initializer for %s`, f.Label()) }, TYPE_FUNCTION, initHash)
}

func (f *function) Equals(other interface{}, g Guard) bool {
	if of, ok := other.(*function); ok {
		return f.override == of.override && f.name == of.name && f.final == of.final && f.typ.Equals(of.typ, g)
	}
	return false
}

func (f *function) FeatureType() string {
	return `function`
}

func (f *function) Label() string {
	return fmt.Sprintf(`function %s[%s]`, f.container.Label(), f.Name())
}

func (f *function) callableType() *CallableType {
	return f.typ.(*CallableType)
}

func (f *function) InitHash() *HashValue {
	return WrapHash3(f.initHash())
}

var objectType_DEFAULT = &ObjectType{
	annotatable: annotatable{annotations:_EMPTY_MAP},
	name:        `Object`,
	hashKey:     HashKey("\x00tObject"),
	parameters:  EMPTY_STRINGHASH,
	attributes:  EMPTY_STRINGHASH,
	functions:   EMPTY_STRINGHASH,
	equality:    nil}

func DefaultObjectType() *ObjectType {
	return objectType_DEFAULT
}

var objectId = int64(0)

func NewObjectType(name string, parent PType, initHashExpression Expression) *ObjectType {
	return &ObjectType{
		annotatable: annotatable{annotations: _EMPTY_MAP},
		name: name,
		hashKey:     HashKey(fmt.Sprintf("\x00tObject%d", atomic.AddInt64(&objectId, 1))),
		initHashExpression: initHashExpression,
		parent:      parent,
		parameters:  EMPTY_STRINGHASH,
		attributes:  EMPTY_STRINGHASH,
		functions:   EMPTY_STRINGHASH,
		equality: nil}
}

func NewObjectType2(initHash *HashValue, loader Loader) *ObjectType {
	result := &ObjectType{hashKey: HashKey(fmt.Sprintf("\x00tObject%d", atomic.AddInt64(&objectId, 1)))}
	result.initFromHash(initHash)
	result.loader = loader
	return result
}

func (t *ObjectType) Accept(v Visitor, g Guard) {
	if g == nil {
		g = make(Guard)
	}
	if g.Seen(t, nil) {
		return
	}
	v(t)
	visitAnnotations(t.annotations, v, g)
	if t.parent != nil {
		t.parent.Accept(v, g)
	}
	t.parameters.EachValue(func(p interface{}) { p.(AnnotatedMember).accept(v, g) })
	t.attributes.EachValue(func(a interface{}) { a.(AnnotatedMember).accept(v, g) })
	t.functions.EachValue(func(f interface{}) { f.(AnnotatedMember).accept(v, g) })
}

func (t *ObjectType) Default() PType {
	return objectType_DEFAULT
}

func (t *ObjectType) Equals(other interface{}, guard Guard) bool {
	ot, ok := other.(*ObjectType)
	if !ok {
		return false
	}

  if t.name == `` || ot.name == `` {
  	return t == ot
	}

	return t.name == ot.name && t.loader.NameAuthority() == ot.loader.NameAuthority()
}

func (t *ObjectType) GetValue(key string, o PValue) (value PValue, ok bool) {
	if pu, ok := o.(PuppetObject); ok {
		return pu.Get(key)
	}

	// TODO: Perhaps use other ways of extracting attributes with reflection
	// in case native types must be described by Object
	return nil, false
}

func (t *ObjectType) Name() string {
	return t.name
}

func (t *ObjectType) Label() string {
	if t.name == `` {
		return `Object`
	}
	return t.name
}

func (t *ObjectType) ToKey() HashKey {
	return t.hashKey
}

func (t *ObjectType) ToString(b io.Writer, s FormatContext, g RDetect) {
	TypeToString(t, b, s, g)
}

func (t *ObjectType) Type() PType {
	return &TypeType{t}
}

func (t *ObjectType) String() string {
	return ToString2(t, EXPANDED)
}

func (t *ObjectType) InitHash(includeName bool) *HashValue {
	return WrapHash3(t.initHash(includeName))
}

func (t *ObjectType) IsInstance(o PValue, g Guard) bool {
	return isAssignable(t, o.Type())
}

func (t *ObjectType) IsAssignable(o PType, g Guard) bool {
	if ot, ok := o.(*ObjectType); ok {
		if t.Equals(ot, g) {
			return true
		}
		if ot.parent != nil {
			return t.IsAssignable(ot.parent, g)
		}
	}
	return false
}

func (t *ObjectType) IsParameterized() bool {
	return !t.parameters.IsEmpty()
}

func (t *ObjectType) Resolve(resolver TypeResolver) PType {
  if t.initHashExpression != nil {
  	ihe := t.initHashExpression
  	t.initHashExpression = nil

  	var initHash *HashValue
  	if lh, ok := ihe.(*LiteralHash); ok {
  		initHash = resolver.Resolve(lh).(*HashValue)
  		if prt, ok := t.parent.(ResolvableType); ok {
  			t.parent = resolveTypeRefs(resolver, prt).(PType)
			}
		} else {
			initHash = resolveTypeRefs(resolver, ihe.(*HashValue)).(*HashValue)
		}
		t.initFromHash(initHash)
		t.loader = resolver.Loader()
	}
	return t
}

func resolveTypeRefs(resolver TypeResolver, v PValue) PValue {
  switch v.(type) {
	case *HashValue:
		hv := v.(*HashValue)
		he := make([]*HashEntry, hv.Len())
		i := 0
		hv.EachPair(func(key, value PValue) {
			he[i] = WrapHashEntry(
				resolveTypeRefs(resolver, key), resolveTypeRefs(resolver, value))
			i++
		})
		return WrapHash(he)
	case *ArrayValue:
		av := v.(*ArrayValue)
		ae := make([]PValue, av.Len())
		i := 0
		av.Each(func(value PValue) {
			ae[i] = resolveTypeRefs(resolver, value)
			i++
		})
		return WrapArray(ae)
	case ResolvableType:
		return v.(ResolvableType).Resolve(resolver)
	default:
		return v
	}
}

func (t *ObjectType) initFromHash(initHash *HashValue) {
	AssertInstance(`object initializer`, TYPE_OBJECT_INIT_HASH, initHash)
	t.parameters = EMPTY_STRINGHASH
	t.attributes = EMPTY_STRINGHASH
	t.functions = EMPTY_STRINGHASH
	t.name = stringArg(initHash, KEY_NAME, t.name)

	t.parent = typeArg(initHash, KEY_PARENT, nil)

	parentMembers := EMPTY_STRINGHASH
	parentTypeParams := EMPTY_STRINGHASH
	var parentObjectType *ObjectType

	if t.parent != nil {
		t.checkSelfRecursion(t)
		parentObjectType = t.resolvedParent()
		parentMembers = parentObjectType.members(true)
		parentTypeParams = parentObjectType.typeParameters(true)
	}

	typeParameters := hashArg(initHash, KEY_TYPE_PARAMETERS)
	if !typeParameters.IsEmpty() {
		parameters := NewStringHash(typeParameters.Len())
		for _, he := range typeParameters.EntriesSlice() {
			key := he.Key().String()
			var paramType PType
			var paramValue PValue
			if ph, ok := he.value.(*HashValue); ok {
				AssertInstance(
					func() string { return fmt.Sprintf(`type_parameter %s[%s]`, t.Label(), key) },
					TYPE_TYPE_PARAMETER, ph)
				paramType = typeArg(ph, KEY_TYPE, DefaultTypeType())
				paramValue = ph.Get2(KEY_VALUE, nil)
			} else {
				paramType = AssertInstance(
					func() string { return fmt.Sprintf(`type_parameter %s[%s]`, t.Label(), key) },
					DefaultTypeType(), he.value).(PType)
				paramValue = nil
			}
			if _, ok := paramType.(*OptionalType); !ok {
				paramType = NewOptionalType(paramType)
			}
			param := newTypeParameter(key, t, WrapHash4(H{
				KEY_TYPE:  paramType,
				KEY_VALUE: paramValue}))
			assertOverride(param, parentTypeParams)
			parameters.Put(key,param)
		}
		parameters.Freeze()
		t.parameters = parameters
	}

	constants := hashArg(initHash, KEY_CONSTANTS)
	attributes := hashArg(initHash, KEY_ATTRIBUTES)
	attrSpecs := NewStringHash(constants.Len() + attributes.Len())
	for _, ae := range attributes.EntriesSlice() {
		attrSpecs.Put(ae.Key().String(), ae.Value())
	}

	if !constants.IsEmpty() {
		for _, he := range constants.EntriesSlice() {
			key := he.Key().String()
			if attrSpecs.Includes(key) {
				panic(Error(EVAL_BOTH_CONSTANT_AND_ATTRIBUTE, H{`label`: t.Label(), `key`: key}))
			}
			value := he.Value().(PValue)
			attrSpec := H{
				KEY_TYPE:  Generalize(value.Type()),
				KEY_VALUE: value,
				KEY_KIND:  CONSTANT}
			attrSpec[KEY_OVERRIDE] = parentMembers.Includes(key)
			attrSpecs.Put(key, WrapHash4(attrSpec))
		}
	}

	if !attrSpecs.IsEmpty() {
		attributes := NewStringHash(attrSpecs.Size())
		attrSpecs.EachPair(func(key string, ifv interface{}) {
			value := ifv.(PValue)
			attrSpec, ok := value.(*HashValue)
			if !ok {
				attrType := AssertInstance(
					func() string { return fmt.Sprintf(`attribute %s[%s]`, t.Label(), key) },
					DefaultTypeType(), value)
				hash := H{KEY_TYPE: attrType}
				if _, ok = attrType.(*OptionalType); ok {
					hash[KEY_VALUE] = UNDEF
				}
				attrSpec = WrapHash4(hash)
			}
			attr := newAttribute(key, t, attrSpec)
			assertOverride(attr, parentMembers)
			attributes.Put(key, attr)
		})
		attributes.Freeze()
		t.attributes = attributes
	}

	funcSpecs := hashArg(initHash, KEY_FUNCTIONS)
	if !funcSpecs.IsEmpty() {
		functions := NewStringHash(funcSpecs.Len())
		funcSpecs.EachPair(func(key, value PValue) {
			funcSpec, ok := value.(*HashValue)
			if !ok {
				funcType := AssertInstance(
					func() string { return fmt.Sprintf(`function %s[%s]`, t.Label(), key) },
					TYPE_FUNCTION_TYPE, value)
				funcSpec = WrapHash4(H{KEY_TYPE: funcType})
			}
			fnc := newFunction(key.String(), t, funcSpec)
			assertOverride(fnc, parentMembers)
			functions.Put(key.String(), fnc)
		})
		functions.Freeze()
		t.functions = functions
	}
	t.equalityIncludeType = boolArg(initHash, KEY_EQUALITY_INCLUDE_TYPE, true)

	var equality []string
	eq := initHash.Get2(KEY_EQUALITY, nil)
	if es, ok := eq.(*StringValue); ok {
		equality = []string{es.String()}
	} else if ea, ok := eq.(*ArrayValue); ok {
		equality = make([]string, ea.Len())
	} else {
		equality = nil
	}
	if equality != nil {
    for _, attrName := range equality {
    	var attr Attribute
    	ok := false
    	mbr := parentMembers.Get(attrName, nil)
    	if mbr == nil {
				mbr = t.attributes.Get(attrName, func() interface{} {
					return t.functions.Get(attrName, nil)
				})
				attr, ok = mbr.(Attribute)
			} else {
				attr, ok = mbr.(Attribute)
				// Assert that attribute is not already include by parent equality
				if ok && parentObjectType.EqualityAttributes().Includes(attrName) {
					includingParent := t.findEqualityDefiner(attrName)
					panic(Error(EVAL_EQUALITY_REDEFINED, H{`label`: t.Label(), `attribute`: attr.Label(), `including_parent`: includingParent}))
				}
			}
			if !ok {
				if mbr == nil {
					panic(Error(EVAL_EQUALITY_ATTRIBUTE_NOT_FOUND, H{`label`: t.Label(), `attribute`: attrName}))
				}
				panic(Error(EVAL_EQUALITY_NOT_ATTRIBUTE, H{`label`: t.Label(), `member`: mbr.(AnnotatedMember).Label()}))
			}
			if attr.Kind() == CONSTANT {
				panic(Error(EVAL_EQUALITY_ON_CONSTANT, H{`label`: t.Label(), `attribute`: mbr.(AnnotatedMember).Label()}))
			}
		}
	}
	t.equality = equality
}

func (t *ObjectType) EqualityAttributes() *StringHash {
  eqa := make([]string, 0, 8)
	tp := t
	for tp != nil {
		if tp.equality != nil {
			eqa = append(eqa, tp.equality...)
		}
		tp = tp.resolvedParent()
	}
	attrs := NewStringHash(len(eqa))
	for _, an := range eqa {
		attrs.Put(an, t.GetAttribute(an))
	}
	return attrs
}

func (t *ObjectType) GetAttribute(name string) Attribute {
	a, _ := t.attributes.Get2(name, func() interface{} {
		p := t.resolvedParent()
		if p != nil {
			return p.GetAttribute(name)
		}
		return nil
	}).(Attribute)
	return a
}

func (t *ObjectType) GetFunction(name string) Function {
	f, _ := t.functions.Get2(name, func() interface{} {
		p := t.resolvedParent()
		if p != nil {
			return p.GetFunction(name)
		}
		return nil
	}).(Function)
	return f
}

func (t *ObjectType) GetMember(name string) AnnotatedMember {
	m, _ := t.attributes.Get2(name, func() interface{} {
		return t.functions.Get2(name, func() interface{} {
			p := t.resolvedParent()
			if p != nil {
				return p.GetMember(name)
			}
			return nil
		})}).(AnnotatedMember)
	return m
}

func (t *ObjectType) Parameters() []PValue {
	return t.Parameters2(true)
}

func (t *ObjectType) Parameters2(includeName bool) []PValue {
	return []PValue{t.InitHash(includeName)}
}

func compressedMembersHash(mh *StringHash) *HashValue {
	he := make([]*HashEntry, 0, mh.Size())
	mh.EachPair(func(key string, value interface{}) {
		fh := value.(AnnotatedMember).InitHash()
		if fh.Len() == 1 {
			tp := fh.Get2(KEY_TYPE, nil)
			if tp != nil {
				he = append(he, WrapHashEntry2(key, tp))
				return
			}
		}
		he = append(he, WrapHashEntry2(key, fh))
	})
	return WrapHash(he)
}

func (t *ObjectType) checkSelfRecursion(originator *ObjectType) {
	if t.parent != nil {
		op := t.resolvedParent()
		if Equals(op, originator) {
			panic(Error(EVAL_OBJECT_INHERITS_SELF, H{`label`: originator.Label()}))
		}
		op.checkSelfRecursion(originator)
	}
}

func (t *ObjectType) findEqualityDefiner(attrName string) *ObjectType {
	tp := t
	for tp != nil {
		p := tp.resolvedParent()
		if p == nil || !p.EqualityAttributes().Includes(attrName) {
			return tp
		}
		tp = p
	}
	return nil
}

func (t *ObjectType) initHash(includeName bool) map[string]PValue {
	h := t.annotatable.initHash()
	if includeName && t.name != `` {
		h[KEY_NAME] = WrapString(t.name)
	}
	if t.parent != nil {
		h[KEY_PARENT] = t.parent
	}
	if !t.parameters.IsEmpty() {
		h[KEY_TYPE_PARAMETERS] = compressedMembersHash(t.parameters)
	}
	if !t.attributes.IsEmpty() {
		// Divide attributes into constants and others
		constants := make([]*HashEntry, 0)
		others := NewStringHash(5)
		t.attributes.EachPair(func(key string, value interface{}) {
			a := value.(Attribute)
			if a.Kind() == CONSTANT && Equals(a.Type(), Generalize(a.Value().Type())) {
				constants = append(constants, WrapHashEntry2(key, a.Value()))
			} else {
				others.Put(key, a)
			}
			if !others.IsEmpty() {
				h[KEY_ATTRIBUTES] = compressedMembersHash(others)
			}
			if len(constants) > 0 {
				h[KEY_CONSTANTS] = WrapHash(constants)
			}
		})
	}
	if !t.functions.IsEmpty() {
		h[KEY_FUNCTIONS] = compressedMembersHash(t.functions)
	}
	if t.equality != nil {
		ev := make([]PValue, len(t.equality))
		for i, e := range t.equality {
			ev[i] = WrapString(e)
		}
		h[KEY_EQUALITY] = WrapArray(ev)
	}
	return h
}

func (t *ObjectType) resolvedParent() *ObjectType {
	tp := t.parent
	for {
		switch tp.(type) {
		case nil:
			return nil
		case *ObjectType:
			return tp.(*ObjectType)
		case *TypeAliasType:
			tp = tp.(*TypeAliasType).resolvedType
		default:
			panic(Error(EVAL_ILLEGAL_OBJECT_INHERITANCE, H{`label`: t.Label(), `type`: tp.Type().String()}))
		}
	}
}

func (t *ObjectType) members(includeParent bool) *StringHash {
	collector := NewStringHash(7)
	t.collectMembers(includeParent, collector)
	return collector
}

func (t *ObjectType) typeParameters(includeParent bool) *StringHash {
	collector := NewStringHash(5)
	t.collectParameters(includeParent, collector)
	return collector
}

func (t *ObjectType) collectAttributes(includeParent bool, collector *StringHash) {
	if includeParent && t.parent != nil {
		t.resolvedParent().collectAttributes(true, collector)
	}
	collector.PutAll(t.attributes)
}

func (t *ObjectType) collectFunctions(includeParent bool, collector *StringHash) {
	if includeParent && t.parent != nil {
		t.resolvedParent().collectFunctions(true, collector)
	}
	collector.PutAll(t.functions)
}

func (t *ObjectType) collectMembers(includeParent bool, collector *StringHash) {
	if includeParent && t.parent != nil {
		t.resolvedParent().collectMembers(true, collector)
	}
	collector.PutAll(t.attributes)
	collector.PutAll(t.functions)
}

func (t *ObjectType) collectParameters(includeParent bool, collector *StringHash) {
	if includeParent && t.parent != nil {
		t.resolvedParent().collectParameters(true, collector)
	}
	collector.PutAll(t.parameters)
}

func NewObjectValue(typ PType, values *HashValue) *ObjectValue {
	return &ObjectValue{typ, values}
}

func (o *ObjectValue) Get(key string) (PValue, bool) {
	return o.values.Get3(key)
}

func (o *ObjectValue) Equals(other interface{}, g Guard) bool {
	if ov, ok := other.(*ObjectValue); ok {
		return o.typ.Equals(ov.typ, g) && o.values.Equals(ov.values, g)
	}
	return false
}

func (o *ObjectValue) String() string {
	return ToString(o)
}

func (o *ObjectValue) ToString(bld io.Writer, format FormatContext, g RDetect) {
	Generalize(o.typ).ToString(bld, format, g)
	utils.WriteByte(bld, '(')
	o.values.ToString(bld, format, g)
	utils.WriteByte(bld, ')')
}

func (o *ObjectValue) Type() PType {
	return o.typ
}

func (o *ObjectValue) InitHash() *HashValue {
  return o.values
}
