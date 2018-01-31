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
)

var Object_Type PType

func init() {
	Object_Type = newType(`ObjectType`,
		`AnyType {
  attributes => {
    '_pcore_init_hash' => Struct[
			Optional[name] => Pattern[/\A[A-Z][\w]*(?:::[A-Z][\w]*)*\z/],
			Optional[parent] => Type,
			Optional[type_parameters] => Hash[Pattern[/\A[a-z_]\w*\z/], NotUndef],
			Optional[attributes] => Hash[Pattern[/\A[a-z_]\w*\z/], NotUndef],
			Optional[constants] => Hash[Pattern[/\A[a-z_]\w*\z/], Any],
			Optional[functions] => Hash[Pattern[/\A[a-z_]\w*\z/], Callable],
			Optional[equality] => Variant[Pattern[/\A[a-z_]\w*\z/], Array[Pattern[/\A[a-z_]\w*\z/]]],
			Optional[equality_includes_type] => Boolean,
      Optional[checks] => Any,
      Optional[annotations] => Hash[Type[Annotation], Hash[Pattern[/\A[a-z_]\w*\z/], Any]]
    ]
  }
}`)
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

	DEFAULT_KIND     = AttributeKind(``)
	CONSTANT         = AttributeKind(`constant`)
	DERIVED          = AttributeKind(`derived`)
	GIVEN_OR_DERIVED = AttributeKind(`given_or_derived`)
	REFERENCE        = AttributeKind(`reference`)
)

var QREF_PATTERN = regexp.MustCompile(`\A[A-Z][\w]*(?:::[A-Z][\w]*)*\z`)

var annotationType_DEFAULT = &objectType{
	annotatable: annotatable{annotations: _EMPTY_MAP},
	hashKey:     HashKey("\x00tAnnotation"),
	name:        `Annotation`,
	parameters:  EMPTY_STRINGHASH,
	attributes:  EMPTY_STRINGHASH,
	functions:   EMPTY_STRINGHASH,
	equality:    nil}

func DefaultAnnotationType() PType {
	return annotationType_DEFAULT
}

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

var TYPE_MEMBER_NAMES = NewArrayType2(TYPE_MEMBER_NAME)
var TYPE_PARAMETERS = NewHashType(TYPE_MEMBER_NAME, DefaultNotUndefType(), nil)
var TYPE_ATTRIBUTES = NewHashType(TYPE_MEMBER_NAME, DefaultNotUndefType(), nil)
var TYPE_CONSTANTS = NewHashType(TYPE_MEMBER_NAME, DefaultAnyType(), nil)
var TYPE_FUNCTIONS = NewHashType(TYPE_MEMBER_NAME, DefaultNotUndefType(), nil)
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
	NewStructElement(NewOptionalType3(KEY_ANNOTATIONS), annotationType_DEFAULT),
})

