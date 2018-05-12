package types

import (
	"fmt"
	"io"
	"regexp"
	"sync/atomic"

	"github.com/puppetlabs/go-evaluator/errors"
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/hash"
	"github.com/puppetlabs/go-issues/issue"
	"github.com/puppetlabs/go-parser/parser"
	"github.com/puppetlabs/go-parser/validator"
	"github.com/puppetlabs/go-semver/semver"
)

var Object_Type eval.ObjectType

func init() {
	oneArgCtor := func(ctx eval.Context, args []eval.PValue) eval.PValue {
		return NewObjectType2(ctx, args[0].(*HashValue), ctx.Loader())
	}
	Object_Type = newObjectType2(`Pcore::ObjectType`, Any_Type,
		WrapHash3(map[string]eval.PValue{
			`attributes`: SingletonHash2(`_pcore_init_hash`, TYPE_OBJECT_INIT_HASH)}),
		oneArgCtor, oneArgCtor)
}

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
	KEY_SERIALIZATION         = `serialization`
	KEY_TYPE                  = `type`
	KEY_TYPE_PARAMETERS       = `type_parameters`
	KEY_VALUE                 = `value`

	DEFAULT_KIND     = eval.AttributeKind(``)
	CONSTANT         = eval.AttributeKind(`constant`)
	DERIVED          = eval.AttributeKind(`derived`)
	GIVEN_OR_DERIVED = eval.AttributeKind(`given_or_derived`)
	REFERENCE        = eval.AttributeKind(`reference`)
)

var QREF_PATTERN = regexp.MustCompile(`\A[A-Z][\w]*(?:::[A-Z][\w]*)*\z`)

var annotationType_DEFAULT = &objectType{
	annotatable: annotatable{annotations: _EMPTY_MAP},
	hashKey:     eval.HashKey("\x00tAnnotation"),
	name:        `Annotation`,
	parameters:  hash.EMPTY_STRINGHASH,
	attributes:  hash.EMPTY_STRINGHASH,
	functions:   hash.EMPTY_STRINGHASH,
	equality:    nil}

func DefaultAnnotationType() eval.PType {
	return annotationType_DEFAULT
}

var TYPE_ANNOTATIONS = NewHashType(annotationType_DEFAULT, DefaultHashType(), nil)

var TYPE_ATTRIBUTE_KIND = NewEnumType([]string{string(CONSTANT), string(DERIVED), string(GIVEN_OR_DERIVED), string(REFERENCE)}, false)
var TYPE_OBJECT_NAME = NewPatternType([]*RegexpType{NewRegexpTypeR(QREF_PATTERN)})

var TYPE_TYPE_PARAMETER = NewStructType([]*StructElement{
	NewStructElement2(KEY_TYPE, DefaultTypeType()),
	NewStructElement(NewOptionalType3(KEY_ANNOTATIONS), TYPE_ANNOTATIONS),
})

var TYPE_ATTRIBUTE = NewStructType([]*StructElement{
	NewStructElement2(KEY_TYPE, DefaultTypeType()),
	NewStructElement(NewOptionalType3(KEY_FINAL), DefaultBooleanType()),
	NewStructElement(NewOptionalType3(KEY_OVERRIDE), DefaultBooleanType()),
	NewStructElement(NewOptionalType3(KEY_KIND), TYPE_ATTRIBUTE_KIND),
	NewStructElement(NewOptionalType3(KEY_VALUE), DefaultAnyType()),
	NewStructElement(NewOptionalType3(KEY_ANNOTATIONS), TYPE_ANNOTATIONS),
})

var TYPE_MEMBER_NAME = NewPatternType2(NewRegexpTypeR(validator.PARAM_NAME))
var TYPE_ATTRIBUTE_CALLABLE = NewCallableType2(NewIntegerType(0, 0))
var TYPE_FUNCTION_TYPE = NewTypeType(DefaultCallableType())

var TYPE_FUNCTION = NewStructType([]*StructElement{
	NewStructElement2(KEY_TYPE, TYPE_FUNCTION_TYPE),
	NewStructElement(NewOptionalType3(KEY_FINAL), DefaultBooleanType()),
	NewStructElement(NewOptionalType3(KEY_OVERRIDE), DefaultBooleanType()),
	NewStructElement(NewOptionalType3(KEY_ANNOTATIONS), TYPE_ANNOTATIONS),
})

var TYPE_MEMBER_NAMES = NewArrayType2(TYPE_MEMBER_NAME)
var TYPE_PARAMETERS = NewHashType(TYPE_MEMBER_NAME, DefaultNotUndefType(), nil)
var TYPE_ATTRIBUTES = NewHashType(TYPE_MEMBER_NAME, DefaultNotUndefType(), nil)
var TYPE_CONSTANTS = NewHashType(TYPE_MEMBER_NAME, DefaultAnyType(), nil)
var TYPE_FUNCTIONS = NewHashType(NewVariantType2(TYPE_MEMBER_NAME, NewPatternType2(NewRegexpTypeR(regexp.MustCompile(`^\[\]$`)))), DefaultNotUndefType(), nil)
var TYPE_EQUALITY = NewVariantType2(TYPE_MEMBER_NAME, TYPE_MEMBER_NAMES)
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
	NewStructElement(NewOptionalType3(KEY_EQUALITY), TYPE_EQUALITY),
	NewStructElement(NewOptionalType3(KEY_SERIALIZATION), TYPE_MEMBER_NAMES),
	NewStructElement(NewOptionalType3(KEY_ANNOTATIONS), TYPE_ANNOTATIONS),
})

