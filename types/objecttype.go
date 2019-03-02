package types

import (
	"fmt"
	"io"
	"reflect"
	"regexp"
	"runtime"
	"sync/atomic"

	dsl "github.com/lyraproj/puppet-parser/parser"
	"github.com/lyraproj/puppet-parser/validator"

	"bytes"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/puppet-evaluator/errors"
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/hash"
	"github.com/lyraproj/puppet-evaluator/utils"
)

var ObjectMetaType eval.ObjectType

func init() {
	oneArgCtor := func(ctx eval.Context, args []eval.Value) eval.Value {
		return newObjectType2(ctx, args...)
	}
	ObjectMetaType = newObjectType3(`Pcore::ObjectType`, AnyMetaType,
		WrapStringToValueMap(map[string]eval.Value{
			`attributes`: SingletonHash2(`_pcore_init_hash`, TypeObjectInitHash)}),
		oneArgCtor, oneArgCtor)
}

const (
	keyAnnotations         = `annotations`
	keyAttributes          = `attributes`
	keyConstants           = `constants`
	keyEquality            = `equality`
	keyEqualityIncludeType = `equality_include_type`
	keyFinal               = `final`
	keyFunctions           = `functions`
	keyKind                = `kind`
	keyName                = `name`
	keyOverride            = `override`
	keyParent              = `parent`
	keySerialization       = `serialization`
	keyType                = `type`
	keyTypeParameters      = `type_parameters`
	keyValue               = `value`

	defaultKind    = eval.AttributeKind(``)
	constant       = eval.AttributeKind(`constant`)
	derived        = eval.AttributeKind(`derived`)
	givenOrDerived = eval.AttributeKind(`given_or_derived`)
	reference      = eval.AttributeKind(`reference`)
)

var TypeNamePattern = regexp.MustCompile(`\A[A-Z][\w]*(?:::[A-Z][\w]*)*\z`)

var TypeTypeName = NewPatternType([]*RegexpType{NewRegexpTypeR(TypeNamePattern)})

var TypeMemberName = newPatternType2(NewRegexpTypeR(validator.PARAM_NAME))

var TypeMemberNames = newArrayType2(TypeMemberName)
var TypeParameters = NewHashType(TypeMemberName, DefaultNotUndefType(), nil)
var TypeAttributes = NewHashType(TypeMemberName, DefaultNotUndefType(), nil)
var TypeConstants = NewHashType(TypeMemberName, DefaultAnyType(), nil)
var TypeFunctions = NewHashType(newVariantType2(TypeMemberName, newPatternType2(NewRegexpTypeR(regexp.MustCompile(`^\[]$`)))), DefaultNotUndefType(), nil)
var TypeEquality = newVariantType2(TypeMemberName, TypeMemberNames)

var TypeObjectInitHash = NewStructType([]*StructElement{
	NewStructElement(newOptionalType3(keyName), TypeTypeName),
	NewStructElement(newOptionalType3(keyParent), NewVariantType(DefaultTypeType(), TypeTypeName)),
	NewStructElement(newOptionalType3(keyTypeParameters), TypeParameters),
	NewStructElement(newOptionalType3(keyAttributes), TypeAttributes),
	NewStructElement(newOptionalType3(keyConstants), TypeConstants),
	NewStructElement(newOptionalType3(keyFunctions), TypeFunctions),
	NewStructElement(newOptionalType3(keyEquality), TypeEquality),
	NewStructElement(newOptionalType3(keyEqualityIncludeType), DefaultBooleanType()),
	NewStructElement(newOptionalType3(keyEquality), TypeEquality),
	NewStructElement(newOptionalType3(keySerialization), TypeMemberNames),
	NewStructElement(newOptionalType3(keyAnnotations), typeAnnotations),
})

type objectType struct {
	annotatable
	hashKey             eval.HashKey
	name                string
	parent              eval.Type
	creators            []eval.DispatchFunction
	parameters          *hash.StringHash // map doesn't preserve order
	attributes          *hash.StringHash
	functions           *hash.StringHash
	equality            []string
	equalityIncludeType bool
	serialization       []string
	loader              eval.Loader
	initHashExpression  interface{} // Expression, *HashValue, or Go zero value
	attrInfo            *attributesInfo
	ctor                eval.Function
	goType              eval.AnnotatedType
	isInterface         bool
}

func (t *objectType) ReflectType(c eval.Context) (reflect.Type, bool) {
	if t.goType != nil {
		return t.goType.Type(), true
	}
	return c.ImplementationRegistry().TypeToReflected(t)
}

func ObjectToString(o eval.PuppetObject, s eval.FormatContext, b io.Writer, g eval.RDetect) {
	indent := s.Indentation()
	if indent.Breaks() {
		utils.WriteString(b, "\n")
		utils.WriteString(b, indent.Padding())
	}
	utils.WriteString(b, o.PType().Name())
	o.InitHash().(*HashValue).ToString2(b, s, eval.GetFormat(s.FormatMap(), o.PType()), '(', g)
}

