package eval

import (
	"reflect"
	"github.com/puppetlabs/go-semver/semver"
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

// A Reflector deals with conversions between Value and reflect.Value and
// between Type and reflect.Type
type Reflector interface {
	// FieldName returns the puppet name for the given field. The puppet name is
	// either picked from the 'puppet' tag of the field or the result of
	// munging the field name through utils.CamelToSnakeCase
	FieldName(f *reflect.StructField) string

	// Reflect returns the reflected value of the native value held
	// by the given src
	Reflect(src Value) reflect.Value

	// Reflect2 returns the reflected value of given type from the native value held
	// by the given src
	Reflect2(src Value, rt reflect.Type) reflect.Value

	// ReflectFieldTags reflects the name, type, and value from a reflect.StructField
	// using the 'puppet' tag.
	ReflectFieldTags(f *reflect.StructField)  (name string, decl OrderedMap)

	// ReflectTo assigns the native value of src to dest
	ReflectTo(src Value, dest reflect.Value)

	// ReflectType returns the reflected type of the given Type if possible. Only
	// PTypes that represent a value can be represented as a reflected type. Types
	// like Any, Default, Unit, or Variant have no reflected type representation
	ReflectType(src Type) (reflect.Type, bool)

	// ObjectTypeFromReflect creates an Object type based on the given reflected type.
	// The new type is automatically added to the ImplementationRegistry registered to
	// the Context from where the Reflector was obtained.
	ObjectTypeFromReflect(typeName string, parent Type, rType reflect.Type) ObjectType

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
