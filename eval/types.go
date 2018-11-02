package eval

import (
	"github.com/puppetlabs/go-issues/issue"
	"github.com/puppetlabs/go-semver/semver"
	"reflect"
)

type (
	Visitor func(t Type)

	Type interface {
		Value

		IsInstance(o Value, g Guard) bool

		IsAssignable(t Type, g Guard) bool

		MetaType() ObjectType

		Name() string

		Accept(visitor Visitor, g Guard)
	}

	SizedType interface {
		Type

		Size() Type
	}

	Resolvable interface {
		Resolve(c Context)
	}

	ResolvableType interface {
		Type

		Resolve(c Context) Type
	}

	ObjectTypeAndCtor interface {
		Type() ObjectType

		Creator() DispatchFunction
	}

	ParameterizedType interface {
		Type

		Default() Type

		// Parameters returns the parameters that is needed in order to recreate
		// an instance of the parameterized type.
		Parameters() []Value
	}

	SerializeAsString interface {
		SerializationString() string
	}

	Annotatable interface {
		Annotations() OrderedMap
	}

	CallableMember interface {
		Call(c Context, receiver Value, block Lambda, args []Value) Value
	}

	CallableGoMember interface {
		// Call a member on a struct pointer with the given arguments
		CallGo(c Context, receiver interface{}, args ...interface{}) interface{}
	}

	TypeWithCallableMembers interface {
		// Member returns an attribute reader or other function and true, or nil and false if no such member exists
		Member(name string) (CallableMember, bool)
	}

	AnnotatedMember interface {
		Equality
		CallableMember

		Name() string

		Label() string

		FeatureType() string

		Container() ObjectType

		Type() Type

		Override() bool

		Final() bool

		InitHash() OrderedMap

		Accept(v Visitor, g Guard)

		CallableType() Type
	}

	AttributeKind string

	Attribute interface {
		AnnotatedMember
		Kind() AttributeKind

		// Get returs this attributes value in the given instance
		Get(instance Value) Value

		// HasValue returns true if a value has been defined for this attribute.
		HasValue() bool

		// Default returns true if the given value equals the default value for this attribute
		Default(value Value) bool

		// Value returns the value of this attribute, or raises an error if no value has been defined.
		Value() Value

		// GoName Returns the name of the struct field that this attribute maps to when applicable or
		// an empty string.
		GoName() string
	}

	ObjFunc interface {
		AnnotatedMember

		// GoName Returns the name of the struct field that this attribute maps to when applicable or
		// an empty string.
		GoName() string
	}

	AttributesInfo interface {
		NameToPos() map[string]int

		PosToName() map[int]string

		Attributes() []Attribute

		EqualityAttributeIndex() []int

		RequiredCount() int

		PositionalFromHash(hash OrderedMap) []Value
	}

	ObjectType interface {
		ParameterizedType
		TypeWithCallableMembers

		HasHashConstructor() bool

		Functions(includeParent bool) []ObjFunc

		// Returns the Go reflect.Type that this type was reflected from, if any.
		//
		GoType() reflect.Type

		// IsInterface returns true for non parameterized types that contains only methods
		IsInterface() bool

		IsMetaType() bool

		IsParameterized() bool

		// Implements returns true the receiver implements all methods of ObjectType
		Implements(ObjectType, Guard) bool

		AttributesInfo() AttributesInfo

		// Constructor returns the function that creates instances of the type
		Constructor() Function

		// FromReflectedValue creates a new instance of the reciever type
		// and initializes that instance from the given src
		FromReflectedValue(c Context, src reflect.Value) PuppetObject

		// Parent returns the type that this type inherits from or nil if
		// the type doesn't have a parent
		Parent() Type

		// ToReflectedValue copies values from src to dest. The src argument
		// must be an instance of the receiver. The dest argument must be
		// a reflected struct. The src must be able to deliver a value to
		// each of the exported fields in dest.
		//
		// Puppets name convention stipulates lower case names using
		// underscores to separate words. The Go conversion is to use
		// camel cased names. ReflectValueTo will convert camel cased names
		// into names with underscores.
		ToReflectedValue(c Context, src PuppetObject, dest reflect.Value)
	}

	TypeSet interface {
		ParameterizedType

		// GetType returns the given type from the receiver together with
		// a flag indicating success or failure
		GetType(typedName TypedName) (Type, bool)

		// GetType2 is like GetType but uses a string to identify the type
		GetType2(name string) (Type, bool)

		// Authority returns the name authority of the receiver
		NameAuthority() URI

		// TypedName returns the name of this type set as a TypedName
		TypedName() TypedName

		// Types returns a hash of all types contained in this set. The keyes
		// in this hash are relative to the receiver name
		Types() OrderedMap

		// Version returns the version of the receiver
		Version() semver.Version
	}

	TypeWithContainedType interface {
		Type

		ContainedType() Type
	}

	// Generalizable implemented by all parameterized types that have type parameters
	Generalizable interface {
		ParameterizedType
		Generic() Type
	}
)

var CommonType func(a Type, b Type) Type

var GenericType func(t Type) Type

var IsInstance func(puppetType Type, value Value) bool

// IsAssignable answers if t is assignable to this type
var IsAssignable func(puppetType Type, other Type) bool

var Generalize func(t Type) Type