var objectTypeDefault = &objectType{
	annotatable:         annotatable{annotations: emptyMap},
	name:                `Object`,
	hashKey:             eval.HashKey("\x00tObject"),
	parameters:          hash.EmptyStringHash,
	attributes:          hash.EmptyStringHash,
	functions:           hash.EmptyStringHash,
	equalityIncludeType: true}

func DefaultObjectType() *objectType {
	return objectTypeDefault
}

var objectId = int64(0)

func AllocObjectType() *objectType {
	return &objectType{
		annotatable:         annotatable{annotations: emptyMap},
		hashKey:             eval.HashKey(fmt.Sprintf("\x00tObject%d", atomic.AddInt64(&objectId, 1))),
		parameters:          hash.EmptyStringHash,
		attributes:          hash.EmptyStringHash,
		functions:           hash.EmptyStringHash,
		equalityIncludeType: true}
}

func BuildObjectType(name string, parent eval.Type, hashProducer func(eval.ObjectType) eval.OrderedMap) eval.ObjectType {
	obj := AllocObjectType()
	obj.name = name
	obj.parent = parent
	obj.initHashExpression = hashProducer(obj)
	return obj
}

func (t *objectType) Initialize(c eval.Context, args []eval.Value) {
	if len(args) == 1 {
		if om, ok := args[0].(eval.OrderedMap); ok {
			t.InitFromHash(c, om)
			return
		}
	}
	panic(eval.Error(eval.Failure, issue.H{`message`: `internal error when creating an Object data type`}))
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

func (t *objectType) Constructor(c eval.Context) eval.Function {
	if t.ctor == nil && t.name != `` {
		t.createNewFunction(c)
	}
	return t.ctor
}

func (t *objectType) Default() eval.Type {
	return objectTypeDefault
}

func (t *objectType) EachAttribute(includeParent bool, consumer func(attr eval.Attribute)) {
	if includeParent && t.parent != nil {
		t.resolvedParent().EachAttribute(includeParent, consumer)
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
		tp = tp.resolvedParent()
	}
	attrs := hash.NewStringHash(len(eqa))
	for _, an := range eqa {
		attrs.Put(an, t.GetAttribute(an))
	}
	return attrs
}

func (t *objectType) Equals(other interface{}, guard eval.Guard) bool {
	if t == other {
		return true
	}
	ot, ok := other.(*objectType)
	if !ok {
		return false
	}
	if t.initHashExpression != nil || ot.initHashExpression != nil {
		// Not yet resolved.
		return false
	}
	if t.name != ot.name {
		return false
	}
	if t.equalityIncludeType != ot.equalityIncludeType {
		return false
	}

	pa := t.resolvedParent()
	pb := ot.resolvedParent()
	if pa == nil {
		if pb != nil {
			return false
		}
	} else {
		if pb == nil {
			return false
		}
		if !pa.Equals(pb, guard) {
			return false
		}
	}
	if guard == nil {
		guard = make(eval.Guard)
	}
	if guard.Seen(t, ot) {
		return true
	}
	return t.attributes.Equals(ot.attributes, guard) &&
		t.functions.Equals(ot.functions, guard) &&
		t.parameters.Equals(ot.parameters, guard) &&
		eval.GuardedEquals(t.equality, ot.equality, guard) &&
		eval.GuardedEquals(t.serialization, ot.serialization, guard)
}

func (t *objectType) FromReflectedValue(c eval.Context, src reflect.Value) eval.PuppetObject {
	if t.goType != nil {
		return NewReflectedValue(t, src).(eval.PuppetObject)
	}
	if src.Kind() == reflect.Ptr {
		src = src.Elem()
	}
	entries := t.appendAttributeValues(c, make([]*HashEntry, 0), &src)
	return eval.New(c, t, WrapHash(entries)).(eval.PuppetObject)
}

func (t *objectType) Get(key string) (value eval.Value, ok bool) {
	if key == `_pcore_init_hash` {
		return t.InitHash(), true
	}
	return nil, false
}

func (t *objectType) GetAttribute(name string) eval.Attribute {
	a, _ := t.attributes.Get2(name, func() interface{} {
		p := t.resolvedParent()
		if p != nil {
			return p.GetAttribute(name)
		}
		return nil
	}).(eval.Attribute)
	return a
}

func (t *objectType) GetFunction(name string) eval.Function {
	f, _ := t.functions.Get2(name, func() interface{} {
		p := t.resolvedParent()
		if p != nil {
			return p.GetFunction(name)
		}
		return nil
	}).(eval.Function)
	return f
}

func (t *objectType) GetValue(key string, o eval.Value) (value eval.Value, ok bool) {
	if pu, ok := o.(eval.ReadableObject); ok {
		return pu.Get(key)
	}
	return nil, false
}

func (t *objectType) GoType() reflect.Type {
	if t.goType != nil {
		return t.goType.Type()
	}
	return nil
}

func (t *objectType) HasHashConstructor() bool {
	return t.creators == nil || len(t.creators) == 2
}

func (t *objectType) parseAttributeType(c eval.Context, receiverType, receiver string, typeString eval.StringValue) eval.Type {
	defer func() {
		if r := recover(); r != nil {
			if err, ok := r.(error); ok {
				var label string
				if receiverType == `` {
					label = fmt.Sprintf(`%s.%s`, t.Label(), receiver)
				} else {
					label = fmt.Sprintf(`%s %s[%s]`, receiverType, t.Label(), receiver)
				}
				panic(eval.Error(eval.BadTypeString,
					issue.H{
						`string`: typeString,
						`label`:  label,
						`detail`: err.Error()}))
			}
			panic(r)
		}
	}()
	return c.ParseType(typeString)
}

func (t *objectType) InitFromHash(c eval.Context, initHash eval.OrderedMap) {
	eval.AssertInstance(`object initializer`, TypeObjectInitHash, initHash)
	t.parameters = hash.EmptyStringHash
	t.attributes = hash.EmptyStringHash
	t.functions = hash.EmptyStringHash
	t.name = stringArg(initHash, keyName, t.name)

	if t.parent == nil {
		if pt, ok := initHash.Get4(keyParent); ok {
			switch pt := pt.(type) {
			case stringValue:
				t.parent = t.parseAttributeType(c, ``, `parent`, pt)
			case eval.ResolvableType:
				t.parent = pt.Resolve(c)
			default:
				t.parent = pt.(eval.Type)
			}
		}
	}

	parentMembers := hash.EmptyStringHash
	parentTypeParams := hash.EmptyStringHash
	var parentObjectType *objectType

	if t.parent != nil {
		t.checkSelfRecursion(c, t)
		parentObjectType = t.resolvedParent()
		parentMembers = parentObjectType.members(true)
		parentTypeParams = parentObjectType.typeParameters(true)
	}

	typeParameters := hashArg(initHash, keyTypeParameters)
	if !typeParameters.IsEmpty() {
		parameters := hash.NewStringHash(typeParameters.Len())
		typeParameters.EachPair(func(k, v eval.Value) {
			key := k.String()
			var paramType eval.Type
			var paramValue eval.Value
			if ph, ok := v.(*HashValue); ok {
				eval.AssertInstance(
					func() string { return fmt.Sprintf(`type_parameter %s[%s]`, t.Label(), key) },
					TypeTypeParameter, ph)
				paramType = typeArg(ph, keyType, DefaultTypeType())
				paramValue = ph.Get5(keyValue, nil)
			} else {
				if tn, ok := v.(stringValue); ok {
					// Type name. Load the type.
					paramType = t.parseAttributeType(c, `type_parameter`, key, tn)
				} else {
					paramType = eval.AssertInstance(
						func() string { return fmt.Sprintf(`type_parameter %s[%s]`, t.Label(), key) },
						DefaultTypeType(), v).(eval.Type)
				}
				paramValue = nil
			}
			if _, ok := paramType.(*OptionalType); !ok {
				paramType = NewOptionalType(paramType)
			}
			param := newTypeParameter(c, key, t, WrapStringToInterfaceMap(c, issue.H{
				keyType:  paramType,
				keyValue: paramValue}))
			assertOverride(param, parentTypeParams)
			parameters.Put(key, param)
		})
		parameters.Freeze()
		t.parameters = parameters
	}

	constants := hashArg(initHash, keyConstants)
	attributes := hashArg(initHash, keyAttributes)
	attrSpecs := hash.NewStringHash(constants.Len() + attributes.Len())
	attributes.EachPair(func(k, v eval.Value) {
		attrSpecs.Put(k.String(), v)
	})

	if !constants.IsEmpty() {
		constants.EachPair(func(k, v eval.Value) {
			key := k.String()
			if attrSpecs.Includes(key) {
				panic(eval.Error(eval.BothConstantAndAttribute, issue.H{`label`: t.Label(), `key`: key}))
			}
			value := v.(eval.Value)
			attrSpec := issue.H{
				keyType:  eval.Generalize(value.PType()),
				keyValue: value,
				keyKind:  constant}
			attrSpec[keyOverride] = parentMembers.Includes(key)
			attrSpecs.Put(key, WrapStringToInterfaceMap(c, attrSpec))
		})
	}

	if !attrSpecs.IsEmpty() {
		ah := hash.NewStringHash(attrSpecs.Len())
		attrSpecs.EachPair(func(key string, ifv interface{}) {
			value := ifv.(eval.Value)
			attrSpec, ok := value.(*HashValue)
			if !ok {
				var attrType eval.Type
				if tn, ok := value.(stringValue); ok {
					// Type name. Load the type.
					attrType = t.parseAttributeType(c, `attribute`, key, tn)
				} else {
					attrType = eval.AssertInstance(
						func() string { return fmt.Sprintf(`attribute %s[%s]`, t.Label(), key) },
						DefaultTypeType(), value).(eval.Type)
				}
				h := issue.H{keyType: attrType}
				if _, ok = attrType.(*OptionalType); ok {
					h[keyValue] = eval.Undef
				}
				attrSpec = WrapStringToInterfaceMap(c, h)
			}
			attr := newAttribute(c, key, t, attrSpec)
			assertOverride(attr, parentMembers)
			ah.Put(key, attr)
		})
		ah.Freeze()
		t.attributes = ah
	}
	isInterface := t.attributes.IsEmpty() && (parentObjectType == nil || parentObjectType.isInterface)

	if t.goType != nil && t.attributes.IsEmpty() {
		if pt, ok := PrimitivePType(t.goType.Type()); ok {
			t.isInterface = false

			// Create the special attribute that holds the primitive value that is
			// reflectable to/from the the go type
			attrs := make([]*HashEntry, 2)
			attrs[0] = WrapHashEntry2(keyType, pt)
			attrs[1] = WrapHashEntry2(KeyGoName, stringValue(keyValue))
			ah := hash.NewStringHash(1)
			ah.Put(keyValue, newAttribute(c, keyValue, t, WrapHash(attrs)))
			ah.Freeze()
			t.attributes = ah
		}
	}

	funcSpecs := hashArg(initHash, keyFunctions)
	if funcSpecs.IsEmpty() {
		if isInterface && parentObjectType == nil {
			isInterface = false
		}
	} else {
		functions := hash.NewStringHash(funcSpecs.Len())
		funcSpecs.EachPair(func(key, value eval.Value) {
			if attributes.IncludesKey(key) {
				panic(eval.Error(eval.MemberNameConflict, issue.H{`label`: fmt.Sprintf(`function %s[%s]`, t.Label(), key)}))
			}
			funcSpec, ok := value.(*HashValue)
			if !ok {
				var funcType eval.Type
				if tn, ok := value.(stringValue); ok {
					// Type name. Load the type.
					funcType = t.parseAttributeType(c, `function`, key.String(), tn)
				} else {
					funcType = eval.AssertInstance(
						func() string { return fmt.Sprintf(`function %s[%s]`, t.Label(), key) },
						typeFunctionType, value).(eval.Type)
				}
				funcSpec = WrapStringToInterfaceMap(c, issue.H{keyType: funcType})
			}
			fnc := newFunction(c, key.String(), t, funcSpec)
			assertOverride(fnc, parentMembers)
			functions.Put(key.String(), fnc)
		})
		functions.Freeze()
		t.functions = functions
	}
	t.equalityIncludeType = boolArg(initHash, keyEqualityIncludeType, true)

	var equality []string
	eq := initHash.Get5(keyEquality, nil)
	if es, ok := eq.(stringValue); ok {
		equality = []string{string(es)}
	} else if ea, ok := eq.(*ArrayValue); ok {
		equality = make([]string, ea.Len())
	} else {
		equality = nil
	}
	for _, attrName := range equality {
		mbr := t.attributes.Get2(attrName, func() interface{} {
			return t.functions.Get2(attrName, func() interface{} {
				return parentMembers.Get(attrName, nil)
			})
		})
		attr, ok := mbr.(eval.Attribute)

		if !ok {
			if mbr == nil {
				panic(eval.Error(eval.EqualityAttributeNotFound, issue.H{`label`: t.Label(), `attribute`: attrName}))
			}
			panic(eval.Error(eval.EqualityNotAttribute, issue.H{`label`: t.Label(), `member`: mbr.(eval.AnnotatedMember).Label()}))
		}
		if attr.Kind() == constant {
			panic(eval.Error(eval.EqualityOnConstant, issue.H{`label`: t.Label(), `attribute`: mbr.(eval.AnnotatedMember).Label()}))
		}
		// Assert that attribute is not already include by parent equality
		if ok && parentObjectType != nil && parentObjectType.EqualityAttributes().Includes(attrName) {
			includingParent := t.findEqualityDefiner(attrName)
			panic(eval.Error(eval.EqualityRedefined, issue.H{`label`: t.Label(), `attribute`: attr.Label(), `including_parent`: includingParent}))
		}
	}
	t.equality = equality

	se, ok := initHash.Get5(keySerialization, nil).(*ArrayValue)
	if ok {
		serialization := make([]string, se.Len())
		var optFound eval.Attribute
		se.EachWithIndex(func(elem eval.Value, i int) {
			attrName := elem.String()
			mbr := t.attributes.Get2(attrName, func() interface{} {
				return t.functions.Get2(attrName, func() interface{} {
					return parentMembers.Get(attrName, nil)
				})
			})
			attr, ok := mbr.(eval.Attribute)

			if !ok {
				if mbr == nil {
					panic(eval.Error(eval.SerializationAttributeNotFound, issue.H{`label`: t.Label(), `attribute`: attrName}))
				}
				panic(eval.Error(eval.SerializationNotAttribute, issue.H{`label`: t.Label(), `member`: mbr.(eval.AnnotatedMember).Label()}))
			}
			if attr.Kind() == constant || attr.Kind() == derived {
				panic(eval.Error(eval.SerializationBadKind, issue.H{`label`: t.Label(), `kind`: attr.Kind(), `attribute`: attr.Label()}))
			}
			if attr.Kind() == givenOrDerived || attr.HasValue() {
				optFound = attr
			} else if optFound != nil {
				panic(eval.Error(eval.SerializationRequiredAfterOptional, issue.H{`label`: t.Label(), `required`: attr.Label(), `optional`: optFound.Label()}))
			}
			serialization[i] = attrName
		})
		t.serialization = serialization
	}

	t.isInterface = isInterface
	t.attrInfo = t.createAttributesInfo()
	t.annotatable.initialize(initHash.(*HashValue))
	t.loader = c.Loader()
}

func (t *objectType) Implements(ifd eval.ObjectType, g eval.Guard) bool {
	if !ifd.IsInterface() {
		return false
	}

	for _, f := range ifd.Functions(true) {
		m, ok := t.Member(f.Name())
		if !ok {
			return false
		}
		mf, ok := m.(eval.ObjFunc)
		if !ok {
			return false
		}
		if !f.Type().Equals(mf.Type(), g) {
			return false
		}
	}
	return true
}

func (t *objectType) InitHash() eval.OrderedMap {
	return WrapStringPValue(t.initHash(true))
}

func (t *objectType) IsInterface() bool {
	return t.isInterface
}

func (t *objectType) IsAssignable(o eval.Type, g eval.Guard) bool {
	var ot *objectType
	switch o := o.(type) {
	case *objectType:
		ot = o
	case *objectTypeExtension:
		ot = o.baseType
	default:
		return false
	}

	if t == DefaultObjectType() {
		return true
	}

	if t.isInterface {
		return ot.Implements(t, g)
	}

	if t == DefaultObjectType() || t.Equals(ot, g) {
		return true
	}
	if ot.parent != nil {
		return t.IsAssignable(ot.parent, g)
	}
	return false
}

func (t *objectType) IsInstance(o eval.Value, g eval.Guard) bool {
	if po, ok := o.(eval.Type); ok {
		return isAssignable(t, po.MetaType())
	}
	return isAssignable(t, o.PType())
}

func (t *objectType) IsParameterized() bool {
	if !t.parameters.IsEmpty() {
		return true
	}
	p := t.resolvedParent()
	if p != nil {
		return p.IsParameterized()
	}
	return false
}

func (t *objectType) IsMetaType() bool {
	return eval.IsAssignable(AnyMetaType, t)
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
			pm, _ := t.resolvedParent().Member(name)
			return pm
		})
	})
	if mbr == nil {
		return nil, false
	}
	return mbr.(eval.CallableMember), true
}

