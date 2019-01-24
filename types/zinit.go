package types

import (
	"github.com/lyraproj/puppet-evaluator/eval"
	"reflect"
	"time"
)

// This init function must be run last in the type package. Hence the file name
func init() {
	primitivePTypes = map[reflect.Kind]eval.Type{
		reflect.String:  DefaultStringType(),
		reflect.Int:     DefaultIntegerType(),
		reflect.Int8:    integerType8,
		reflect.Int16:   integerType16,
		reflect.Int32:   integerType32,
		reflect.Int64:   DefaultIntegerType(),
		reflect.Uint:    integerTypeU64,
		reflect.Uint8:   integerTypeU8,
		reflect.Uint16:  integerTypeU16,
		reflect.Uint32:  integerTypeU32,
		reflect.Uint64:  integerTypeU64,
		reflect.Float32: floatType32,
		reflect.Float64: DefaultFloatType(),
		reflect.Bool:    DefaultBooleanType(),
	}

	wellknowns = map[reflect.Type]eval.Type{
		reflect.TypeOf(&ArrayValue{}):                    DefaultArrayType(),
		reflect.TypeOf((*eval.List)(nil)).Elem():         DefaultArrayType(),
		reflect.TypeOf(&BinaryValue{}):                   DefaultBinaryType(),
		reflect.TypeOf(floatValue(0.0)):                  DefaultFloatType(),
		reflect.TypeOf((*eval.FloatValue)(nil)).Elem():   DefaultFloatType(),
		reflect.TypeOf(&HashValue{}):                     DefaultHashType(),
		reflect.TypeOf((*eval.OrderedMap)(nil)).Elem():   DefaultHashType(),
		reflect.TypeOf(integerValue(0)):                  DefaultIntegerType(),
		reflect.TypeOf((*eval.IntegerValue)(nil)).Elem(): DefaultIntegerType(),
		reflect.TypeOf(&RegexpValue{}):                   DefaultRegexpType(),
		reflect.TypeOf(&SemVerValue{}):                   DefaultSemVerType(),
		reflect.TypeOf(&SensitiveValue{}):                DefaultSensitiveType(),
		reflect.TypeOf(stringValue(``)):                  DefaultStringType(),
		reflect.TypeOf((*eval.StringValue)(nil)).Elem():  DefaultStringType(),
		reflect.TypeOf(TimespanValue(0)):                 DefaultTimespanType(),
		reflect.TypeOf(time.Duration(0)):                 DefaultTimespanType(),
		reflect.TypeOf(time.Time{}):                      DefaultTimestampType(),
		reflect.TypeOf(&TimestampValue{}):                DefaultTimestampType(),
		evalValueType:                                    DefaultAnyType(),
		reflect.TypeOf((*eval.PuppetObject)(nil)).Elem(): DefaultObjectType(),
		reflect.TypeOf((*eval.Object)(nil)).Elem():       DefaultObjectType(),
		evalObjectTypeType:                               ObjectMetaType,
		reflect.TypeOf(&TypeType{}):                      DefaultTypeType(),
		evalTypeType:                                     DefaultTypeType(),
		reflect.TypeOf(&typeSet{}):                       TypeSetMetaType,
		evalTypeSetType:                                  TypeSetMetaType,
		reflect.TypeOf((*eval.TypedName)(nil)).Elem():    TypedName_Type,
		reflect.TypeOf(&UndefValue{}):                    DefaultUndefType(),
		reflect.TypeOf(&UriValue{}):                      DefaultUriType(),
	}
}