var Normalize func(t Type) Type

var DefaultFor func(t Type) Type

var ToArray func(elements []Value) List

func All(elements []Value, predicate Predicate) bool {
	for _, elem := range elements {
		if !predicate(elem) {
			return false
		}
	}
	return true
}

func All2(array List, predicate Predicate) bool {
	top := array.Len()
	for idx := 0; idx < top; idx++ {
		if !predicate(array.At(idx)) {
			return false
		}
	}
	return true
}

func Any(elements []Value, predicate Predicate) bool {
	for _, elem := range elements {
		if predicate(elem) {
			return true
		}
	}
	return false
}

func Any2(array List, predicate Predicate) bool {
	top := array.Len()
	for idx := 0; idx < top; idx++ {
		if predicate(array.At(idx)) {
			return true
		}
	}
	return false
}

func AssertType(pfx interface{}, expected, actual Type) Type {
	if !IsAssignable(expected, actual) {
		panic(Error(EVAL_TYPE_MISMATCH, issue.H{`detail`: DescribeMismatch(getPrefix(pfx), expected, actual)}))
	}
	return actual
}

func AssertInstance(pfx interface{}, expected Type, value Value) Value {
	if !IsInstance(expected, value) {
		panic(Error(EVAL_TYPE_MISMATCH, issue.H{`detail`: DescribeMismatch(getPrefix(pfx), expected, DetailedValueType(value))}))
	}
	return value
}

func Each(elements []Value, consumer Consumer) {
	for _, elem := range elements {
		consumer(elem)
	}
}

func Each2(array List, consumer Consumer) {
	top := array.Len()
	for idx := 0; idx < top; idx++ {
		consumer(array.At(idx))
	}
}

func Find(elements []Value, dflt Value, predicate Predicate) Value {
	for _, elem := range elements {
		if predicate(elem) {
			return elem
		}
	}
	return dflt
}

func Find2(array List, dflt Value, predicate Predicate) Value {
	top := array.Len()
	for idx := 0; idx < top; idx++ {
		v := array.At(idx)
		if predicate(v) {
			return v
		}
	}
	return dflt
}

func Map1(elements []Value, mapper Mapper) []Value {
	result := make([]Value, len(elements))
	for idx, elem := range elements {
		result[idx] = mapper(elem)
	}
	return result
}

func Map2(array List, mapper Mapper) List {
	top := array.Len()
	result := make([]Value, top)
	for idx := 0; idx < top; idx++ {
		result[idx] = mapper(array.At(idx))
	}
	return ToArray(result)
}

func MapTypes(types []Type, mapper TypeMapper) []Value {
	result := make([]Value, len(types))
	for idx, elem := range types {
		result[idx] = mapper(elem)
	}
	return result
}

// New creates a new instance of type t
func New(c Context, t Type, args ...Value) Value {
	if ctor, ok := Load(c, NewTypedName(CONSTRUCTOR, t.Name())); ok {
		return ctor.(Function).Call(c, nil, args...)
	}
	panic(Error(EVAL_CTOR_NOT_FOUND, issue.H{`type`: t.Name()}))
}

func Reduce(elements []Value, memo Value, reductor BiMapper) Value {
	for _, elem := range elements {
		memo = reductor(memo, elem)
	}
	return memo
}

func Reduce2(array List, memo Value, reductor BiMapper) Value {
	top := array.Len()
	for idx := 0; idx < top; idx++ {
		memo = reductor(memo, array.At(idx))
	}
	return memo
}

func Select1(elements []Value, predicate Predicate) []Value {
	result := make([]Value, 0, 8)
	for _, elem := range elements {
		if predicate(elem) {
			result = append(result, elem)
		}
	}
	return result
}

func Select2(array List, predicate Predicate) List {
	result := make([]Value, 0, 8)
	top := array.Len()
	for idx := 0; idx < top; idx++ {
		v := array.At(idx)
		if predicate(v) {
			result = append(result, v)
		}
	}
	return ToArray(result)
}

func Reject(elements []Value, predicate Predicate) []Value {
	result := make([]Value, 0, 8)
	for _, elem := range elements {
		if !predicate(elem) {
			result = append(result, elem)
		}
	}
	return result
}

func Reject2(array List, predicate Predicate) List {
	result := make([]Value, 0, 8)
	top := array.Len()
	for idx := 0; idx < top; idx++ {
		v := array.At(idx)
		if !predicate(v) {
			result = append(result, v)
		}
	}
	return ToArray(result)
}

var DescribeSignatures func(signatures []Signature, argsTuple Type, block Lambda) string

var DescribeMismatch func(pfx string, expected Type, actual Type) string

var NewTypeAlias func(name, typeDecl string) Type

var NewObjectType func(name, typeDecl string, creators ...DispatchFunction) ObjectType

var NewTypeSet func(name, typeDecl string) TypeSet

var NewError func(c Context, message, kind, issueCode string, partialResult Value, details OrderedMap) ErrorObject

var ErrorFromReported func(c Context, err issue.Reported) ErrorObject

var WrapType func(c Context, rt reflect.Type) Type

func getPrefix(pfx interface{}) string {
	name := ``
	if s, ok := pfx.(string); ok {
		name = s
	} else if f, ok := pfx.(func() string); ok {
		name = f()
	}
	return name
}