type (
	annotatable struct {
		annotations *HashValue
	}

	annotatedMember struct {
		annotatable
		name      string
		container *objectType
		typ       PType
		override  bool
		final     bool
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

	attributesInfo struct {
		nameToPos                map[string]int
		posToName                map[int]string
		attributes               []Attribute
		equalityAttributeIndexes []int
		requiredCount            int
	}

	objectType struct {
		annotatable
		hashKey             HashKey
		name                string
		parent              PType
		parameters          *StringHash // map doesn't preserve order
		attributes          *StringHash
		functions           *StringHash
		equality            []string
		equalityIncludeType bool
		serialization       []string
		loader              Loader
		initHashExpression  interface{} // Expression or *HashValue
		attrInfo            *attributesInfo
	}

	objectValue struct {
		typ    ObjectType
		values []PValue
	}
)

func argError(e PType, a PValue) errors.InstantiationError {
	return errors.NewArgumentsError(``, DescribeMismatch(`assert`, e, a.Type()))
}

func typeArg(hash KeyedValue, key string, d PType) PType {
	v := hash.Get5(key, nil)
	if v == nil {
		return d
	}
	if t, ok := v.(PType); ok {
		return t
	}
	panic(argError(DefaultTypeType(), v))
}

func hashArg(hash KeyedValue, key string) *HashValue {
	v := hash.Get5(key, nil)
	if v == nil {
		return _EMPTY_MAP
	}
	if t, ok := v.(*HashValue); ok {
		return t
	}
	panic(argError(DefaultHashType(), v))
}

func boolArg(hash KeyedValue, key string, d bool) bool {
	v := hash.Get5(key, nil)
	if v == nil {
		return d
	}
	if t, ok := v.(*BooleanValue); ok {
		return t.Bool()
	}
	panic(argError(DefaultBooleanType(), v))
}

func stringArg(hash KeyedValue, key string, d string) string {
	v := hash.Get5(key, nil)
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

func (a *annotatedMember) initialize(name string, container *objectType, initHash *HashValue) {
	a.annotatable.initialize(initHash)
	a.name = name
	a.container = container
	a.typ = typeArg(initHash, KEY_TYPE, DefaultTypeType())
	a.override = boolArg(initHash, KEY_OVERRIDE, false)
	a.final = boolArg(initHash, KEY_FINAL, false)
}

func (a *annotatedMember) Accept(v Visitor, g Guard) {
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

func (a *annotatedMember) Call(c EvalContext, receiver PValue, block Lambda, args []PValue) PValue {
	// TODO:
	panic("implement me")
}

func (a *annotatedMember) Name() string {
	return a.name
}

func (a *annotatedMember) Container() ObjectType {
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

func newAttribute(name string, container *objectType, initHash *HashValue) Attribute {
	a := &attribute{}
	a.initialize(name, container, initHash)
	return a
}

func (a *attribute) initialize(name string, container *objectType, initHash *HashValue) {
	a.annotatedMember.initialize(name, container, initHash)
	AssertInstance(func() string { return fmt.Sprintf(`initializer for %s`, a.Label()) }, TYPE_ATTRIBUTE, initHash)
	a.kind = AttributeKind(stringArg(initHash, KEY_KIND, ``))
	if a.kind == CONSTANT { // final is implied
		if initHash.IncludesKey2(KEY_FINAL) && !a.final {
			panic(Error(EVAL_CONSTANT_WITH_FINAL, H{`label`: a.Label()}))
		}
		a.final = true
	}
	v := initHash.Get5(KEY_VALUE, nil)
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

func (a *attribute) Call(c EvalContext, receiver PValue, block Lambda, args []PValue) PValue {
	if block == nil && len(args) == 0 {
		if v, ok := a.container.GetValue(a.name, receiver); ok {
			return v
		}
	}
	panic(Error(EVAL_TYPE_MISMATCH, H{`detail`: DescribeSignatures(
		[]Signature{a.CallableType().(*CallableType)}, NewTupleType2(args...), block)}))
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

func (a *attribute) InitHash() KeyedValue {
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

func (a *attribute) CallableType() PType {
	return TYPE_ATTRIBUTE_CALLABLE
}

func newTypeParameter(name string, container *objectType, initHash *HashValue) Attribute {
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

func (t *typeParameter) InitHash() KeyedValue {
	return WrapHash3(t.initHash())
}

func (t *typeParameter) FeatureType() string {
	return `type_parameter`
}

func newFunction(name string, container *objectType, initHash *HashValue) ObjFunc {
	f := &function{}
	f.initialize(name, container, initHash)
	return f
}

func (f *function) initialize(name string, container *objectType, initHash *HashValue) {
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

func (f *function) CallableType() PType {
	return f.typ.(*CallableType)
}

func (f *function) InitHash() KeyedValue {
	return WrapHash3(f.initHash())
}

var objectType_DEFAULT = &objectType{
	annotatable: annotatable{annotations: _EMPTY_MAP},
	name:        `Object`,
	hashKey:     HashKey("\x00tObject"),
	parameters:  EMPTY_STRINGHASH,
	attributes:  EMPTY_STRINGHASH,
	functions:   EMPTY_STRINGHASH}

func DefaultObjectType() *objectType {
	return objectType_DEFAULT
}

var objectId = int64(0)

func AllocObjectType() *objectType {
	return &objectType{
		annotatable:        annotatable{annotations: _EMPTY_MAP},
		name:               ``,
		hashKey:            HashKey(fmt.Sprintf("\x00tObject%d", atomic.AddInt64(&objectId, 1))),
		initHashExpression: nil,
		parent:             nil,
		parameters:         EMPTY_STRINGHASH,
		attributes:         EMPTY_STRINGHASH,
		functions:          EMPTY_STRINGHASH}
}

func (f *objectType) Initialize(args []PValue) {
	if len(args) == 1 {
		if hash, ok := args[0].(KeyedValue); ok {
			f.InitFromHash(hash)
			return
		}
	}
	panic(Error(EVAL_FAILURE, H{`message`: `internal error when creating an Object data type`}))
}

func NewObjectType(name string, parent PType, initHashExpression Expression) *objectType {
	obj := AllocObjectType()
	obj.name = name
	obj.initHashExpression = initHashExpression
	obj.parent = parent
	return obj
}

func NewObjectType2(initHash *HashValue, loader Loader) *objectType {
	obj := AllocObjectType()
	obj.InitFromHash(initHash)
	obj.loader = loader
	return obj
}

func (t *objectType) Accept(v Visitor, g Guard) {
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
	t.parameters.EachValue(func(p interface{}) { p.(AnnotatedMember).Accept(v, g) })
	t.attributes.EachValue(func(a interface{}) { a.(AnnotatedMember).Accept(v, g) })
	t.functions.EachValue(func(f interface{}) { f.(AnnotatedMember).Accept(v, g) })
}

func (t *objectType) Default() PType {
	return objectType_DEFAULT
}

func (t *objectType) Equals(other interface{}, guard Guard) bool {
	ot, ok := other.(*objectType)
	if !ok {
		return false
	}

	if t.name == `` || ot.name == `` {
		return t == ot
	}

	return t.name == ot.name && t.loader.NameAuthority() == ot.loader.NameAuthority()
}

func (t *objectType) GetValue(key string, o PValue) (value PValue, ok bool) {
	if pu, ok := o.(PuppetObject); ok {
		return pu.Get(key)
	}

	// TODO: Perhaps use other ways of extracting attributes with reflection
	// in case native types must be described by Object
	return nil, false
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

func (t *objectType) Member(name string) (CallableMember, bool) {
	mbr := t.attributes.Get(name, func(k string) interface{} {
		return t.functions.Get(name, func(k string) interface{} {
			if t.parent == nil {
				return nil
			}
			pm, _ := t.resolvedParent().Member(name)
			return pm
		})
	})
	if mbr == nil {
		return nil, false
	}
	return mbr.(CallableMember), true
}

func (t *objectType) ToKey() HashKey {
	return t.hashKey
}

func (t *objectType) ToString(b io.Writer, s FormatContext, g RDetect) {
	TypeToString(t, b, s, g)
}

func (t *objectType) Type() PType {
	return Object_Type
}

func (t *objectType) String() string {
	return ToString2(t, EXPANDED)
}

func (t *objectType) Get(key string) (value PValue, ok bool) {
	if key == `_pcore_init_hash` {
		return t.InitHash(), true
	}
	return nil, false
}

func (t *objectType) InitHash() KeyedValue {
	return WrapHash3(t.initHash(true))
}

func (t *objectType) IsInstance(o PValue, g Guard) bool {
	return isAssignable(t, o.Type())
}

func (t *objectType) IsAssignable(o PType, g Guard) bool {
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
	return !t.parameters.IsEmpty()
}

func (t *objectType) Resolve(c EvalContext) PType {
	if t.initHashExpression != nil {
		ihe := t.initHashExpression
		t.initHashExpression = nil

		var initHash *HashValue
		if lh, ok := ihe.(*LiteralHash); ok {
			initHash = c.Resolve(lh).(*HashValue)
			if prt, ok := t.parent.(ResolvableType); ok {
				t.parent = resolveTypeRefs(c, prt).(PType)
			}
		} else {
			initHash = resolveTypeRefs(c, ihe.(*HashValue)).(*HashValue)
		}
		t.loader = c.Loader()
		t.InitFromHash(initHash)
		if t.name != `` {
			t.createNewFunction(c)
		}
	}
	return t
}

func resolveTypeRefs(c EvalContext, v PValue) PValue {
	switch v.(type) {
	case *HashValue:
		hv := v.(*HashValue)
		he := make([]*HashEntry, hv.Len())
		i := 0
		hv.EachPair(func(key, value PValue) {
			he[i] = WrapHashEntry(
				resolveTypeRefs(c, key), resolveTypeRefs(c, value))
			i++
		})
		return WrapHash(he)
	case *ArrayValue:
		av := v.(*ArrayValue)
		ae := make([]PValue, av.Len())
		i := 0
		av.Each(func(value PValue) {
			ae[i] = resolveTypeRefs(c, value)
			i++
		})
		return WrapArray(ae)
	case ResolvableType:
		return v.(ResolvableType).Resolve(c)
	default:
		return v
	}
}

func (t *objectType) InitFromHash(initHash KeyedValue) {
	AssertInstance(`object initializer`, TYPE_OBJECT_INIT_HASH, initHash)
	t.parameters = EMPTY_STRINGHASH
	t.attributes = EMPTY_STRINGHASH
	t.functions = EMPTY_STRINGHASH
	t.name = stringArg(initHash, KEY_NAME, t.name)

	t.parent = typeArg(initHash, KEY_PARENT, nil)

	parentMembers := EMPTY_STRINGHASH
	parentTypeParams := EMPTY_STRINGHASH
	var parentObjectType *objectType

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
				paramValue = ph.Get5(KEY_VALUE, nil)
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
			parameters.Put(key, param)
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
			if attributes.IncludesKey(key) {
				panic(Error(EVAL_MEMBER_NAME_CONFLICT, H{`label`: fmt.Sprintf(`function %s[%s]`, t.Label(), key)}))
			}
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
			var attr Attribute
			ok := false
			mbr := t.attributes.Get2(attrName, func() interface{} {
				return t.functions.Get2(attrName, func() interface{} {
					return parentMembers.Get(attrName, nil)
				})
			})
			attr, ok = mbr.(Attribute)

			if !ok {
				if mbr == nil {
					panic(Error(EVAL_EQUALITY_ATTRIBUTE_NOT_FOUND, H{`label`: t.Label(), `attribute`: attrName}))
				}
				panic(Error(EVAL_EQUALITY_NOT_ATTRIBUTE, H{`label`: t.Label(), `member`: mbr.(AnnotatedMember).Label()}))
			}
			if attr.Kind() == CONSTANT {
				panic(Error(EVAL_EQUALITY_ON_CONSTANT, H{`label`: t.Label(), `attribute`: mbr.(AnnotatedMember).Label()}))
			}
			// Assert that attribute is not already include by parent equality
			if ok && parentObjectType.EqualityAttributes().Includes(attrName) {
				includingParent := t.findEqualityDefiner(attrName)
				panic(Error(EVAL_EQUALITY_REDEFINED, H{`label`: t.Label(), `attribute`: attr.Label(), `including_parent`: includingParent}))
			}
		}
	}
	t.equality = equality

	se, ok := initHash.Get5(KEY_SERIALIZATION, nil).(*ArrayValue)
	if ok {
		serialization := make([]string, se.Len())
		var optFound Attribute
		for i, elem := range se.elements {
			attrName := elem.String()
			var attr Attribute
			ok := false
			mbr := t.attributes.Get2(attrName, func() interface{} {
				return t.functions.Get2(attrName, func() interface{} {
					return parentMembers.Get(attrName, nil)
				})
			})
			attr, ok = mbr.(Attribute)

			if !ok {
				if mbr == nil {
					panic(Error(EVAL_SERIALIZATION_ATTRIBUTE_NOT_FOUND, H{`label`: t.Label(), `attribute`: attrName}))
				}
				panic(Error(EVAL_SERIALIZATION_NOT_ATTRIBUTE, H{`label`: t.Label(), `member`: mbr.(AnnotatedMember).Label()}))
			}
			if attr.Kind() == CONSTANT || attr.Kind() == DERIVED {
				panic(Error(EVAL_SERIALIZATION_BAD_KIND, H{`label`: t.Label(), `kind`: attr.Kind(), `attribute`: attr.Label()}))
			}
			if attr.HasValue() {
				optFound = attr
			} else if optFound != nil {
				panic(Error(EVAL_SERIALIZATION_REQUIRED_AFTER_OPTIONAL, H{`label`: t.Label(), `required`: attr.Label(), `optional`: optFound.Label()}))
			}
			serialization[i] = attrName
		}
		t.serialization = serialization
	}
	t.attrInfo = t.createAttributesInfo()
}

func (o *objectType) createNewFunction(c EvalContext) {
	pi := o.AttributesInfo()
	dl := o.loader.(DefiningLoader)

	dl.SetEntry(NewTypedName(ALLOCATOR, o.name), NewLoaderEntry(MakeGoAllocator(func(ctx EvalContext, args []PValue) PValue {
		return AllocObjectValue(o)
	}), ``))

	ctor := MakeGoConstructor(o.name,
		func(d Dispatch) {
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
			d.Function(func(c EvalContext, args []PValue) PValue {
				return NewObjectValue(o, args)
			})
		},
		func(d Dispatch) {
			d.Param2(o.createInitType())
			d.Function(func(c EvalContext, args []PValue) PValue {
				return NewObjectValue2(o, args[0].(*HashValue))
			})
		})
	dl.SetEntry(NewTypedName(CONSTRUCTOR, o.name), NewLoaderEntry(ctor.Resolve(c), ``))
}

func (o *objectType) AttributesInfo() AttributesInfo {
	return o.attrInfo
}

func (o *objectType) createInitType() *StructType {
	elements := make([]*StructElement, 0)
	o.EachAttribute(true, func(attr Attribute) {
		switch attr.Kind() {
		case CONSTANT, DERIVED:
		default:
			var key PType
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

func (o *objectType) EachAttribute(includeParent bool, consumer func(attr Attribute)) {
	if includeParent && o.parent != nil {
		o.resolvedParent().EachAttribute(includeParent, consumer)
	}
	o.attributes.EachValue(func(a interface{}) { consumer(a.(Attribute)) })
}

func (o *objectType) IsMetaType() bool {
	return IsAssignable(Any_Type, o)
}

func (o *objectType) createAttributesInfo() *attributesInfo {
	attrs := make([]Attribute, 0)
	nonOptSize := 0
	if o.serialization == nil {
		optAttrs := make([]Attribute, 0)
		o.EachAttribute(true, func(attr Attribute) {
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
		atMap := NewStringHash(15)
		o.collectAttributes(true, atMap)
		for _, key := range o.serialization {
			attr := atMap.Get(key, nil).(Attribute)
			if attr.HasValue() {
				nonOptSize++
			}
			attrs = append(attrs, attr)
		}
	}
	return NewParamInfo(attrs, nonOptSize, o.EqualityAttributes().Keys())
}

func NewParamInfo(attributes []Attribute, requiredCount int, equality []string) *attributesInfo {
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

func (pi *attributesInfo) NameToPos() map[string]int {
	return pi.nameToPos
}

func (pi *attributesInfo) PosToName() map[int]string {
	return pi.posToName
}

func (pi *attributesInfo) Attributes() []Attribute {
	return pi.attributes
}

func (pi *attributesInfo) EqualityAttributeIndex() []int {
	return pi.equalityAttributeIndexes
}

func (pi *attributesInfo) RequiredCount() int {
	return pi.requiredCount
}

func (t *objectType) EqualityAttributes() *StringHash {
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

func (t *objectType) GetAttribute(name string) Attribute {
	a, _ := t.attributes.Get2(name, func() interface{} {
		p := t.resolvedParent()
		if p != nil {
			return p.GetAttribute(name)
		}
		return nil
	}).(Attribute)
	return a
}

func (t *objectType) GetFunction(name string) Function {
	f, _ := t.functions.Get2(name, func() interface{} {
		p := t.resolvedParent()
		if p != nil {
			return p.GetFunction(name)
		}
		return nil
	}).(Function)
	return f
}

func (t *objectType) Parameters() []PValue {
	return t.Parameters2(true)
}

func (t *objectType) Parameters2(includeName bool) []PValue {
	return []PValue{WrapHash3(t.initHash(includeName))}
}

func compressedMembersHash(mh *StringHash) *HashValue {
	he := make([]*HashEntry, 0, mh.Size())
	mh.EachPair(func(key string, value interface{}) {
		fh := value.(AnnotatedMember).InitHash()
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

func (t *objectType) checkSelfRecursion(originator *objectType) {
	if t.parent != nil {
		op := t.resolvedParent()
		if Equals(op, originator) {
			panic(Error(EVAL_OBJECT_INHERITS_SELF, H{`label`: originator.Label()}))
		}
		op.checkSelfRecursion(originator)
	}
}

func (t *objectType) findEqualityDefiner(attrName string) *objectType {
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

func (t *objectType) initHash(includeName bool) map[string]PValue {
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
	if t.serialization != nil {
		sv := make([]PValue, len(t.serialization))
		for i, s := range t.serialization {
			sv[i] = WrapString(s)
		}
		h[KEY_SERIALIZATION] = WrapArray(sv)
	}
	return h
}

func (t *objectType) resolvedParent() *objectType {
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
			panic(Error(EVAL_ILLEGAL_OBJECT_INHERITANCE, H{`label`: t.Label(), `type`: tp.Type().String()}))
		}
	}
}

func (t *objectType) members(includeParent bool) *StringHash {
	collector := NewStringHash(7)
	t.collectMembers(includeParent, collector)
	return collector
}

func (t *objectType) typeParameters(includeParent bool) *StringHash {
	collector := NewStringHash(5)
	t.collectParameters(includeParent, collector)
	return collector
}

func (t *objectType) collectAttributes(includeParent bool, collector *StringHash) {
	if includeParent && t.parent != nil {
		t.resolvedParent().collectAttributes(true, collector)
	}
	collector.PutAll(t.attributes)
}

func (t *objectType) collectFunctions(includeParent bool, collector *StringHash) {
	if includeParent && t.parent != nil {
		t.resolvedParent().collectFunctions(true, collector)
	}
	collector.PutAll(t.functions)
}

func (t *objectType) collectMembers(includeParent bool, collector *StringHash) {
	if includeParent && t.parent != nil {
		t.resolvedParent().collectMembers(true, collector)
	}
	collector.PutAll(t.attributes)
	collector.PutAll(t.functions)
}

func (t *objectType) collectParameters(includeParent bool, collector *StringHash) {
	if includeParent && t.parent != nil {
		t.resolvedParent().collectParameters(true, collector)
	}
	collector.PutAll(t.parameters)
}

func AllocObjectValue(typ *objectType) ObjectValue {
	if typ.IsMetaType() {
		return AllocObjectType()
	}
	return &objectValue{typ, EMPTY_VALUES}
}

func (ov *objectValue) Initialize(values []PValue) {
	if len(values) > 0 && ov.typ.IsParameterized() {
		ov.InitFromHash(makeValueHash(ov.typ.AttributesInfo(), values))
		return
	}
	fillValueSlice(values, ov.typ.AttributesInfo().Attributes())
	ov.values = values
}

func (ov *objectValue) InitFromHash(hash KeyedValue) {
	typ := ov.typ.(*objectType)
	ai := typ.AttributesInfo()

	nameToPos := ai.NameToPos()
	va := make([]PValue, len(nameToPos))

	hash.EachPair(func(k PValue, v PValue) {
		if ix, ok := nameToPos[k.String()]; ok {
			va[ix] = v
		}
	})
	fillValueSlice(va, ai.Attributes())
	if len(va) > 0 && typ.IsParameterized() {
		params := make([]*HashEntry, 0)
		typ.typeParameters(true).EachPair(func(k string, v interface{}) {
			if pv, ok := hash.Get4(k); ok && IsInstance(v.(*typeParameter).typ, pv) {
				params = append(params, WrapHashEntry2(k, pv))
			}
		})
		if len(params) > 0 {
			ov.typ = NewObjectTypeExtension(typ, []PValue{WrapHash(params)})
		}
	}
	ov.values = va
}

func NewObjectValue(typ *objectType, values []PValue) ObjectValue {
	ov := AllocObjectValue(typ)
	ov.Initialize(values)
	return ov
}

func NewObjectValue2(typ *objectType, hash *HashValue) ObjectValue {
	ov := AllocObjectValue(typ)
	ov.InitFromHash(hash)
	return ov
}

// Ensure that all entries in the value slice that are nil receive default values from the given attributes
func fillValueSlice(values []PValue, attrs []Attribute) {
	for ix, v := range values {
		if v == nil {
			at := attrs[ix]
			if !at.HasValue() {
				panic(Error(EVAL_MISSING_REQUIRED_ATTRIBUTE, H{`label`: at.Label()}))
			}
			values[ix] = at.Value()
		}
	}
}

func (o *objectValue) Get(key string) (PValue, bool) {
	pi := o.typ.AttributesInfo()
	if idx, ok := pi.NameToPos()[key]; ok {
		if idx < len(o.values) {
			return o.values[idx], ok
		}
		return pi.Attributes()[idx].Value(), ok
	}
	return nil, false
}

func (o *objectValue) Equals(other interface{}, g Guard) bool {
	if ov, ok := other.(*objectValue); ok {
		return o.typ.Equals(ov.typ, g) && GuardedEquals(o.values, ov.values, g)
	}
	return false
}

func (o *objectValue) String() string {
	return ToString(o)
}

func (o *objectValue) ToString(b io.Writer, s FormatContext, g RDetect) {
	io.WriteString(b, o.typ.Name())
	o.InitHash().(*HashValue).ToString2(b, s, GetFormat(s.FormatMap(), o.typ), '(', g)
}

func (o *objectValue) Type() PType {
	return o.typ
}

func (o *objectValue) InitHash() KeyedValue {
	return makeValueHash(o.typ.AttributesInfo(), o.values)
}

// Turn a positional argument list into a hash. The hash will exclude all values
// that are equal to the default value of the corresponding attribute
func makeValueHash(pi AttributesInfo, values []PValue) *HashValue {
	posToName := pi.PosToName()
	entries := make([]*HashEntry, 0, len(posToName))
	for i, v := range values {
		attr := pi.Attributes()[i]
		if !(attr.HasValue() && Equals(v, attr.Value())) {
			entries = append(entries, WrapHashEntry2(attr.Name(), v))
		}
	}
	return WrapHash(entries)
}
