package values

import (
	"bytes"
	. "io"
	"reflect"

	. "github.com/puppetlabs/go-evaluator/eval/errors"
	. "github.com/puppetlabs/go-evaluator/eval/values/api"
)

const (
	NO_STRING = "\x00"
)

// isInstance answers if value is an instance of the given puppeType
func isInstance(puppetType PType, value PValue) bool {
	return GuardedIsInstance(puppetType, value, nil)
}

// isAssignable answers if t is assignable to this type
func isAssignable(puppetType PType, other PType) bool {
	return GuardedIsAssignable(puppetType, other, nil)
}

func generalize(a PType) PType {
	if g, ok := a.(Generalizable); ok {
		return g.Generic()
	}
	return a
}

func GuardedIsInstance(a PType, v PValue, g Guard) bool {
	return a.IsInstance(v, g)
}

func GuardedIsAssignable(a PType, b PType, g Guard) bool {
	switch b.(type) {
	case nil:
		return false
	case *UnitType:
		return true
	case *NotUndefType:
		nt := b.(*NotUndefType).typ
		if !GuardedIsAssignable(nt, undefType_DEFAULT, g) {
			return GuardedIsAssignable(a, nt, g)
		}
		return false
	case *OptionalType:
		if GuardedIsAssignable(a, undefType_DEFAULT, g) {
			ot := b.(*OptionalType).typ
			return ot == nil || GuardedIsAssignable(a, ot, g)
		}
		return false
	case *TypeAliasType:
		return GuardedIsAssignable(a, b.(*TypeAliasType).resolvedType, g)
	case *VariantType:
		return b.(*VariantType).allAssignableTo(a, g)
	}
	return a.IsAssignable(b, g)
}

func UniqueTypes(types []PType) []PType {
	top := len(types)
	if top < 2 {
		return types
	}

	result := make([]PType, 0, top)
	exists := make(map[HashKey]bool, top)
	for _, t := range types {
		key := ToKey(t)
		if !exists[key] {
			exists[key] = true
			result = append(result, t)
		}
	}
	return result
}

// Convert a slice of values that implement the PValue interface to []PValue. The
// method will panic if the given argument is not a slice or array, or if not all
// elements implement the PValue interface
func ValueSlice(slice interface{}) []PValue {
	sv := reflect.ValueOf(slice)
	top := sv.Len()
	result := make([]PValue, top)
	for idx := 0; idx < top; idx++ {
		result[idx] = sv.Index(idx).Interface().(PValue)
	}
	return result
}

func UniqueValues(values []PValue) []PValue {
	top := len(values)
	if top < 2 {
		return values
	}

	result := make([]PValue, 0, top)
	exists := make(map[HashKey]bool, top)
	for _, v := range values {
		key := ToKey(v)
		if !exists[key] {
			exists[key] = true
			result = append(result, v)
		}
	}
	return result
}

func NewIllegalArgumentType2(name string, index int, expected string, actual PValue) InstantiationError {
	return NewIllegalArgumentType(name, index, expected, DetailedType(actual).String())
}

func TypeToString(t PType, b Writer, s FormatContext, g RDetect) {
	f := GetFormat(s.FormatMap(), t.Type())
	switch f.FormatChar() {
	case 's', 'p':
		quoted := f.IsAlt() && f.FormatChar() == 's'
		if quoted || f.HasStringFlags() {
			bld := bytes.NewBufferString(``)
			basicTypeToString(t, bld, s, g)
			f.ApplyStringFlags(b, bld.String(), quoted)
		} else {
			basicTypeToString(t, b, s, g)
		}
	default:
		panic(s.UnsupportedFormat(t.Type(), `sp`, f))
	}
}

func basicTypeToString(t PType, b Writer, s FormatContext, g RDetect) {
	WriteString(b, t.Name())
	if pt, ok := t.(ParameterizedType); ok {
		params := pt.Parameters()
		if len(params) > 0 {
			WrapArray(params).ToString(b, s, g)
		}
	}
}

type alterFunc func(t PType) PType

func alterTypes(types []PType, function alterFunc) []PType {
	al := make([]PType, len(types))
	for idx, t := range types {
		al[idx] = function(t)
	}
	return al
}

func toTypes(types ...PValue) ([]PType, int) {
	top := len(types)
	if top == 1 {
		if a, ok := types[0].(IndexedValue); ok {
			return toTypes(a.Elements()...)
		}
	}
	result := make([]PType, top)
	for idx, t := range types {
		if pt, ok := t.(PType); ok {
			result[idx] = pt
			continue
		}
		return nil, idx
	}
	return result, -1
}

func DefaultDataType() *TypeAliasType {
	return dataType_DEFAULT
}

func DefaultRichDataType() *TypeAliasType {
	return richDataType_DEFAULT
}

var dataArrayType_DEFAULT = &ArrayType{integerType_POSITIVE, &TypeReferenceType{`Data`}}
var dataHashType_DEFAULT = &HashType{integerType_POSITIVE, stringType_DEFAULT, &TypeReferenceType{`Data`}}
var dataType_DEFAULT = &TypeAliasType{name: `Data`, resolvedType: &VariantType{[]PType{scalarDataType_DEFAULT, undefType_DEFAULT, dataArrayType_DEFAULT, dataHashType_DEFAULT}}}

var richKeyType_DEFAULT = &VariantType{[]PType{stringType_DEFAULT, numericType_DEFAULT}}
var richDataArrayType_DEFAULT = &ArrayType{integerType_POSITIVE, &TypeReferenceType{`RichData`}}
var richDataHashType_DEFAULT = &HashType{integerType_POSITIVE, richKeyType_DEFAULT, &TypeReferenceType{`RichData`}}
var richDataType_DEFAULT = &TypeAliasType{`RichData`, nil, &VariantType{
	[]PType{scalarType_DEFAULT, binaryType_DEFAULT, defaultType_DEFAULT, typeType_DEFAULT, undefType_DEFAULT, richDataArrayType_DEFAULT, richDataHashType_DEFAULT}}, nil}

func init() {
	// "resolve" the dataType and richDataType
	dataArrayType_DEFAULT.typ = dataType_DEFAULT
	dataHashType_DEFAULT.valueType = dataType_DEFAULT
	richDataArrayType_DEFAULT.typ = richDataType_DEFAULT
	richDataHashType_DEFAULT.valueType = richDataType_DEFAULT

	Generalize = generalize
	IsAssignable = isAssignable
	IsInstance = isInstance

	DetailedType = func(value PValue) PType {
		if dt, ok := value.(DetailedTypeValue); ok {
			return dt.DetailedType()
		}
		return value.Type()
	}

	GenericType = func(value PValue) PType {
		t := value.Type()
		if g, ok := t.(Generalizable); ok {
			return g.Generic()
		}
		return t
	}

	ToArray = func(elements []PValue) IndexedValue {
		return WrapArray(elements)
	}

	ToKey = func(value PValue) HashKey {
		if hk, ok := value.(HashKeyValue); ok {
			return hk.ToKey()
		}
		if pt, ok := value.(PType); ok {
			return HashKey("\x01T" + pt.String())
		}
		panic(NewIllegalArgumentType2(`ToKey`, 0, `value usable as hash key`, value))
	}

	IsTruthy = func(tv PValue) bool {
		switch tv.(type) {
		case *UndefValue:
			return false
		case *BooleanValue:
			return tv.(*BooleanValue).value
		default:
			return true
		}
	}
}