type (
	annotatable struct {
		annotations *HashValue
	}

	annotatedMember struct {
		annotatable
		name      string
		container *objectType
		typ       eval.PType
		override  bool
		final     bool
	}

	attribute struct {
		annotatedMember
		kind  eval.AttributeKind
		value eval.PValue
	}

	typeParameter struct {
		attribute
	}

	function struct {
		annotatedMember
	}

	attributesInfo struct {
		nameToPos                map[string]int
		posToName                map[int]string
		attributes               []eval.Attribute
		equalityAttributeIndexes []int
		requiredCount            int
	}

	objectType struct {
		annotatable
		hashKey             eval.HashKey
		name                string
		parent              eval.PType
		creators            []eval.DispatchFunction
		parameters          *hash.StringHash // map doesn't preserve order
		attributes          *hash.StringHash
		functions           *hash.StringHash
		equality            []string
		equalityIncludeType bool
		serialization       []string
		loader              eval.Loader
		initHashExpression  interface{} // Expression or *HashValue
		attrInfo            *attributesInfo
	}

	objectValue struct {
		typ    eval.ObjectType
		values []eval.PValue
	}
)

func ObjectToString(o eval.PuppetObject, s eval.FormatContext, writer io.Writer, g eval.RDetect) {
	io.WriteString(writer, o.Type().Name())
	o.InitHash().(*HashValue).ToString2(writer, s, eval.GetFormat(s.FormatMap(), o.Type()), '(', g)
}

func argError(e eval.PType, a eval.PValue) errors.InstantiationError {
	return errors.NewArgumentsError(``, eval.DescribeMismatch(`assert`, e, a.Type()))
}

func typeArg(hash eval.KeyedValue, key string, d eval.PType) eval.PType {
	v := hash.Get5(key, nil)
	if v == nil {
		return d
	}
	if t, ok := v.(eval.PType); ok {
		return t
	}
	panic(argError(DefaultTypeType(), v))
}

func hashArg(hash eval.KeyedValue, key string) *HashValue {
	v := hash.Get5(key, nil)
	if v == nil {
		return _EMPTY_MAP
	}
	if t, ok := v.(*HashValue); ok {
		return t
	}
	panic(argError(DefaultHashType(), v))
}

func boolArg(hash eval.KeyedValue, key string, d bool) bool {
	v := hash.Get5(key, nil)
	if v == nil {
		return d
	}
	if t, ok := v.(*BooleanValue); ok {
		return t.Bool()
	}
	panic(argError(DefaultBooleanType(), v))
}

func stringArg(hash eval.KeyedValue, key string, d string) string {
	v := hash.Get5(key, nil)
	if v == nil {
		return d
	}
	if t, ok := v.(*StringValue); ok {
		return t.String()
	}
	panic(argError(DefaultStringType(), v))
}

func uriArg(c eval.Context, hash eval.KeyedValue, key string, d eval.URI) eval.URI {
	v := hash.Get5(key, nil)
	if v == nil {
		return d
	}
	if t, ok := v.(*StringValue); ok {
		str := t.String()
		if _, err := ParseURI2(str, true); err != nil {
			panic(eval.Error(c, eval.EVAL_INVALID_URI, issue.H{`str`: str, `detail`: err.Error()}))
		}
		return eval.URI(str)
	}
	if t, ok := v.(*UriValue); ok {
		return eval.URI(t.URL().String())
	}
	panic(argError(DefaultUriType(), v))
}

func versionArg(c eval.Context, hash eval.KeyedValue, key string, d semver.Version) semver.Version {
	v := hash.Get5(key, nil)
	if v == nil {
		return d
	}
	if s, ok := v.(*StringValue); ok {
		sv, error := semver.ParseVersion(s.String())
		if error != nil {
			panic(eval.Error(c, eval.EVAL_INVALID_VERSION, issue.H{`str`: s.String(), `detail`: error.Error()}))
		}
		return sv
	}
	if sv, ok := v.(*SemVerValue); ok {
		return sv.Version()
	}
	panic(argError(DefaultSemVerType(), v))
}

func versionRangeArg(c eval.Context, hash eval.KeyedValue, key string, d semver.VersionRange) semver.VersionRange {
	v := hash.Get5(key, nil)
	if v == nil {
		return d
	}
	if s, ok := v.(*StringValue); ok {
		sr, error := semver.ParseVersionRange(s.String())
		if error != nil {
			panic(eval.Error(c, eval.EVAL_INVALID_VERSION_RANGE, issue.H{`str`: s.String(), `detail`: error.Error()}))
		}
		return sr
	}
	if sv, ok := v.(*SemVerRangeValue); ok {
		return sv.VersionRange()
	}
	panic(argError(DefaultSemVerType(), v))
}

// Visit the keys of an annotations map. All keys are known to be types
func visitAnnotations(a *HashValue, v eval.Visitor, g eval.Guard) {
	if a != nil {
		a.EachKey(func(key eval.PValue) {
			key.(eval.PType).Accept(v, g)
		})
	}
}

func (a *annotatable) initialize(initHash *HashValue) {
	a.annotations = hashArg(initHash, KEY_ANNOTATIONS)
}

func (a *annotatable) initHash() map[string]eval.PValue {
	h := make(map[string]eval.PValue, 5)
	if a.annotations.Len() > 0 {
		h[KEY_ANNOTATIONS] = a.annotations
	}
	return h
}

func (a *annotatedMember) initialize(name string, container *objectType, initHash *HashValue) {
	a.annotatable.initialize(initHash)
	a.name = name
	a.container = container
	a.typ = typeArg(initHash, KEY_TYPE, DefaultTypeType())
	a.override = boolArg(initHash, KEY_OVERRIDE, false)
	a.final = boolArg(initHash, KEY_FINAL, false)
}

func (a *annotatedMember) Accept(v eval.Visitor, g eval.Guard) {
	a.typ.Accept(v, g)
	visitAnnotations(a.annotations, v, g)
}

func (a *annotatedMember) Annotations() *HashValue {
	return a.annotations
}

// Checks if the this _member_ overrides an inherited member, and if so, that this member is declared with
// override = true and that the inherited member accepts to be overridden by this member.
func assertOverride(c eval.Context, a eval.AnnotatedMember, parentMembers *hash.StringHash) {
	parentMember, _ := parentMembers.Get(a.Name(), nil).(eval.AnnotatedMember)
	if parentMember == nil {
		if a.Override() {
			panic(eval.Error(c, eval.EVAL_OVERRIDDEN_NOT_FOUND, issue.H{`label`: a.Label(), `feature_type`: a.FeatureType()}))
		}
	} else {
		assertCanBeOverridden(c, parentMember, a)
	}
}

