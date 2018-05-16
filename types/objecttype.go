package types

import (
	"fmt"
	"io"
	"reflect"
	"regexp"
	"runtime"
	"sync/atomic"

	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/hash"
	"github.com/puppetlabs/go-issues/issue"
	"github.com/puppetlabs/go-parser/parser"
	"github.com/puppetlabs/go-parser/validator"
	"bytes"
	"github.com/puppetlabs/go-evaluator/utils"
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

var TYPE_OBJECT_NAME = NewPatternType([]*RegexpType{NewRegexpTypeR(QREF_PATTERN)})

var TYPE_MEMBER_NAME = NewPatternType2(NewRegexpTypeR(validator.PARAM_NAME))

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

func (t *objectType) AttributesInfo() eval.AttributesInfo {
	return t.attrInfo
}

func (t *objectType) Default() eval.PType {
	return objectType_DEFAULT
}

func (t *objectType) EachAttribute(includeParent bool, consumer func(attr eval.Attribute)) {
	if includeParent && t.parent != nil {
		t.resolvedParent(nil).EachAttribute(includeParent, consumer)
	}
	t.attributes.EachValue(func(a interface{}) { consumer(a.(eval.Attribute)) })
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

func (t *objectType) FromReflectedValue(c eval.Context, src reflect.Value) eval.PuppetObject {
	if src.Kind() == reflect.Ptr {
		src = src.Elem()
	}
	entries := t.appendAttributeValues(c, make([]*HashEntry, 0), src)
	return eval.New(c, t, WrapHash(entries)).(eval.PuppetObject)
}

func (t *objectType) Get(c eval.Context, key string) (value eval.PValue, ok bool) {
	if key == `_pcore_init_hash` {
		return t.InitHash(), true
	}
	return nil, false
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

func (t *objectType) GetValue(c eval.Context, key string, o eval.PValue) (value eval.PValue, ok bool) {
	if pu, ok := o.(eval.ReadableObject); ok {
		return pu.Get(c, key)
	}

	// TODO: Perhaps use other ways of extracting attributes with reflection
	// in case native types must be described by Object
	return nil, false
}

func (t *objectType) HasHashConstructor() bool {
	return t.creators == nil || len(t.creators) == 2
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
			param := newTypeParameter(c, key, t, WrapHash4(c, issue.H{
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
			attrSpecs.Put(key, WrapHash4(c, attrSpec))
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
				attrSpec = WrapHash4(c, hash)
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
				funcSpec = WrapHash4(c, issue.H{KEY_TYPE: funcType})
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

func (t *objectType) InitHash() eval.KeyedValue {
	return WrapStringPValue(t.initHash(true))
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

func (t *objectType) IsInstance(c eval.Context, o eval.PValue, g eval.Guard) bool {
	return isAssignable(t, o.Type())
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

func (t *objectType) IsMetaType() bool {
	return eval.IsAssignable(Any_Type, t)
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

func (t *objectType) MetaType() eval.ObjectType {
	return Object_Type
}

func (t *objectType) Name() string {
	return t.name
}

func (t *objectType) Parameters() []eval.PValue {
	return t.Parameters2(true)
}

func (t *objectType) Parameters2(includeName bool) []eval.PValue {
	if t == objectType_DEFAULT {
		return eval.EMPTY_VALUES
	}
	return []eval.PValue{WrapStringPValue(t.initHash(includeName))}
}

func (t *objectType) Resolve(c eval.Context) eval.PType {
	if t.initHashExpression != nil {
		ihe := t.initHashExpression
		t.initHashExpression = nil

		if prt, ok := t.parent.(eval.ResolvableType); ok {
			t.parent = resolveTypeRefs(c, prt).(eval.PType)
		}

		var initHash *HashValue
		if lh, ok := ihe.(*parser.LiteralHash); ok {
			initHash = c.Evaluate(lh).(*HashValue)
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

func (t *objectType) String() string {
	return eval.ToString2(t, EXPANDED)
}

func (t *objectType) ToKey() eval.HashKey {
	return t.hashKey
}

func (t *objectType) ToReflectedValue(c eval.Context, src eval.PuppetObject, dest reflect.Value) {
	eval.AssertKind(c, dest, reflect.Struct)
	dt := dest.Type()
	n := dest.NumField()
	for i := 0; i < n; i++ {
		field := dt.Field(i)
		if field.Anonymous && i == 0 && t.parent != nil {
			t.resolvedParent(c).ToReflectedValue(c, src, dest.Field(i))
			continue
		}
		an := FieldName(c, &field)
		if av, ok := src.Get(c, an); ok {
			eval.ReflectTo(c, av, dest.Field(i))
			continue
		}
		panic(eval.Error(c, eval.EVAL_ATTRIBUTE_NOT_FOUND, issue.H{`name`: an}))
	}
}

func (t *objectType) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	f := eval.GetFormat(s.FormatMap(), t.Type())
	switch f.FormatChar() {
	case 's', 'p':
		quoted := f.IsAlt() && f.FormatChar() == 's'
		if quoted || f.HasStringFlags() {
			bld := bytes.NewBufferString(``)
			t.basicTypeToString(bld, f, s, g)
			f.ApplyStringFlags(b, bld.String(), quoted)
		} else {t.basicTypeToString( b, f,s, g)}
	default:
		panic(s.UnsupportedFormat(t.Type(), `sp`, f))
	}
}

func (t *objectType) basicTypeToString(b io.Writer, f eval.Format, s eval.FormatContext, g eval.RDetect) {

	if t.Equals(DefaultObjectType(), nil) {
		io.WriteString(b, `Object`)
		return
	}

	typeSetName, inTypeSet := s.Property(`typeSet`)
	if ex, ok := s.Property(`expanded`); !(ok && ex == `true`) {
		name := t.Name()
		if inTypeSet {
			name = stripTypeSetName(typeSetName, name)
		}
		io.WriteString(b, name)
		return
	}

	// Avoid nested expansions
	s = s.WithProperties(map[string]string{`expanded`: `false`})

	indent1 := s.Indentation()
	indent2 := indent1.Increase(f.IsAlt())
	indent3 := indent2.Increase(f.IsAlt())
	padding1 := ``
	padding2 := ``
	padding3 := ``
	if f.IsAlt() {
		padding1 = indent1.Padding()
		padding2 = indent2.Padding()
		padding3 = indent3.Padding()
	}

	cf := f.ContainerFormats()
	if cf == nil {
		cf = DEFAULT_CONTAINER_FORMATS
	}

	ctx2 := eval.NewFormatContext2(indent2, s.FormatMap(), s.Properties())
	cti2 := eval.NewFormatContext2(indent2, cf, s.Properties())
	ctx3 := eval.NewFormatContext2(indent3, s.FormatMap(), s.Properties())

	if inTypeSet {
		if t.parent != nil {
			io.WriteString(b, stripTypeSetName(typeSetName, t.parent.Name()))
		}
		io.WriteString(b, `{`)
	} else {
		io.WriteString(b, `Object{`)
	}

	first2 := true
	ih := t.initHash(!inTypeSet)
	for _, key := range ih.Keys() {
		if inTypeSet && key == `parent` {
			continue
		}

		value := ih.Get(key, nil).(eval.PValue)
		if first2 {
			first2 = false
		} else {
			io.WriteString(b, `,`)
			if !f.IsAlt() {
				io.WriteString(b, ` `)
			}
		}
		if f.IsAlt() {
			io.WriteString(b, "\n")
			io.WriteString(b, padding2)
		}
		io.WriteString(b, key)
		io.WriteString(b, ` => `)
		switch key {
		case `attributes`, `functions`:
			// The keys should not be quoted in this hash
			io.WriteString(b, `{`)
			first3 := true
			value.(*HashValue).EachPair(func(name, typ eval.PValue) {
				if first3 {
					first3 = false
				} else {
					io.WriteString(b, `,`)
					if !f.IsAlt() {
						io.WriteString(b, ` `)
					}
				}
				if f.IsAlt() {
					io.WriteString(b, "\n")
					io.WriteString(b, padding3)
				}
				utils.PuppetQuote(b, name.String())
				io.WriteString(b, ` => `)
				typ.ToString(b, ctx3, g)
			})
			if f.IsAlt() {
				io.WriteString(b, "\n")
				io.WriteString(b, padding2)
			}
			io.WriteString(b, "}")
		default:
			cx := cti2
			if isContainer(value) {
				cx = ctx2
			}
			value.ToString(b, cx, g)
		}
	}
	if f.IsAlt() {
		io.WriteString(b, "\n")
		io.WriteString(b, padding1)
	}
	io.WriteString(b, "}")
}

func (t *objectType) Type() eval.PType {
	return &TypeType{t}
}

func (t *objectType) appendAttributeValues(c eval.Context, entries []*HashEntry, src reflect.Value) []*HashEntry {
	eval.AssertKind(c, src, reflect.Struct)
	dt := src.Type()
	n := src.NumField()
	for i := 0; i < n; i++ {
		field := dt.Field(i)
		if field.Anonymous && i == 0 && t.parent != nil {
			entries = t.resolvedParent(c).appendAttributeValues(c, entries, src.Field(i))
			continue
		}
		an := FieldName(c, &field)
		entries = append(entries, WrapHashEntry2(an, wrap(c, src.Field(i))))
	}
	return entries
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
	return newAttributesInfo(attrs, nonOptSize, t.EqualityAttributes().Keys())
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

func (t *objectType) initHash(includeName bool) *hash.StringHash {
	h := t.annotatable.initHash()
	if includeName && t.name != `` && t.name != `Object` {
		h.Put(KEY_NAME, WrapString(t.name))
	}
	if t.parent != nil {
		h.Put(KEY_PARENT, t.parent)
	}
	if !t.parameters.IsEmpty() {
		h.Put(KEY_TYPE_PARAMETERS, compressedMembersHash(t.parameters))
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
				h.Put(KEY_ATTRIBUTES, compressedMembersHash(others))
			}
			if len(constants) > 0 {
				h.Put(KEY_CONSTANTS, WrapHash(constants))
			}
		})
	}
	if !t.functions.IsEmpty() {
		h.Put(KEY_FUNCTIONS, compressedMembersHash(t.functions))
	}
	if t.equality != nil {
		ev := make([]eval.PValue, len(t.equality))
		for i, e := range t.equality {
			ev[i] = WrapString(e)
		}
		h.Put(KEY_EQUALITY, WrapArray(ev))
	}
	if t.serialization != nil {
		sv := make([]eval.PValue, len(t.serialization))
		for i, s := range t.serialization {
			sv[i] = WrapString(s)
		}
		h.Put(KEY_SERIALIZATION, WrapArray(sv))
	}
	return h
}

func (t *objectType) members(includeParent bool) *hash.StringHash {
	collector := hash.NewStringHash(7)
	t.collectMembers(includeParent, collector)
	return collector
}

func (t *objectType) namedInitSignature() eval.Signature {
	return NewCallableType(NewTupleType([]eval.PType{t.createInitType()}, NewIntegerType(1, 1)), t, nil)
}

func (t *objectType) positionalInitSignature() eval.Signature {
	ai := t.AttributesInfo()
	argTypes := make([]eval.PType, len(ai.Attributes()))
	for i, attr := range ai.Attributes() {
		argTypes[i] = attr.Type()
	}
	return NewCallableType(NewTupleType(argTypes, NewIntegerType(int64(ai.RequiredCount()), int64(len(argTypes)))), t, nil)
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

// setCreators takes one or two arguments. The first function is for positional arguments, the second
// for named arguments (expects exactly one argument which is a Hash.
func (t *objectType) setCreators(creators ...eval.DispatchFunction) {
	t.creators = creators
}

func (t *objectType) typeParameters(includeParent bool) *hash.StringHash {
	collector := hash.NewStringHash(5)
	t.collectParameters(includeParent, collector)
	return collector
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

func newObjectType(name, typeDecl string, creators ...eval.DispatchFunction) eval.ObjectType {
	p := parser.CreateParser()
	_, fileName, fileLine, _ := runtime.Caller(1)
	expr, err := p.Parse(fileName, fmt.Sprintf(`type %s = %s`, name, typeDecl), true)
	if err != nil {
		err = convertReported(err, fileName, fileLine)
		panic(err)
	}

	if ta, ok := expr.(*parser.TypeAlias); ok {
		rt, _ := CreateTypeDefinition(ta, eval.RUNTIME_NAME_AUTHORITY)
		ot := rt.(*objectType)
		ot.setCreators(creators...)
		registerResolvableType(ot)
		return ot
	}
	panic(convertReported(eval.Error2(expr, eval.EVAL_NO_DEFINITION, issue.H{`source`: ``, `type`: eval.TYPE, `name`: name}), fileName, fileLine))
}

func newObjectType2(name string, parent eval.PType, initHash *HashValue, creators ...eval.DispatchFunction) eval.ObjectType {
	ta := NewObjectType(name, parent, initHash)
	ta.setCreators(creators...)
	registerResolvableType(ta)
	return ta
}