func (t *objectType) MetaType() eval.ObjectType {
	return ObjectMetaType
}

func (t *objectType) Name() string {
	return t.name
}

func (t *objectType) Parameters() []eval.Value {
	return t.Parameters2(true)
}

func (t *objectType) Parameters2(includeName bool) []eval.Value {
	if t == objectTypeDefault {
		return eval.EmptyValues
	}
	return []eval.Value{WrapStringPValue(t.initHash(includeName))}
}

func (t *objectType) Parent() eval.Type {
	return t.parent
}

func (t *objectType) Resolve(c eval.Context) eval.Type {
	if t.initHashExpression != nil {
		ihe := t.initHashExpression
		t.initHashExpression = nil

		if prt, ok := t.parent.(eval.ResolvableType); ok {
			t.parent = resolveTypeRefs(c, prt).(eval.Type)
		}

		var initHash eval.OrderedMap
		if lh, ok := ihe.(*dsl.LiteralHash); ok {
			if ec, ok := c.(eval.EvaluationContext); ok {
				c.(eval.EvaluationContext).DoStatic(func() {
					initHash = eval.Evaluate(ec, lh).(eval.OrderedMap)
				})
			}
		} else if hv, ok := ihe.(*HashValue); ok {
			initHash = resolveTypeRefs(c, hv).(eval.OrderedMap)
		} else if tg, ok := ihe.(*taggedType); ok {
			t.goType = tg
			initHash = c.Reflector().InitializerFromTagged(t.name, t.parent, tg)
			c.ImplementationRegistry().RegisterType(t, tg.Type())
		} else {
			tg := eval.NewTaggedType(reflect.TypeOf(ihe), nil)
			t.goType = tg
			initHash = c.Reflector().InitializerFromTagged(t.name, t.parent, tg)
			c.ImplementationRegistry().RegisterType(t, tg.Type())
		}
		t.InitFromHash(c, initHash)
	}
	return t
}

