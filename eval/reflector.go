package eval

import (
	"reflect"
	"github.com/puppetlabs/go-semver/semver"
)

// A PReflectedType is implemented by PTypes that can have a potential to
// present themselves as a reflect.Type
type PReflectedType interface {
	PType

	// ReflectType returns the reflect.Type that corresponds to the receiver
	// if possible
	ReflectType() (reflect.Type, bool)
}

// A PReflected is a value that can reflect itself into a given reflect.Value
type PReflected interface {
	Reflect(c Context) reflect.Value

	ReflectTo(c Context, value reflect.Value)
}

// An ImplementationRegistry contains mappings between ObjectType and reflect.Type
type ImplementationRegistry interface {
	// RegisterType registers the mapping between the given PType and reflect.Type
	RegisterType(c Context, t ObjectType, r reflect.Type)

	// RegisterType2 registers the mapping between the given name of a PType and reflect.Type
	RegisterType2(c Context, tn string, r reflect.Type)

	// PTypeToReflected returns the reflect.Type for the given PType
	PTypeToReflected(t ObjectType) (reflect.Type, bool)

	// ReflectedToPtype returns the PType for the given reflect.Type
	ReflectedToPtype(t reflect.Type) (ObjectType, bool)
}

// A Reflector deals with conversions between PValue and reflect.Value and
// between PType and reflect.Type
type Reflector interface {
	// FieldName returns the puppet name for the given field. The puppet name is
	// either picked from the 'puppet' tag of the field or the result of
	// munging the field name through utils.CamelToSnakeCase
	FieldName(f *reflect.StructField) string

	// Reflect returns the reflected value of the native value held
	// by the given src
	Reflect(src PValue, rt reflect.Type) reflect.Value

	// ReflectTo assigns the native value of src to dest
	ReflectTo(src PValue, dest reflect.Value)

	// ReflectType returns the reflected type of the given PType if possible. Only
	// PTypes that represent a value can be represented as a reflected type. Types
	// like Any, Default, Unit, or Variant have no reflected type representation
	ReflectType(src PType) (reflect.Type, bool)

	// ObjectTypeFromReflect creates an Object type based on the given reflected type
	// which has to be a struct or a pointer to a struct
	ObjectTypeFromReflect(typeName string, parent PType, structType reflect.Type) ObjectType

	// TypeSetFromReflect creates a TypeSet based on the given reflected types which have
	// to be structs or pointer to structs
	TypeSetFromReflect(typeSetName string, version semver.Version, structTypes ...reflect.Type) TypeSet

	// TagHash returns the parsed and evaluated hash from the 'puppet' tag
	TagHash(f *reflect.StructField) (KeyedValue, bool)
}