func assertCanBeOverridden(c eval.Context, a eval.AnnotatedMember, member eval.AnnotatedMember) {
	if a.FeatureType() != member.FeatureType() {
		panic(eval.Error(c, eval.EVAL_OVERRIDE_MEMBER_MISMATCH, issue.H{`member`: member.Label(), `label`: a.Label()}))
	}
	if a.Final() {
		aa, ok := a.(eval.Attribute)
		if !(ok && aa.Kind() == CONSTANT && member.(eval.Attribute).Kind() == CONSTANT) {
			panic(eval.Error(c, eval.EVAL_OVERRIDE_OF_FINAL, issue.H{`member`: member.Label(), `label`: a.Label()}))
		}
	}
	if !member.Override() {
		panic(eval.Error(c, eval.EVAL_OVERRIDE_IS_MISSING, issue.H{`member`: member.Label(), `label`: a.Label()}))
	}
	if !eval.IsAssignable(a.Type(), member.Type()) {
		panic(eval.Error(c, eval.EVAL_OVERRIDE_TYPE_MISMATCH, issue.H{`member`: member.Label(), `label`: a.Label()}))
	}
}

func (a *annotatedMember) Call(c eval.Context, receiver eval.PValue, block eval.Lambda, args []eval.PValue) eval.PValue {
	// TODO:
	panic("implement me")
}

func (a *annotatedMember) Name() string {
	return a.name
}

func (a *annotatedMember) Container() eval.ObjectType {
	return a.container
}

func (a *annotatedMember) Type() eval.PType {
	return a.typ
}

func (a *annotatedMember) Override() bool {
	return a.override
}