func (t *objectType) CanSerializeAsString() bool {
	return t == objectTypeDefault
}

func (t *objectType) SerializationString() string {
	return t.String()
}

func (t *objectType) String() string {
	return eval.ToString(t)
}

func (t *objectType) ToKey() eval.HashKey {
	return t.hashKey
}

func (t *objectType) ToReflectedValue(c eval.Context, src eval.PuppetObject, dest reflect.Value) {
	dt := dest.Type()
	rf := c.Reflector()
	fs := rf.Fields(dt)
	for i, field := range fs {
		f := dest.Field(i)
		if field.Anonymous && i == 0 && t.parent != nil {
			t.resolvedParent().ToReflectedValue(c, src, f)
			continue
		}
		an := rf.FieldName(&field)
		if av, ok := src.Get(an); ok {
			rf.ReflectTo(av, f)
			continue
		}
		panic(eval.Error(eval.AttributeNotFound, issue.H{`name`: an}))
	}
}

func (t *objectType) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	f := eval.GetFormat(s.FormatMap(), t.PType())
	switch f.FormatChar() {
	case 's', 'p':
		quoted := f.IsAlt() && f.FormatChar() == 's'
		if quoted || f.HasStringFlags() {
			bld := bytes.NewBufferString(``)
			t.basicTypeToString(bld, f, s, g)
			f.ApplyStringFlags(b, bld.String(), quoted)
		} else {
			t.basicTypeToString(b, f, s, g)
		}
	default:
		panic(s.UnsupportedFormat(t.PType(), `sp`, f))
	}
}

