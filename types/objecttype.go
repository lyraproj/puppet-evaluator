package types

import (
	"io"
	"regexp"

	"github.com/puppetlabs/go-evaluator/errors"
	. "github.com/puppetlabs/go-evaluator/evaluator"
	. "github.com/puppetlabs/go-parser/parser"
	"github.com/puppetlabs/go-parser/validator"
	. "github.com/puppetlabs/go-parser/issue"
	"fmt"
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
	name:           `Annotation`,
	typeParameters: map[string]AnnotatedMember{},
	attributes:     map[string]AnnotatedMember{},
	functions:      map[string]AnnotatedMember{},
	equality:       nil,
	annotations:    _EMPTY_MAP}

var TYPE_ATTRIBUTE_KIND = NewEnumType([]string{string(CONSTANT), string(DERIVED), string(GIVEN_OR_DERIVED), string(REFERENCE)})
var TYPE_OBJECT_NAME = NewPatternType([]*RegexpType{NewRegexpTypeR(QREF_PATTERN)})
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
		HashKeyValue

		Name() string

		Label() string

		FeatureType() string

		Container() *ObjectType

		Type() PType

		Override() bool

		Final() bool

		accept(v Visitor, g Guard)
	}

	AttributeKind string

	Attribute interface {
		AnnotatedMember
		Kind() AttributeKind
	}

	Method interface {
		AnnotatedMember
	}

	PuppetObject interface {
		PValue

		InitHash() *HashValue
	}

	annotatedMember struct {
		name        string
		container   *ObjectType
		typ         PType
		override    bool
		final       bool
		annotations *HashValue
	}

	attribute struct {
		annotatedMember
		kind AttributeKind
		value PValue
	}

	typeParameter struct {
		attribute
	}

	function struct {
		annotatedMember
	}

	ObjectType struct {
		name               string
		parent             PType
		typeParameters     map[string]AnnotatedMember
		attributes         map[string]AnnotatedMember
		functions          map[string]AnnotatedMember
		equality           []string
		annotations        *HashValue
		loader             Loader
		initHashExpression Expression
	}
)

func argError(e PType, a PValue) errors.InstantiationError {
	return errors.NewArgumentsError(``, DescribeMismatch(`assert`, e, a.Type()))
}

func typeArg(v PValue) PType {
	if t, ok := v.(PType); ok {
		return t
	}
	panic(argError(DefaultTypeType(), v))
}

func hashArg(v PValue) *HashValue {
	if v == _UNDEF {
		return _EMPTY_MAP
	}
	if t, ok := v.(*HashValue); ok {
		return t
	}
	panic(argError(DefaultHashType(), v))
}

func boolArg(v PValue, d bool) bool {
	if v == _UNDEF {
		return d
	}
	if t, ok := v.(*BooleanValue); ok {
		return t.Bool()
	}
	panic(argError(DefaultBooleanType(), v))
}

func stringArg(v PValue, d string) string {
	if v == _UNDEF {
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

func (a *annotatedMember) initialize(name string, container *ObjectType, initHash *HashValue) {
	a.name = name
	a.container = container
	a.typ = typeArg(initHash.Get2(KEY_TYPE))
	a.override = boolArg(initHash.Get2(KEY_OVERRIDE), false)
	a.final = boolArg(initHash.Get2(KEY_FINAL), false)
	a.annotations = hashArg(initHash.Get2(KEY_ANNOTATIONS))
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
func assertOverride(a AnnotatedMember, parentMembers map[string]AnnotatedMember) {
	parentMember := parentMembers[a.Name()]
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

func (a *annotatedMember) InitHash() *HashValue {
	return WrapHash([]*HashEntry{})
}

func (a *annotatedMember) Final() bool {
	return a.final
}

func (a *attribute) initialize(name string, container *ObjectType, initHash *HashValue) {
	a.annotatedMember.initialize(name, container, initHash)
	a.kind = AttributeKind(stringArg(initHash.Get2(KEY_KIND), ``))
	if a.kind == CONSTANT { // final is implied
    if initHash.IncludesKey2(KEY_FINAL) && !a.final {
    	panic(Error(EVAL_CONSTANT_WITH_FINAL, H{`label`: a.Label()}))
		}
		a.final = true
	}
	if initHash.IncludesKey2(KEY_VALUE) {
		if a.kind == DERIVED || a.kind == GIVEN_OR_DERIVED {
			panic(Error(EVAL_ILLEGAL_KIND_VALUE_COMBINATION, H{`label`: a.Label(), `kind`: a.kind}))
		}
		v := initHash.Get2(KEY_VALUE)
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

func (a *attribute) Kind() AttributeKind {
	return a.kind
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

func (a *attribute) ToKey() HashKey {
	// TODO: Improve this
	return HashKey(a.Label())
}

func (a *function) FeatureType() string {
	return `function`
}

func (a *function) Label() string {
	return fmt.Sprintf(`function %s[%s]`, a.container.Label(), a.Name())
}

var objectType_DEFAULT = &ObjectType{
	name:           `Object`,
	typeParameters: map[string]AnnotatedMember{},
	attributes:     map[string]AnnotatedMember{},
	functions:      map[string]AnnotatedMember{},
	equality:       nil,
	annotations:    _EMPTY_MAP}

func DefaultObjectType() *ObjectType {
	return objectType_DEFAULT
}

func NewObjectType(name string, initHashExpression Expression) *ObjectType {
	return &ObjectType{name: name, initHashExpression: initHashExpression, attributes: nil, functions: nil, equality: nil, annotations: _EMPTY_MAP}
}

func NewObjectType2(initHash map[string]PValue) *ObjectType {
	result := &ObjectType{}
	result.Pcore_initFromHash(initHash)
	return result
}

func NewObjectType3(initHash map[string]PValue, loader Loader) *ObjectType {
	result := &ObjectType{}
	result.Pcore_initFromHash(initHash)
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
	for _, m := range t.typeParameters {
		m.accept(v, g)
	}
	for _, m := range t.attributes {
		m.accept(v, g)
	}
	for _, m := range t.functions {
		m.accept(v, g)
	}
}

func (t *ObjectType) Default() PType {
	return objectType_DEFAULT
}

func (t *ObjectType) Equals(other interface{}, guard Guard) bool {
	ot, ok := other.(*ObjectType)
	return ok && t.name == ot.name && t.loader == ot.loader
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

func (t *ObjectType) ToString(bld io.Writer, format FormatContext, g RDetect) {
	/*
		if fmt == EXPANDED {
			appendObjectHash(bld, t.InitHash())
		} else {
			if ts, ok := fmt.TypeSet(); ok {
				WriteString(bld, ts.NameFor(t, t.Name()))
			}
		}
		/*
			# @api private
			def string_PObjectType(t)
			if @expanded
			append_object_hash(t._pcore_init_hash(@type_set.nil? || !@type_set.defines_type?(t)))
			else
			@bld << (@type_set ? @type_set.name_for(t, t.label) : t.label)
			end
			end
	*/
}

func (t *ObjectType) Type() PType {
	return &TypeType{t}
}

func (t *ObjectType) String() string {
	return ToString2(t, EXPANDED)
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

func (t *ObjectType) Resolve(resolver TypeResolver) {

}

func (t *ObjectType) Pcore_initFromHash(initHash map[string]PValue) {

}
