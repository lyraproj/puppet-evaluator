package types

import (
	"io"
	"regexp"

	. "github.com/puppetlabs/go-evaluator/evaluator"
	. "github.com/puppetlabs/go-parser/parser"
	"github.com/puppetlabs/go-parser/validator"
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
	KEY_VALUE                 = `value`

	CONSTANT         = AttributeKind(`constant`)
	DERIVED          = AttributeKind(`derived`)
	GIVEN_OR_DERIVED = AttributeKind(`given_or_derived`)
	REFERENCE        = AttributeKind(`reference`)
)

var QREF_PATTERN = regexp.MustCompile(`\A[A-Z][\w]*(?:::[A-Z][\w]*)*\z`)

var annotationType_DEFAULT = &ObjectType{name: `Annotation`, attributes: []*Attribute{}, functions: []*ObjFunc{}, equality: nil, annotations: _EMPTY_MAP}

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

var TYPE_ATTRIBUTES = NewHashType(TYPE_MEMBER_NAME, DefaultNotUndefType(), nil)
var TYPE_CONSTANTS = NewHashType(TYPE_MEMBER_NAME, DefaultAnyType(), nil)
var TYPE_FUNCTIONS = NewHashType(TYPE_MEMBER_NAME, DefaultNotUndefType(), nil)
var TYPE_EQUALITY = NewVariantType2(TYPE_MEMBER_NAME, NewArrayType2(TYPE_MEMBER_NAME))
var TYPE_CHECKS = DefaultAnyType()

var TYPE_OBJECT_INIT_HASH = NewStructType([]*StructElement{
	NewStructElement(NewOptionalType3(KEY_NAME), TYPE_OBJECT_NAME),
	NewStructElement(NewOptionalType3(KEY_PARENT), DefaultTypeType()),
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
		Name() string

		Container() *ObjectType

		Type() PType
	}

	AttributeKind string

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

	Attribute struct {
		annotatedMember
		kind AttributeKind
	}

	ObjFunc struct {
		annotatedMember
	}

	ObjectType struct {
		name               string
		parent             PType
		attributes         []*Attribute
		functions          []*ObjFunc
		equality           []string
		annotations        *HashValue
		loader             Loader
		initHashExpression Expression
	}
)

func (a *annotatedMember) Annotations() *HashValue {
	return a.annotations
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

func (a *Attribute) Kind() AttributeKind {
	return a.kind
}

var objectType_DEFAULT = &ObjectType{name: `Object`, initHashExpression: nil, attributes: []*Attribute{}, functions: []*ObjFunc{}, equality: nil, annotations: _EMPTY_MAP}

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

func (t *ObjectType) Equals(other interface{}, guard Guard) bool {
	ot, ok := other.(*ObjectType)
	return ok && t.name == ot.name && t.loader == ot.loader
}

func (t *ObjectType) Name() string {
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