func (t *objectType) basicTypeToString(b io.Writer, f eval.Format, s eval.FormatContext, g eval.RDetect) {

	if t.Equals(DefaultObjectType(), nil) {
		utils.WriteString(b, `Object`)
		return
	}

	typeSetName, inTypeSet := s.Property(`typeSet`)
	if ex, ok := s.Property(`expanded`); !(ok && ex == `true`) {
		name := t.Name()
		if inTypeSet {
			name = stripTypeSetName(typeSetName, name)
		}
		utils.WriteString(b, name)
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
		cf = DefaultContainerFormats
	}

	ctx2 := eval.NewFormatContext2(indent2, s.FormatMap(), s.Properties())
	cti2 := eval.NewFormatContext2(indent2, cf, s.Properties())
	ctx3 := eval.NewFormatContext2(indent3, s.FormatMap(), s.Properties())

	if inTypeSet {
		if t.parent != nil {
			utils.WriteString(b, stripTypeSetName(typeSetName, t.parent.Name()))
		}
		utils.WriteString(b, `{`)
	} else {
		utils.WriteString(b, `Object[{`)
	}

	first2 := true
	ih := t.initHash(!inTypeSet)
	for _, key := range ih.Keys() {
		if inTypeSet && key == `parent` {
			continue
		}

		value := ih.Get(key, nil).(eval.Value)
		if first2 {
			first2 = false
		} else {
			utils.WriteString(b, `,`)
			if !f.IsAlt() {
				utils.WriteString(b, ` `)
			}
		}
		if f.IsAlt() {
			utils.WriteString(b, "\n")
			utils.WriteString(b, padding2)
		}
		utils.WriteString(b, key)
		utils.WriteString(b, ` => `)
		switch key {
		case `attributes`, `functions`:
			// The keys should not be quoted in this hash
			utils.WriteString(b, `{`)
			first3 := true
			value.(*HashValue).EachPair(func(name, typ eval.Value) {
				if first3 {
					first3 = false
				} else {
					utils.WriteString(b, `,`)
					if !f.IsAlt() {
						utils.WriteString(b, ` `)
					}
				}
				if f.IsAlt() {
					utils.WriteString(b, "\n")
					utils.WriteString(b, padding3)
				}
				utils.PuppetQuote(b, name.String())
				utils.WriteString(b, ` => `)
				typ.ToString(b, ctx3, g)
			})
			if f.IsAlt() {
				utils.WriteString(b, "\n")
				utils.WriteString(b, padding2)
			}
			utils.WriteString(b, "}")
		default:
			cx := cti2
			if isContainer(value, s) {
				cx = ctx2
			}
			value.ToString(b, cx, g)
		}
	}
	if f.IsAlt() {
		utils.WriteString(b, "\n")
		utils.WriteString(b, padding1)
	}
	utils.WriteString(b, "}")
	if !inTypeSet {
		utils.WriteString(b, "]")
	}
}

