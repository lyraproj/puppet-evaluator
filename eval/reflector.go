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
	// RegisterType registers the mapping between the given Type name and reflect.Type
	RegisterType(c Context, t string, r reflect.Type)

	// TypeToReflected returns the reflect.Type for the given Type name
	TypeToReflected(t string) (reflect.Type, bool)

	// ReflectedToType returns the Type name for the given reflect.Type
	ReflectedToType(t reflect.Type) (string, bool)

	// ReflectedNameToType returns the PCore Type name for the given Go Type name
	ReflectedNameToType(name string) (string, bool)
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

	// ObjectTypeFromReflect creates an Object type based on the given reflected type
	ObjectTypeFromReflect(typeName string, parent Type, rType reflect.Type) ObjectType

	// TypeSetFromReflect creates a TypeSet based on the given reflected types
	TypeSetFromReflect(typeSetName string, version semver.Version, rTypes ...reflect.Type) TypeSet

	// TagHash returns the parsed and evaluated hash from the 'puppet' tag
	TagHash(f *reflect.StructField) (OrderedMap, bool)

	// TypeName returns the name of the type. The name is either obtained from the ImplementationRegistry
	// or created by concatenating prefix with the name of the given reflected type. If noPrefixOverride
	// is true, then it is an error when a name found in the ImplementationRegistry isn't prefixed with
	// the given prefix.
	TypeName(prefix string, t reflect.Type, noPrefixOverride bool) string

	// Fields returns all fields of the given reflected type.
	Fields(t reflect.Type) []reflect.StructField

	// Methods returns all methods of the given reflected type.
	Methods(ptr reflect.Type) []reflect.Method
}
