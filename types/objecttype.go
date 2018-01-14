package types

import (
	"io"
	"regexp"

	"github.com/puppetlabs/go-evaluator/errors"
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
	KEY_TYPE_PARAMETERS       = `type_parameters`
	KEY_VALUE                 = `value`

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
		Name() string

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
	if v == UNDEF || v == nil {
		return nil
	}
	if t, ok := v.(*HashValue); ok {
		return t
	}
	panic(argError(DefaultHashType(), v))
}

func boolArg(v PValue, d bool) bool {
	if v == UNDEF || v == nil {
		return d
	}
	if t, ok := v.(*BooleanValue); ok {
		return t.Bool()
	}
	panic(argError(DefaultBooleanType(), v))
}

// Visit the keys of an annotations map. All keys are known to be types
func visitAnnotations(a *HashValue, v Visitor, g Guard) {
	if a != nil {
		for _, e := range a.EntriesSlice() {
			e.key.(PType).Accept(v, g)
		}
	}
}

func (a *annotatedMember) init(name string, container *ObjectType, initHash map[string]PValue) {
	a.name = name
	a.container = container
	a.typ = typeArg(initHash[`type`])
	a.override = boolArg(initHash[`override`], false)
	a.final = boolArg(initHash[`final`], false)
	a.annotations = hashArg(initHash[`annotations`])
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
			panic(errors.NewArgumentsError(``, `expected %{label} to override an inherited %{feature_type}, but no such %{feature_type} was found`))
		}
	}

	/*
		    # Checks if the this _member_ overrides an inherited member, and if so, that this member is declared with override = true and that
	    # the inherited member accepts to be overridden by this member.
	    #
	    # @param parent_members [Hash{String=>PAnnotatedMember}] the hash of inherited members
	    # @return [PAnnotatedMember] this instance
	    # @raises [Puppet::ParseError] if the assertion fails
	    # @api private
	    def assert_override(parent_members)
	      parent_member = parent_members[@name]
	      if parent_member.nil?
	        if @override
	          raise Puppet::ParseError, _("expected %{label} to override an inherited %{feature_type}, but no such %{feature_type} was found") %
	              { label: label, feature_type: feature_type }
	        end
	        self
	      else
	        parent_member.assert_can_be_overridden(self)
	      end
	    end

	*/
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

func (a *attribute) Kind() AttributeKind {
	return a.kind
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