func (t *objectType) PType() eval.Type {
	return &TypeType{t}
}

func (t *objectType) appendAttributeValues(c eval.Context, entries []*HashEntry, src *reflect.Value) []*HashEntry {
	dt := src.Type()
	rf := c.Reflector()
	fs := rf.Fields(dt)

	for i, field := range fs {
		sf := src.Field(i)
		if sf.Kind() == reflect.Ptr {
			sf = sf.Elem()
		}
		if field.Anonymous && i == 0 && t.parent != nil {
			entries = t.resolvedParent().appendAttributeValues(c, entries, &sf)
		} else {
			entries = append(entries, WrapHashEntry2(rf.FieldName(&field), wrap(c, sf)))
		}
	}
	return entries
}

func (t *objectType) checkSelfRecursion(c eval.Context, originator *objectType) {
	if t.parent != nil {
		op := t.resolvedParent()
		if eval.Equals(op, originator) {
			panic(eval.Error(eval.ObjectInheritsSelf, issue.H{`label`: originator.Label()}))
		}
		op.checkSelfRecursion(c, originator)
	}
}

func (t *objectType) collectAttributes(includeParent bool, collector *hash.StringHash) {
	if includeParent && t.parent != nil {
		t.resolvedParent().collectAttributes(true, collector)
	}
	collector.PutAll(t.attributes)
}

