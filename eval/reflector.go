package eval

import (
	"github.com/lyraproj/semver/semver"
	"reflect"
)

// A ReflectedType is implemented by PTypes that can have a potential to
// present themselves as a reflect.Type
type ReflectedType interface {
	Type

	// ReflectType returns the reflect.Type that corresponds to the receiver
	// if possible
	ReflectType(c Context) (reflect.Type, bool)
}

// A Reflected is a value that can reflect itself into a given reflect.Value
type Reflected interface {
	Reflect(c Context) reflect.Value

	ReflectTo(c Context, value reflect.Value)
}

// An ImplementationRegistry contains mappings between ObjectType and reflect.Type
type ImplementationRegistry interface {
	// RegisterType registers the mapping between the given Type and reflect.Type
	RegisterType(c Context, t Type, r reflect.Type)

	// TypeToReflected returns the reflect.Type for the given Type
	TypeToReflected(t Type) (reflect.Type, bool)

	// ReflectedToType returns the Type name for the given reflect.Type
	ReflectedToType(t reflect.Type) (Type, bool)

	// ReflectedNameToType returns the Type for the given Go Type name
	ReflectedNameToType(name string) (Type, bool)
}

// A AnnotatedType represenst a reflect.Type with fields that may have 'puppet' tag overrides.
type AnnotatedType interface {
	// Type returns the reflect.Type
	Type() reflect.Type

	// Annotations returns a map of annotations where each key is an Annotation type and the
	// associated value is an instance of Init[<AnnotationType>].
	Annotations() OrderedMap

	// Tags returns a map, keyed by field names, containing values that are the
	// 'puppet' tag parsed into an OrderedMap. The map is merged with possible
	// overrides given when the AnnotatedType instance was created
	Tags(c Context) map[string]OrderedMap
}

// NewTaggedType returns a new instance of a AnnotatedType
var NewTaggedType func(reflect.Type, map[string]string) AnnotatedType

// NewAnnotatedType returns a new instance of a AnnotatedType that is annotated
var NewAnnotatedType func(reflect.Type, map[string]string, OrderedMap) AnnotatedType

// A Reflector deals with conversions between Value and reflect.Value and
// between Type and reflect.Type
type Reflector interface {
	// FieldName returns the puppet name for the given field. The puppet name is
	// either picked from the 'puppet' tag of the field or the result of
	// munging the field name through utils.CamelToSnakeCase
	FieldName(f *reflect.StructField) string

	// FunctionDeclFromReflect creates a function declaration suitable for inclusion in an ObjectType initialization
	// hash.
	FunctionDeclFromReflect(name string, mt reflect.Type, fromInterface bool) OrderedMap

	// Reflect returns the reflected value of the native value held
	// by the given src
	Reflect(src Value) reflect.Value

	// Reflect2 returns the reflected value of given type from the native value held
	// by the given src
	Reflect2(src Value, rt reflect.Type) reflect.Value

	// ReflectFieldTags reflects the name, type, and value from a reflect.StructField
	// using the field tags and the optionally given puppetTag
	ReflectFieldTags(f *reflect.StructField, puppetTag OrderedMap) (name string, decl OrderedMap)

	// ReflectTo assigns the native value of src to dest
	ReflectTo(src Value, dest reflect.Value)

	// ReflectType returns the reflected type of the given Type if possible. Only
	// PTypes that represent a value can be represented as a reflected type. Types
	// like Any, Default, Unit, or Variant have no reflected type representation
	ReflectType(src Type) (reflect.Type, bool)

	// InitializerFromTagged creates an Object initializer hash based on the given reflected type.
	InitializerFromTagged(typeName string, parent Type, rType AnnotatedType) OrderedMap

	// TypeFromReflect creates an ObjectType based on the given reflected type.
	// The new type is automatically added to the ImplementationRegistry registered to
	// the Context from where the Reflector was obtained.
	TypeFromReflect(typeName string, parent Type, rType reflect.Type) ObjectType

	// TypeFromTagged creates an Object type based on the given reflected type.
	// The new type is automatically added to the ImplementationRegistry registered to
	// the Context from where the Reflector was obtained.
	TypeFromTagged(typeName string, parent Type, rType AnnotatedType) ObjectType

	// TypeSetFromReflect creates a TypeSet based on the given reflected types The new types are automatically
	// added to the ImplementationRegistry registered to the Context from where the Reflector was obtained.
	// The aliases map maps the names of the reflected types to the unqualified name of the created types.
	// The aliases map may be nil and if present, may map only a subset of the reflected type names
	TypeSetFromReflect(typeSetName string, version semver.Version, aliases map[string]string, rTypes ...reflect.Type) TypeSet

	// TagHash returns the parsed and evaluated hash from the 'puppet' tag
	TagHash(f *reflect.StructField) (OrderedMap, bool)

	// Fields returns all fields of the given reflected type or an empty slice if no fields exists.
	Fields(t reflect.Type) []reflect.StructField

	// Methods returns all methods of the given reflected type or an empty slice if no methods exists.
	Methods(ptr reflect.Type) []reflect.Method
}