func (a *annotatedMember) initHash() map[string]eval.PValue {
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

func (t *typeParameter) initHash() map[string]eval.PValue {
	hash := t.attribute.initHash()
	hash[KEY_TYPE] = hash[KEY_TYPE].(*TypeType).Type()
	if v, ok := hash[KEY_VALUE]; ok && eval.Equals(v, _UNDEF) {
		delete(hash, KEY_VALUE)
	}
	return hash
}

func (t *typeParameter) InitHash() eval.KeyedValue {
	return WrapHash3(t.initHash())
}

func (t *typeParameter) FeatureType() string {
	return `type_parameter`
}

func newFunction(c eval.Context, name string, container *objectType, initHash *HashValue) eval.ObjFunc {
	f := &function{}
	f.initialize(c, name, container, initHash)
	return f
}

func (f *function) initialize(c eval.Context, name string, container *objectType, initHash *HashValue) {
	f.annotatedMember.initialize(name, container, initHash)
	eval.AssertInstance(c, func() string { return fmt.Sprintf(`initializer for %s`, f.Label()) }, TYPE_FUNCTION, initHash)
}

func (a *function) Call(c eval.Context, receiver eval.PValue, block eval.Lambda, args []eval.PValue) eval.PValue {
	if a.CallableType().(*CallableType).CallableWith(c, args, block) {
		if co, ok := receiver.(eval.CallableObject); ok {
			if result, ok := co.Call(c, a.name, args, block); ok {
				return result
			}
		}
		panic(eval.Error(c, eval.EVAL_INSTANCE_DOES_NOT_RESPOND, issue.H{`instance`: receiver, `message`: a.name}))
	}
	panic(eval.Error(c, eval.EVAL_TYPE_MISMATCH, issue.H{`detail`: eval.DescribeSignatures(
		[]eval.Signature{a.CallableType().(*CallableType)}, NewTupleType2(args...), block)}))
}

func (f *function) Equals(other interface{}, g eval.Guard) bool {
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

func (f *function) CallableType() eval.PType {
	return f.typ.(*CallableType)
}

func (f *function) InitHash() eval.KeyedValue {
	return WrapHash3(f.initHash())
}

var objectType_DEFAULT = &objectType{
	annotatable: annotatable{annotations: _EMPTY_MAP},
	name:        `Object`,
	hashKey:     eval.HashKey("\x00tObject"),
	parameters:  hash.EMPTY_STRINGHASH,
	attributes:  hash.EMPTY_STRINGHASH,
	functions:   hash.EMPTY_STRINGHASH}

func DefaultObjectType() *objectType {
	return objectType_DEFAULT
}

var objectId = int64(0)

func AllocObjectType() *objectType {
	return &objectType{
		annotatable: annotatable{annotations: _EMPTY_MAP},
		hashKey:     eval.HashKey(fmt.Sprintf("\x00tObject%d", atomic.AddInt64(&objectId, 1))),
		parameters:  hash.EMPTY_STRINGHASH,
		attributes:  hash.EMPTY_STRINGHASH,
		functions:   hash.EMPTY_STRINGHASH}
}

func (t *objectType) Initialize(c eval.Context, args []eval.PValue) {
	if len(args) == 1 {
		if hash, ok := args[0].(eval.KeyedValue); ok {
			t.InitFromHash(c, hash)
			return
		}
	}
	panic(eval.Error(c, eval.EVAL_FAILURE, issue.H{`message`: `internal error when creating an Object data type`}))
}

func NewObjectType(name string, parent eval.PType, initHashExpression interface{}) *objectType {
	obj := AllocObjectType()
	obj.name = name
	obj.initHashExpression = initHashExpression
	obj.parent = parent
	return obj
}

func NewObjectType2(c eval.Context, initHash *HashValue, loader eval.Loader) *objectType {
	if initHash.IsEmpty() {
		return DefaultObjectType()
	}
	obj := AllocObjectType()
	obj.InitFromHash(c, initHash)
	obj.loader = loader
	return obj
}

func (t *objectType) Accept(v eval.Visitor, g eval.Guard) {
	if g == nil {
		g = make(eval.Guard)
	}
	if g.Seen(t, nil) {
		return
	}
	v(t)
	visitAnnotations(t.annotations, v, g)
	if t.parent != nil {
		t.parent.Accept(v, g)
	}
	t.parameters.EachValue(func(p interface{}) { p.(eval.AnnotatedMember).Accept(v, g) })
	t.attributes.EachValue(func(a interface{}) { a.(eval.AnnotatedMember).Accept(v, g) })
	t.functions.EachValue(func(f interface{}) { f.(eval.AnnotatedMember).Accept(v, g) })
}

func (t *objectType) Default() eval.PType {
	return objectType_DEFAULT
}

func (t *objectType) Equals(other interface{}, guard eval.Guard) bool {
	ot, ok := other.(*objectType)
	if !ok {
		return false
	}

	if t.name == `` || ot.name == `` {
		return t == ot
	}

	if t.name == ot.name {
		if t.loader == nil {
			return ot.loader == nil
		}
		return ot.loader != nil && t.loader.NameAuthority() == ot.loader.NameAuthority()
	}
	return false
}

func (t *objectType) GetValue(c eval.Context, key string, o eval.PValue) (value eval.PValue, ok bool) {
	if pu, ok := o.(eval.ReadableObject); ok {
		return pu.Get(c, key)
	}

	// TODO: Perhaps use other ways of extracting attributes with reflection
	// in case native types must be described by Object
	return nil, false
}

func (t *objectType) MetaType() eval.ObjectType {
	return Object_Type
}

func (t *objectType) Name() string {
	return t.name
}

func (t *objectType) Label() string {
	if t.name == `` {
		return `Object`
	}
	return t.name
}

func (t *objectType) Member(name string) (eval.CallableMember, bool) {
	mbr := t.attributes.Get2(name, func() interface{} {
		return t.functions.Get2(name, func() interface{} {
			if t.parent == nil {
				return nil
			}
			pm, _ := t.resolvedParent(nil).Member(name)
			return pm
		})
	})
	if mbr == nil {
		return nil, false
	}
	return mbr.(eval.CallableMember), true
}

func (t *objectType) ToKey() eval.HashKey {
	return t.hashKey
}

func (t *objectType) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *objectType) Type() eval.PType {
	return &TypeType{t}
}

func (t *objectType) String() string {
	return eval.ToString2(t, EXPANDED)
}

func (t *objectType) Get(c eval.Context, key string) (value eval.PValue, ok bool) {
	if key == `_pcore_init_hash` {
		return t.InitHash(), true
	}
	return nil, false
}

func (t *objectType) InitHash() eval.KeyedValue {
	return WrapHash3(t.initHash(true))
}

func (t *objectType) IsInstance(c eval.Context, o eval.PValue, g eval.Guard) bool {
	return isAssignable(t, o.Type())
}

func (t *objectType) IsAssignable(o eval.PType, g eval.Guard) bool {
	var ot *objectType
	switch o.(type) {
	case *objectType:
		ot = o.(*objectType)
	case *objectTypeExtension:
		ot = o.(*objectTypeExtension).baseType
	default:
		return false
	}

	if t == DefaultObjectType() || t.Equals(ot, g) {
		return true
	}
	if ot.parent != nil {
		return t.IsAssignable(ot.parent, g)
	}
	return false
}

func (t *objectType) IsParameterized() bool {
	if !t.parameters.IsEmpty() {
		return true
	}
	p := t.resolvedParent(nil)
	if p != nil {
		return p.IsParameterized()
	}
	return false
}

func (t *objectType) Resolve(c eval.Context) eval.PType {
	if t.initHashExpression != nil {
		ihe := t.initHashExpression
		t.initHashExpression = nil

		var initHash *HashValue
		if lh, ok := ihe.(*parser.LiteralHash); ok {
			initHash = c.Evaluate(lh).(*HashValue)
			if prt, ok := t.parent.(eval.ResolvableType); ok {
				t.parent = resolveTypeRefs(c, prt).(eval.PType)
			}
		} else {
			initHash = resolveTypeRefs(c, ihe.(*HashValue)).(*HashValue)
		}
		t.loader = c.Loader()
		t.InitFromHash(c, initHash)
		if t.name != `` {
			t.createNewFunction(c)
		}
	}
	return t
}

func resolveTypeRefs(c eval.Context, v eval.PValue) eval.PValue {
	switch v.(type) {
	case *HashValue:
		hv := v.(*HashValue)
		he := make([]*HashEntry, hv.Len())
		i := 0
		hv.EachPair(func(key, value eval.PValue) {
			he[i] = WrapHashEntry(
				resolveTypeRefs(c, key), resolveTypeRefs(c, value))
			i++
		})
		return WrapHash(he)
	case *ArrayValue:
		av := v.(*ArrayValue)
		ae := make([]eval.PValue, av.Len())
		i := 0
		av.Each(func(value eval.PValue) {
			ae[i] = resolveTypeRefs(c, value)
			i++
		})
		return WrapArray(ae)
	case eval.ResolvableType:
		return v.(eval.ResolvableType).Resolve(c)
	default:
		return v
	}
}

func (t *objectType) InitFromHash(c eval.Context, initHash eval.KeyedValue) {
	eval.AssertInstance(c, `object initializer`, TYPE_OBJECT_INIT_HASH, initHash)
	t.parameters = hash.EMPTY_STRINGHASH
	t.attributes = hash.EMPTY_STRINGHASH
	t.functions = hash.EMPTY_STRINGHASH
	t.name = stringArg(initHash, KEY_NAME, t.name)

	if t.parent == nil {
		t.parent = typeArg(initHash, KEY_PARENT, nil)
	}

	parentMembers := hash.EMPTY_STRINGHASH
	parentTypeParams := hash.EMPTY_STRINGHASH
	var parentObjectType *objectType

	if t.parent != nil {
		t.checkSelfRecursion(c, t)
		parentObjectType = t.resolvedParent(c)
		parentMembers = parentObjectType.members(true)
		parentTypeParams = parentObjectType.typeParameters(true)
	}

	typeParameters := hashArg(initHash, KEY_TYPE_PARAMETERS)
	if !typeParameters.IsEmpty() {
		parameters := hash.NewStringHash(typeParameters.Len())
		typeParameters.EachPair(func(k, v eval.PValue) {
			key := k.String()
			var paramType eval.PType
			var paramValue eval.PValue
			if ph, ok := v.(*HashValue); ok {
				eval.AssertInstance(c,
					func() string { return fmt.Sprintf(`type_parameter %s[%s]`, t.Label(), key) },
					TYPE_TYPE_PARAMETER, ph)
				paramType = typeArg(ph, KEY_TYPE, DefaultTypeType())
				paramValue = ph.Get5(KEY_VALUE, nil)
			} else {
				paramType = eval.AssertInstance(c,
					func() string { return fmt.Sprintf(`type_parameter %s[%s]`, t.Label(), key) },
					DefaultTypeType(), v).(eval.PType)
				paramValue = nil
			}
			if _, ok := paramType.(*OptionalType); !ok {
				paramType = NewOptionalType(paramType)
			}
			param := newTypeParameter(c, key, t, WrapHash4(issue.H{
				KEY_TYPE:  paramType,
				KEY_VALUE: paramValue}))
			assertOverride(c, param, parentTypeParams)
			parameters.Put(key, param)
		})
		parameters.Freeze()
		t.parameters = parameters
	}

	constants := hashArg(initHash, KEY_CONSTANTS)
	attributes := hashArg(initHash, KEY_ATTRIBUTES)
	attrSpecs := hash.NewStringHash(constants.Len() + attributes.Len())
	attributes.EachPair(func(k, v eval.PValue) {
		attrSpecs.Put(k.String(), v)
	})

	if !constants.IsEmpty() {
		constants.EachPair(func(k, v eval.PValue) {
			key := k.String()
			if attrSpecs.Includes(key) {
				panic(eval.Error(c, eval.EVAL_BOTH_CONSTANT_AND_ATTRIBUTE, issue.H{`label`: t.Label(), `key`: key}))
			}
			value := v.(eval.PValue)
			attrSpec := issue.H{
				KEY_TYPE:  eval.Generalize(value.Type()),
				KEY_VALUE: value,
				KEY_KIND:  CONSTANT}
			attrSpec[KEY_OVERRIDE] = parentMembers.Includes(key)
			attrSpecs.Put(key, WrapHash4(attrSpec))
		})
	}

	if !attrSpecs.IsEmpty() {
		attributes := hash.NewStringHash(attrSpecs.Len())
		attrSpecs.EachPair(func(key string, ifv interface{}) {
			value := ifv.(eval.PValue)
			attrSpec, ok := value.(*HashValue)
			if !ok {
				attrType := eval.AssertInstance(c,
					func() string { return fmt.Sprintf(`attribute %s[%s]`, t.Label(), key) },
					DefaultTypeType(), value)
				hash := issue.H{KEY_TYPE: attrType}
				if _, ok = attrType.(*OptionalType); ok {
					hash[KEY_VALUE] = eval.UNDEF
				}
				attrSpec = WrapHash4(hash)
			}
			attr := newAttribute(c, key, t, attrSpec)
			assertOverride(c, attr, parentMembers)
			attributes.Put(key, attr)
		})
		attributes.Freeze()
		t.attributes = attributes
	}

	funcSpecs := hashArg(initHash, KEY_FUNCTIONS)
	if !funcSpecs.IsEmpty() {
		functions := hash.NewStringHash(funcSpecs.Len())
		funcSpecs.EachPair(func(key, value eval.PValue) {
			if attributes.IncludesKey(key) {
				panic(eval.Error(c, eval.EVAL_MEMBER_NAME_CONFLICT, issue.H{`label`: fmt.Sprintf(`function %s[%s]`, t.Label(), key)}))
			}
			funcSpec, ok := value.(*HashValue)
			if !ok {
				funcType := eval.AssertInstance(c,
					func() string { return fmt.Sprintf(`function %s[%s]`, t.Label(), key) },
					TYPE_FUNCTION_TYPE, value)
				funcSpec = WrapHash4(issue.H{KEY_TYPE: funcType})
			}
			fnc := newFunction(c, key.String(), t, funcSpec)
			assertOverride(c, fnc, parentMembers)
			functions.Put(key.String(), fnc)
		})
		functions.Freeze()
		t.functions = functions
	}
	t.equalityIncludeType = boolArg(initHash, KEY_EQUALITY_INCLUDE_TYPE, true)

	var equality []string
	eq := initHash.Get5(KEY_EQUALITY, nil)
	if es, ok := eq.(*StringValue); ok {
		equality = []string{es.String()}
	} else if ea, ok := eq.(*ArrayValue); ok {
		equality = make([]string, ea.Len())
	} else {
		equality = nil
	}
	if equality != nil {
		for _, attrName := range equality {
			var attr eval.Attribute
			ok := false
			mbr := t.attributes.Get2(attrName, func() interface{} {
				return t.functions.Get2(attrName, func() interface{} {
					return parentMembers.Get(attrName, nil)
				})
			})
			attr, ok = mbr.(eval.Attribute)

			if !ok {
				if mbr == nil {
					panic(eval.Error(c, eval.EVAL_EQUALITY_ATTRIBUTE_NOT_FOUND, issue.H{`label`: t.Label(), `attribute`: attrName}))
				}
				panic(eval.Error(c, eval.EVAL_EQUALITY_NOT_ATTRIBUTE, issue.H{`label`: t.Label(), `member`: mbr.(eval.AnnotatedMember).Label()}))
			}
			if attr.Kind() == CONSTANT {
				panic(eval.Error(c, eval.EVAL_EQUALITY_ON_CONSTANT, issue.H{`label`: t.Label(), `attribute`: mbr.(eval.AnnotatedMember).Label()}))
			}
			// Assert that attribute is not already include by parent equality
			if ok && parentObjectType.EqualityAttributes().Includes(attrName) {
				includingParent := t.findEqualityDefiner(attrName)
				panic(eval.Error(c, eval.EVAL_EQUALITY_REDEFINED, issue.H{`label`: t.Label(), `attribute`: attr.Label(), `including_parent`: includingParent}))
			}
		}
	}
	t.equality = equality

	se, ok := initHash.Get5(KEY_SERIALIZATION, nil).(*ArrayValue)
	if ok {
		serialization := make([]string, se.Len())
		var optFound eval.Attribute
		se.EachWithIndex(func(elem eval.PValue, i int) {
			attrName := elem.String()
			var attr eval.Attribute
			ok := false
			mbr := t.attributes.Get2(attrName, func() interface{} {
				return t.functions.Get2(attrName, func() interface{} {
					return parentMembers.Get(attrName, nil)
				})
			})
			attr, ok = mbr.(eval.Attribute)

			if !ok {
				if mbr == nil {
					panic(eval.Error(c, eval.EVAL_SERIALIZATION_ATTRIBUTE_NOT_FOUND, issue.H{`label`: t.Label(), `attribute`: attrName}))
				}
				panic(eval.Error(c, eval.EVAL_SERIALIZATION_NOT_ATTRIBUTE, issue.H{`label`: t.Label(), `member`: mbr.(eval.AnnotatedMember).Label()}))
			}
			if attr.Kind() == CONSTANT || attr.Kind() == DERIVED {
				panic(eval.Error(c, eval.EVAL_SERIALIZATION_BAD_KIND, issue.H{`label`: t.Label(), `kind`: attr.Kind(), `attribute`: attr.Label()}))
			}
			if attr.HasValue() {
				optFound = attr
			} else if optFound != nil {
				panic(eval.Error(c, eval.EVAL_SERIALIZATION_REQUIRED_AFTER_OPTIONAL, issue.H{`label`: t.Label(), `required`: attr.Label(), `optional`: optFound.Label()}))
			}
			serialization[i] = attrName
		})
		t.serialization = serialization
	}
	t.attrInfo = t.createAttributesInfo()
	t.annotatable.initialize(initHash.(*HashValue))
}

func (t *objectType) HasHashConstructor() bool {
	return t.creators == nil || len(t.creators) == 2
}

func (t *objectType) createNewFunction(c eval.Context) {
	pi := t.AttributesInfo()
	dl := t.loader.(eval.DefiningLoader)

	var ctor eval.Function

	var functions []eval.DispatchFunction
	if t.creators != nil {
		functions = t.creators
		if functions[0] == nil {
			// Specific instruction not to create a constructor
			return
		}
	} else {
		dl.SetEntry(eval.NewTypedName(eval.ALLOCATOR, t.name), eval.NewLoaderEntry(eval.MakeGoAllocator(func(ctx eval.Context, args []eval.PValue) eval.PValue {
			return AllocObjectValue(t)
		}), nil))

		functions = []eval.DispatchFunction{
			// Positional argument creator
			func(c eval.Context, args []eval.PValue) eval.PValue {
				return NewObjectValue(c, t, args)
			},
			// Named argument creator
			func(c eval.Context, args []eval.PValue) eval.PValue {
				return NewObjectValue2(c, t, args[0].(*HashValue))
			}}
	}

	creators := []eval.DispatchCreator{}
	creators = append(creators, func(d eval.Dispatch) {
		for i, attr := range pi.Attributes() {
			switch attr.Kind() {
			case CONSTANT, DERIVED:
			default:
				if i >= pi.RequiredCount() {
					d.OptionalParam2(attr.Type())
				} else {
					d.Param2(attr.Type())
				}
			}
		}
		d.Function(functions[0])
	})

	if len(functions) > 1 {
		creators = append(creators, func(d eval.Dispatch) {
			d.Param2(t.createInitType())
			d.Function(functions[1])
		})
	}

	ctor = eval.MakeGoConstructor(t.name, creators...).Resolve(c)
	dl.SetEntry(eval.NewTypedName(eval.CONSTRUCTOR, t.name), eval.NewLoaderEntry(ctor, nil))
}

func (t *objectType) AttributesInfo() eval.AttributesInfo {
	return t.attrInfo
}

// setCreators takes one or two arguments. The first function is for positional arguments, the second
// for named arguments (expects exactly one argument which is a Hash.
func (t *objectType) setCreators(creators ...eval.DispatchFunction) {
	t.creators = creators
}

func (t *objectType) positionalInitSignature() eval.Signature {
	ai := t.AttributesInfo()
	argTypes := make([]eval.PType, len(ai.Attributes()))
	for i, attr := range ai.Attributes() {
		argTypes[i] = attr.Type()
	}
	return NewCallableType(NewTupleType(argTypes, NewIntegerType(int64(ai.RequiredCount()), int64(len(argTypes)))), t, nil)
}

func (t *objectType) namedInitSignature() eval.Signature {
	return NewCallableType(NewTupleType([]eval.PType{t.createInitType()}, NewIntegerType(1, 1)), t, nil)
}

func (t *objectType) createInitType() *StructType {
	elements := make([]*StructElement, 0)
	t.EachAttribute(true, func(attr eval.Attribute) {
		switch attr.Kind() {
		case CONSTANT, DERIVED:
		default:
			var key eval.PType
			if attr.HasValue() {
				key = NewOptionalType3(attr.Name())
			} else {
				key = NewStringType(nil, attr.Name())
			}
			elements = append(elements, NewStructElement(key, attr.Type()))
		}
	})
	return NewStructType(elements)
}

func (t *objectType) EachAttribute(includeParent bool, consumer func(attr eval.Attribute)) {
	if includeParent && t.parent != nil {
		t.resolvedParent(nil).EachAttribute(includeParent, consumer)
	}
	t.attributes.EachValue(func(a interface{}) { consumer(a.(eval.Attribute)) })
}

func (t *objectType) IsMetaType() bool {
	return eval.IsAssignable(Any_Type, t)
}

func (t *objectType) createAttributesInfo() *attributesInfo {
	attrs := make([]eval.Attribute, 0)
	nonOptSize := 0
	if t.serialization == nil {
		optAttrs := make([]eval.Attribute, 0)
		t.EachAttribute(true, func(attr eval.Attribute) {
			switch attr.Kind() {
			case CONSTANT, DERIVED:
			default:
				if attr.HasValue() {
					optAttrs = append(optAttrs, attr)
				} else {
					attrs = append(attrs, attr)
				}
			}
		})
		nonOptSize = len(attrs)
		attrs = append(attrs, optAttrs...)
	} else {
		atMap := hash.NewStringHash(15)
		t.collectAttributes(true, atMap)
		for _, key := range t.serialization {
			attr := atMap.Get(key, nil).(eval.Attribute)
			if attr.HasValue() {
				nonOptSize++
			}
			attrs = append(attrs, attr)
		}
	}
	return NewParamInfo(attrs, nonOptSize, t.EqualityAttributes().Keys())
}

func NewParamInfo(attributes []eval.Attribute, requiredCount int, equality []string) *attributesInfo {
	nameToPos := make(map[string]int, len(attributes))
	posToName := make(map[int]string, len(attributes))
	for ix, at := range attributes {
		nameToPos[at.Name()] = ix
		posToName[ix] = at.Name()
	}

	ei := make([]int, len(equality))
	for ix, e := range equality {
		ei[ix] = nameToPos[e]
	}

	return &attributesInfo{attributes: attributes, nameToPos: nameToPos, posToName: posToName, equalityAttributeIndexes: ei, requiredCount: requiredCount}
}

func (ai *attributesInfo) NameToPos() map[string]int {
	return ai.nameToPos
}

func (ai *attributesInfo) PosToName() map[int]string {
	return ai.posToName
}

func (pi *attributesInfo) Attributes() []eval.Attribute {
	return pi.attributes
}

func (ai *attributesInfo) EqualityAttributeIndex() []int {
	return ai.equalityAttributeIndexes
}

func (ai *attributesInfo) RequiredCount() int {
	return ai.requiredCount
}

func (ai *attributesInfo) PositionalFromHash(c eval.Context, hash eval.KeyedValue) []eval.PValue {
	nameToPos := ai.NameToPos()
	va := make([]eval.PValue, len(nameToPos))

	hash.EachPair(func(k eval.PValue, v eval.PValue) {
		if ix, ok := nameToPos[k.String()]; ok {
			va[ix] = v
		}
	})
	attrs := ai.Attributes()
	fillValueSlice(c, va, attrs)
	for i := len(va) - 1; i >= ai.RequiredCount(); i-- {
		if !attrs[i].Default(va[i]) {
			break
		}
		va = va[:i]
	}
	return va
}

func (t *objectType) EqualityAttributes() *hash.StringHash {
	eqa := make([]string, 0, 8)
	tp := t
	for tp != nil {
		if tp.equality != nil {
			eqa = append(eqa, tp.equality...)
		}
		tp = tp.resolvedParent(nil)
	}
	attrs := hash.NewStringHash(len(eqa))
	for _, an := range eqa {
		attrs.Put(an, t.GetAttribute(an))
	}
	return attrs
}

func (t *objectType) GetAttribute(name string) eval.Attribute {
	a, _ := t.attributes.Get2(name, func() interface{} {
		p := t.resolvedParent(nil)
		if p != nil {
			return p.GetAttribute(name)
		}
		return nil
	}).(eval.Attribute)
	return a
}

func (t *objectType) GetFunction(name string) eval.Function {
	f, _ := t.functions.Get2(name, func() interface{} {
		p := t.resolvedParent(nil)
		if p != nil {
			return p.GetFunction(name)
		}
		return nil
	}).(eval.Function)
	return f
}

func (t *objectType) Parameters() []eval.PValue {
	return t.Parameters2(true)
}

func (t *objectType) Parameters2(includeName bool) []eval.PValue {
	if t == objectType_DEFAULT {
		return eval.EMPTY_VALUES
	}
	return []eval.PValue{WrapHash3(t.initHash(includeName))}
}

func compressedMembersHash(mh *hash.StringHash) *HashValue {
	he := make([]*HashEntry, 0, mh.Len())
	mh.EachPair(func(key string, value interface{}) {
		fh := value.(eval.AnnotatedMember).InitHash()
		if fh.Len() == 1 {
			tp := fh.Get5(KEY_TYPE, nil)
			if tp != nil {
				he = append(he, WrapHashEntry2(key, tp))
				return
			}
		}
		he = append(he, WrapHashEntry2(key, fh))
	})
	return WrapHash(he)
}

func (t *objectType) checkSelfRecursion(c eval.Context, originator *objectType) {
	if t.parent != nil {
		op := t.resolvedParent(c)
		if eval.Equals(op, originator) {
			panic(eval.Error(c, eval.EVAL_OBJECT_INHERITS_SELF, issue.H{`label`: originator.Label()}))
		}
		op.checkSelfRecursion(c, originator)
	}
}

func (t *objectType) findEqualityDefiner(attrName string) *objectType {
	tp := t
	for tp != nil {
		p := tp.resolvedParent(nil)
		if p == nil || !p.EqualityAttributes().Includes(attrName) {
			return tp
		}
		tp = p
	}
	return nil
}

func (t *objectType) initHash(includeName bool) map[string]eval.PValue {
	h := t.annotatable.initHash()
	if includeName && t.name != `` && t.name != `Object` {
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
		others := hash.NewStringHash(5)
		t.attributes.EachPair(func(key string, value interface{}) {
			a := value.(eval.Attribute)
			if a.Kind() == CONSTANT && eval.Equals(a.Type(), eval.Generalize(a.Value(nil).Type())) {
				constants = append(constants, WrapHashEntry2(key, a.Value(nil)))
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
		ev := make([]eval.PValue, len(t.equality))
		for i, e := range t.equality {
			ev[i] = WrapString(e)
		}
		h[KEY_EQUALITY] = WrapArray(ev)
	}
	if t.serialization != nil {
		sv := make([]eval.PValue, len(t.serialization))
		for i, s := range t.serialization {
			sv[i] = WrapString(s)
		}
		h[KEY_SERIALIZATION] = WrapArray(sv)
	}
	return h
}

func (t *objectType) resolvedParent(c eval.Context) *objectType {
	tp := t.parent
	for {
		switch tp.(type) {
		case nil:
			return nil
		case *objectType:
			return tp.(*objectType)
		case *TypeAliasType:
			tp = tp.(*TypeAliasType).resolvedType
		default:
			panic(eval.Error(nil, eval.EVAL_ILLEGAL_OBJECT_INHERITANCE, issue.H{`label`: t.Label(), `type`: tp.Type().String()}))
		}
	}
}

func (t *objectType) members(includeParent bool) *hash.StringHash {
	collector := hash.NewStringHash(7)
	t.collectMembers(includeParent, collector)
	return collector
}

func (t *objectType) typeParameters(includeParent bool) *hash.StringHash {
	collector := hash.NewStringHash(5)
	t.collectParameters(includeParent, collector)
	return collector
}

func (t *objectType) collectAttributes(includeParent bool, collector *hash.StringHash) {
	if includeParent && t.parent != nil {
		t.resolvedParent(nil).collectAttributes(true, collector)
	}
	collector.PutAll(t.attributes)
}

func (t *objectType) collectFunctions(includeParent bool, collector *hash.StringHash) {
	if includeParent && t.parent != nil {
		t.resolvedParent(nil).collectFunctions(true, collector)
	}
	collector.PutAll(t.functions)
}

func (t *objectType) collectMembers(includeParent bool, collector *hash.StringHash) {
	if includeParent && t.parent != nil {
		t.resolvedParent(nil).collectMembers(true, collector)
	}
	collector.PutAll(t.attributes)
	collector.PutAll(t.functions)
}

func (t *objectType) collectParameters(includeParent bool, collector *hash.StringHash) {
	if includeParent && t.parent != nil {
		t.resolvedParent(nil).collectParameters(true, collector)
	}
	collector.PutAll(t.parameters)
}

func AllocObjectValue(typ eval.ObjectType) eval.ObjectValue {
	if typ.IsMetaType() {
		return AllocObjectType()
	}
	return &objectValue{typ, eval.EMPTY_VALUES}
}

func (ov *objectValue) Initialize(c eval.Context, values []eval.PValue) {
	if len(values) > 0 && ov.typ.IsParameterized() {
		ov.InitFromHash(c, makeValueHash(ov.typ.AttributesInfo(), values))
		return
	}
	fillValueSlice(c, values, ov.typ.AttributesInfo().Attributes())
	ov.values = values
}

func (ov *objectValue) InitFromHash(c eval.Context, hash eval.KeyedValue) {
	typ := ov.typ.(*objectType)
	va := typ.AttributesInfo().PositionalFromHash(c, hash)
	if len(va) > 0 && typ.IsParameterized() {
		params := make([]*HashEntry, 0)
		typ.typeParameters(true).EachPair(func(k string, v interface{}) {
			if pv, ok := hash.Get4(k); ok && eval.IsInstance(c, v.(*typeParameter).typ, pv) {
				params = append(params, WrapHashEntry2(k, pv))
			}
		})
		if len(params) > 0 {
			ov.typ = NewObjectTypeExtension(c, typ, []eval.PValue{WrapHash(params)})
		}
	}
	ov.values = va
}

func NewObjectValue(c eval.Context, typ eval.ObjectType, values []eval.PValue) eval.ObjectValue {
	ov := AllocObjectValue(typ)
	ov.Initialize(c, values)
	return ov
}

func NewObjectValue2(c eval.Context, typ eval.ObjectType, hash *HashValue) eval.ObjectValue {
	ov := AllocObjectValue(typ)
	ov.InitFromHash(c, hash)
	return ov
}

// Ensure that all entries in the value slice that are nil receive default values from the given attributes
func fillValueSlice(c eval.Context, values []eval.PValue, attrs []eval.Attribute) {
	for ix, v := range values {
		if v == nil {
			at := attrs[ix]
			if !at.HasValue() {
				panic(eval.Error(c, eval.EVAL_MISSING_REQUIRED_ATTRIBUTE, issue.H{`label`: at.Label()}))
			}
			values[ix] = at.Value(c)
		}
	}
}

func (o *objectValue) Get(c eval.Context, key string) (eval.PValue, bool) {
	pi := o.typ.AttributesInfo()
	if idx, ok := pi.NameToPos()[key]; ok {
		if idx < len(o.values) {
			return o.values[idx], ok
		}
		return pi.Attributes()[idx].Value(c), ok
	}
	return nil, false
}

func (o *objectValue) Equals(other interface{}, g eval.Guard) bool {
	if ov, ok := other.(*objectValue); ok {
		return o.typ.Equals(ov.typ, g) && eval.GuardedEquals(o.values, ov.values, g)
	}
	return false
}

func (o *objectValue) String() string {
	return eval.ToString(o)
}

func (o *objectValue) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	ObjectToString(o, s, b, g)
}

func (o *objectValue) Type() eval.PType {
	return o.typ
}

func (o *objectValue) InitHash() eval.KeyedValue {
	return makeValueHash(o.typ.AttributesInfo(), o.values)
}

// Turn a positional argument list into a hash. The hash will exclude all values
// that are equal to the default value of the corresponding attribute
func makeValueHash(pi eval.AttributesInfo, values []eval.PValue) *HashValue {
	posToName := pi.PosToName()
	entries := make([]*HashEntry, 0, len(posToName))
	for i, v := range values {
		attr := pi.Attributes()[i]
		if !(attr.HasValue() && eval.Equals(v, attr.Value(nil))) {
			entries = append(entries, WrapHashEntry2(attr.Name(), v))
		}
	}
	return WrapHash(entries)
}