func (t *objectType) Functions(includeParent bool) []eval.ObjFunc {
	collector := hash.NewStringHash(7)
	t.collectFunctions(includeParent, collector)
	vs := collector.Values()
	fs := make([]eval.ObjFunc, len(vs))
	for i, v := range vs {
		fs[i] = v.(eval.ObjFunc)
	}
	return fs
}

func (t *objectType) collectFunctions(includeParent bool, collector *hash.StringHash) {
	if includeParent && t.parent != nil {
		t.resolvedParent().collectFunctions(true, collector)
	}
	collector.PutAll(t.functions)
}

func (t *objectType) collectMembers(includeParent bool, collector *hash.StringHash) {
	if includeParent && t.parent != nil {
		t.resolvedParent().collectMembers(true, collector)
	}
	collector.PutAll(t.attributes)
	collector.PutAll(t.functions)
}

func (t *objectType) collectParameters(includeParent bool, collector *hash.StringHash) {
	if includeParent && t.parent != nil {
		t.resolvedParent().collectParameters(true, collector)
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
			case constant, derived:
			case givenOrDerived:
				optAttrs = append(optAttrs, attr)
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
		case constant, derived:
		case givenOrDerived:
			elements = append(elements, NewStructElement(newOptionalType3(attr.Name()), attr.Type()))
		default:
			var key eval.Type
			if attr.HasValue() {
				key = newOptionalType3(attr.Name())
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
	var dl eval.DefiningLoader
	if t.loader == nil {
		dl = c.DefiningLoader()
	} else {
		dl = t.loader.(eval.DefiningLoader)
	}

	var functions []eval.DispatchFunction
	if t.creators != nil {
		functions = t.creators
		if functions[0] == nil {
			// Specific instruction not to create a constructor
			return
		}
	} else {
		tn := eval.NewTypedName(eval.NsAllocator, t.name)
		le := dl.LoadEntry(c, tn)
		if le == nil || le.Value() == nil {
			dl.SetEntry(tn, eval.NewLoaderEntry(eval.MakeGoAllocator(func(ctx eval.Context, args []eval.Value) eval.Value {
				return AllocObjectValue(t)
			}), nil))
		}

		functions = []eval.DispatchFunction{
			// Positional argument creator
			func(c eval.Context, args []eval.Value) eval.Value {
				return NewObjectValue(c, t, args)
			},
			// Named argument creator
			func(c eval.Context, args []eval.Value) eval.Value {
				return newObjectValue2(c, t, args[0].(*HashValue))
			}}
	}

	paCreator := func(d eval.Dispatch) {
		for i, attr := range pi.Attributes() {
			switch attr.Kind() {
			case constant, derived:
			case givenOrDerived:
				d.OptionalParam2(attr.Type())
			default:
				if i >= pi.RequiredCount() {
					d.OptionalParam2(attr.Type())
				} else {
					d.Param2(attr.Type())
				}
			}
		}
		d.Function(functions[0])
	}

	var creators []eval.DispatchCreator
	if len(functions) > 1 {
		// A named argument constructor exists. Place it first.
		creators = []eval.DispatchCreator{func(d eval.Dispatch) {
			d.Param2(t.createInitType())
			d.Function(functions[1])
		}, paCreator}
	} else {
		creators = []eval.DispatchCreator{paCreator}
	}

	t.ctor = eval.MakeGoConstructor(t.name, creators...).Resolve(c)
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

func (t *objectType) initHash(includeName bool) *hash.StringHash {
	h := t.annotatable.initHash()
	if includeName && t.name != `` && t.name != `Object` {
		h.Put(keyName, stringValue(t.name))
	}
	if t.parent != nil {
		h.Put(keyParent, t.parent)
	}
	if !t.parameters.IsEmpty() {
		h.Put(keyTypeParameters, compressedMembersHash(t.parameters))
	}
	if !t.attributes.IsEmpty() {
		// Divide attributes into constants and others
		constants := make([]*HashEntry, 0)
		others := hash.NewStringHash(5)
		t.attributes.EachPair(func(key string, value interface{}) {
			a := value.(eval.Attribute)
			if a.Kind() == constant && eval.Equals(a.Type(), eval.Generalize(a.Value().PType())) {
				constants = append(constants, WrapHashEntry2(key, a.Value()))
			} else {
				others.Put(key, a)
			}
			if !others.IsEmpty() {
				h.Put(keyAttributes, compressedMembersHash(others))
			}
			if len(constants) > 0 {
				h.Put(keyConstants, WrapHash(constants))
			}
		})
	}
	if !t.functions.IsEmpty() {
		h.Put(keyFunctions, compressedMembersHash(t.functions))
	}
	if t.equality != nil {
		ev := make([]eval.Value, len(t.equality))
		for i, e := range t.equality {
			ev[i] = stringValue(e)
		}
		h.Put(keyEquality, WrapValues(ev))
	}

	if !t.equalityIncludeType {
		h.Put(keyEqualityIncludeType, BooleanFalse)
	}

	if t.serialization != nil {
		sv := make([]eval.Value, len(t.serialization))
		for i, s := range t.serialization {
			sv[i] = stringValue(s)
		}
		h.Put(keySerialization, WrapValues(sv))
	}
	return h
}

func (t *objectType) members(includeParent bool) *hash.StringHash {
	collector := hash.NewStringHash(7)
	t.collectMembers(includeParent, collector)
	return collector
}

func (t *objectType) resolvedParent() *objectType {
	tp := t.parent
	for {
		switch at := tp.(type) {
		case nil:
			return nil
		case *objectType:
			return at
		case *TypeAliasType:
			tp = at.resolvedType
		default:
			panic(eval.Error(eval.IllegalObjectInheritance, issue.H{`label`: t.Label(), `type`: tp.PType().String()}))
		}
	}
}

// setCreators takes one or two arguments. The first function is for positional arguments, the second
// for named arguments (expects exactly one argument which is a Hash.
func (t *objectType) setCreators(creators ...eval.DispatchFunction) {
	t.creators = creators
}

func (t *objectType) typeParameters(includeParent bool) *hash.StringHash {
	c := hash.NewStringHash(5)
	t.collectParameters(includeParent, c)
	return c
}

func compressedMembersHash(mh *hash.StringHash) *HashValue {
	he := make([]*HashEntry, 0, mh.Len())
	mh.EachPair(func(key string, value interface{}) {
		fh := value.(eval.AnnotatedMember).InitHash()
		if fh.Len() == 1 {
			tp := fh.Get5(keyType, nil)
			if tp != nil {
				he = append(he, WrapHashEntry2(key, tp))
				return
			}
		}
		he = append(he, WrapHashEntry2(key, fh))
	})
	return WrapHash(he)
}

func resolveTypeRefs(c eval.Context, v interface{}) eval.Value {
	switch v := v.(type) {
	case *HashValue:
		he := make([]*HashEntry, v.Len())
		i := 0
		v.EachPair(func(key, value eval.Value) {
			he[i] = WrapHashEntry(
				resolveTypeRefs(c, key), resolveTypeRefs(c, value))
			i++
		})
		return WrapHash(he)
	case *ArrayValue:
		ae := make([]eval.Value, v.Len())
		i := 0
		v.Each(func(value eval.Value) {
			ae[i] = resolveTypeRefs(c, value)
			i++
		})
		return WrapValues(ae)
	case eval.ResolvableType:
		return v.Resolve(c)
	default:
		return v.(eval.Value)
	}
}

func ObjectTypeFromAST(name string, parent eval.Type, initHashExpression *dsl.LiteralHash) *objectType {
	t := AllocObjectType()
	if name == `` {
		ne := initHashExpression.Get(`name`)
		if s, ok := ne.(*dsl.LiteralString); ok {
			name = s.StringValue()
		}
	}
	t.name = name
	t.initHashExpression = initHashExpression
	t.parent = parent
	return t
}

func newObjectType(name, typeDecl string, creators ...eval.DispatchFunction) eval.ObjectType {
	ta, err := Parse(typeDecl)
	if err != nil {
		_, fileName, fileLine, _ := runtime.Caller(1)
		panic(convertReported(err, fileName, fileLine))
	}
	if h, ok := ta.(*HashValue); ok {
		// "type = {}"
		return newObjectType3(name, nil, h, creators...)
	}
	if dt, ok := ta.(*DeferredType); ok {
		ps := dt.Parameters()
		if len(ps) == 1 {
			if h, ok := ps[0].(*HashValue); ok {
				var p eval.Type
				if pn := dt.Name(); pn != `TypeSet` && pn != `Object` {
					p = NewTypeReferenceType(pn)
				}
				return newObjectType3(name, p, h, creators...)
			}
		}
	}
	panic(eval.Error(eval.NoDefinition, issue.H{`source`: ``, `type`: eval.NsType, `name`: name}))
}

func newObjectType2(c eval.Context, args ...eval.Value) *objectType {
	argc := len(args)
	switch argc {
	case 0:
		return DefaultObjectType()
	case 1:
		arg := args[0]
		if initHash, ok := arg.(*HashValue); ok {
			if initHash.IsEmpty() {
				return DefaultObjectType()
			}
			obj := AllocObjectType()
			obj.InitFromHash(c, initHash)
			obj.loader = c.Loader()
			return obj
		}
		panic(NewIllegalArgumentType(`Object[]`, 0, `Hash[String,Any]`, arg.PType()))
	default:
		panic(errors.NewIllegalArgumentCount(`Object[]`, `1`, argc))
	}
}

func newObjectType3(name string, parent eval.Type, initHash eval.OrderedMap, creators ...eval.DispatchFunction) *objectType {
	if name == `` {
		name = initHash.Get5(`name`, emptyString).String()
	}
	obj := AllocObjectType()
	obj.name = name
	obj.initHashExpression = initHash
	obj.parent = parent
	obj.setCreators(creators...)
	registerResolvableType(obj)
	return obj
}

func newGoObjectType(name string, rType reflect.Type, typeDecl string, creators ...eval.DispatchFunction) eval.ObjectType {
	t := newObjectType(name, typeDecl, creators...)
	registerMapping(t, rType)
	return t
}
