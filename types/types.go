package types

import (
	"bytes"
	"io"
	. "reflect"

	. "github.com/puppetlabs/go-evaluator/errors"
	. "github.com/puppetlabs/go-evaluator/evaluator"
	"github.com/puppetlabs/go-evaluator/semver"
	"time"
	"regexp"
	. "github.com/puppetlabs/go-parser/parser"
	"runtime"
	"github.com/puppetlabs/go-parser/issue"
	"fmt"
	"sync"
	"encoding/json"
)

const (
	NO_STRING = "\x00"

	HK_BINARY        = byte('B')
	HK_BOOLEAN       = byte('b')
	HK_DEFAULT       = byte('d')
	HK_FLOAT         = byte('f')
	HK_INTEGER       = byte('i')
	HK_REGEXP        = byte('r')
	HK_TIMESPAN      = byte('D')
	HK_TIMESTAMP     = byte('T')
	HK_TYPE          = byte('t')
	HK_UNDEF         = byte('u')
	HK_URI           = byte('U')
	HK_VERSION       = byte('v')
	HK_VERSION_RANGE = byte('R')
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
	if g, ok := a.(ParameterizedType); ok {
		return g.Default()
	}
	return a
}

func defaultFor(t PType) PType {
	if g, ok := t.(ParameterizedType); ok {
		return g.Default()
	}
	return t
}

func normalize(t PType) PType {
	// TODO: Implement for ParameterizedType
	return t
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
	sv := ValueOf(slice)
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
	return NewIllegalArgumentType(name, index, expected, DetailedValueType(actual).String())
}

func TypeToString(t PType, b io.Writer, s FormatContext, g RDetect) {
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

func basicTypeToString(t PType, b io.Writer, s FormatContext, g RDetect) {
	io.WriteString(b, t.Name())
	if s != EXPANDED {
		switch t.(type) {
		case *objectType, *TypeAliasType, *TypeSetType:
			return
		}
	}
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

func NilAs(dflt, t PType) PType {
	if t == nil {
		t = dflt
	}
	return t
}

func CopyAppend(types []PType, t PType) []PType {
	top := len(types)
	tc := make([]PType, top+1, top+1)
	copy(tc, types)
	tc[top] = t
	return tc
}

var dataArrayType_DEFAULT = &ArrayType{integerType_POSITIVE, &TypeReferenceType{`Data`}}
var dataHashType_DEFAULT = &HashType{integerType_POSITIVE, stringType_DEFAULT, &TypeReferenceType{`Data`}}
var dataType_DEFAULT = &TypeAliasType{name: `Data`, resolvedType: &VariantType{[]PType{scalarDataType_DEFAULT, undefType_DEFAULT, dataArrayType_DEFAULT, dataHashType_DEFAULT}}}

var richKeyType_DEFAULT = &VariantType{[]PType{stringType_DEFAULT, numericType_DEFAULT}}
var richDataArrayType_DEFAULT = &ArrayType{integerType_POSITIVE, &TypeReferenceType{`RichData`}}
var richDataHashType_DEFAULT = &HashType{integerType_POSITIVE, richKeyType_DEFAULT, &TypeReferenceType{`RichData`}}
var richDataType_DEFAULT = &TypeAliasType{`RichData`, nil, &VariantType{
	[]PType{scalarType_DEFAULT, binaryType_DEFAULT, defaultType_DEFAULT, typeType_DEFAULT, undefType_DEFAULT, richDataArrayType_DEFAULT, richDataHashType_DEFAULT}}, nil}

var resolvableTypes = make([]ResolvableType, 0, 16)
var resolvableTypesLock sync.Mutex

func init() {
	// "resolve" the dataType and richDataType
	dataArrayType_DEFAULT.typ = dataType_DEFAULT
	dataHashType_DEFAULT.valueType = dataType_DEFAULT
	richDataArrayType_DEFAULT.typ = richDataType_DEFAULT
	richDataHashType_DEFAULT.valueType = richDataType_DEFAULT

	DefaultFor = defaultFor
	Generalize = generalize
	Normalize = normalize
	IsAssignable = isAssignable
	IsInstance = isInstance

	DetailedValueType = func(value PValue) PType {
		if dt, ok := value.(DetailedTypeValue); ok {
			return dt.DetailedType()
		}
		return value.Type()
	}

	GenericType = func(t PType) PType {
		if g, ok := t.(Generalizable); ok {
			return g.Generic()
		}
		return t
	}

	GenericValueType = func(value PValue) PType {
		return GenericType(value.Type())
	}

	ToArray = func(elements []PValue) IndexedValue {
		return WrapArray(elements)
	}

	ToKey = func(value PValue) HashKey {
		if hk, ok := value.(HashKeyValue); ok {
			return hk.ToKey()
		}
		b := bytes.NewBuffer([]byte{})
		appendKey(b, value)
		return HashKey(b.String())
	}

	IsTruthy = func(tv PValue) bool {
		switch tv.(type) {
		case *UndefValue:
			return false
		case *BooleanValue:
			return tv.(*BooleanValue).Bool()
		default:
			return true
		}
	}

	RegisterResolvableType = registerResolvableType

	WrapUnknown = wrap
}

func PopDeclaredTypes() (types []ResolvableType) {
	resolvableTypesLock.Lock()
	types = resolvableTypes
	if len(types) > 0 {
		resolvableTypes = make([]ResolvableType, 0, 16)
	}
	resolvableTypesLock.Unlock()
	return
}

func registerResolvableType(tp ResolvableType) {
	resolvableTypesLock.Lock()
	resolvableTypes = append(resolvableTypes, tp)
	resolvableTypesLock.Unlock()
}

func appendKey(b *bytes.Buffer, v PValue) {
	if hk, ok := v.(StreamHashKeyValue); ok {
		hk.ToKey(b)
	} else if pt, ok := v.(PType); ok {
		b.WriteByte(1)
		b.WriteByte(HK_TYPE)
		b.Write([]byte(pt.Name()))
		if ppt, ok := pt.(ParameterizedType); ok {
			for _, p := range ppt.Parameters() {
				appendKey(b, p)
			}
		}
	} else if hk, ok := v.(HashKeyValue); ok {
		b.Write([]byte(hk.ToKey()))
	} else {
		panic(NewIllegalArgumentType2(`ToKey`, 0, `value used as hash key`, v))
	}
}

func wrap(v interface{}) (pv PValue) {
	switch v.(type) {
	case nil:
		pv = _UNDEF
	case PValue:
		pv = v.(PValue)
	case string:
		pv = WrapString(v.(string))
	case int64:
		pv = WrapInteger(v.(int64))
	case int:
		pv = WrapInteger(int64(v.(int)))
	case float64:
		pv = WrapFloat(v.(float64))
	case bool:
		pv = WrapBoolean(v.(bool))
	case *regexp.Regexp:
		pv = WrapRegexp2(v.(*regexp.Regexp))
	case []byte:
		pv = WrapBinary(v.([]byte))
	case *semver.Version:
		pv = WrapSemVer(v.(*semver.Version))
	case *semver.VersionRange:
		pv = WrapSemVerRange(v.(*semver.VersionRange))
	case time.Duration:
		pv = WrapTimespan(v.(time.Duration))
	case time.Time:
		pv = WrapTimestamp(v.(time.Time))
	case []PValue:
		pv = WrapArray(v.([]PValue))
	case map[string]interface{}:
		pv = WrapHash4(v.(map[string]interface{}))
	case map[string]PValue:
		pv = WrapHash3(v.(map[string]PValue))
	case json.Number:
		if i, err := v.(json.Number).Int64(); err == nil {
			pv = WrapInteger(i)
		} else {
			f, _ := v.(json.Number).Float64()
			pv = WrapFloat(f)
		}
	default:
		// Can still be an alias, slice, or map in which case reflection conversion will work
		pv = wrapValue(ValueOf(v))
	}
	return pv
}

func wrapValue(vr Value) (pv PValue) {
	switch vr.Kind() {
	case String:
		pv = WrapString(vr.String())
	case Int64, Int32, Int16, Int8:
		pv = WrapInteger(vr.Int())
	case Uint, Uint64, Uint32, Uint16, Uint8:
		pv = WrapInteger(int64(vr.Uint())) // Possible loss for very large numbers
	case Bool:
		pv = WrapBoolean(vr.Bool())
	case Float64, Float32:
		pv = WrapFloat(vr.Float())
	case Slice, Array:
		top := vr.Len()
		els := make([]PValue, top)
		for i := 0; i < top; i++ {
			els[i] = wrap(interfaceOrNil(vr.Index(i)))
		}
		pv = WrapArray(els)
	case Map:
		keys := vr.MapKeys()
		els := make([]*HashEntry, len(keys))
		for i, k := range keys {
			els[i] = WrapHashEntry(wrap(interfaceOrNil(k)), wrap(interfaceOrNil(vr.MapIndex(k))))
		}
		pv = WrapHash(els)
	default:
		if vr.CanInterface() {
			pv = WrapRuntime(vr.Interface())
		} else {
			pv = _UNDEF
		}
	}
	return pv
}

func interfaceOrNil(vr Value) interface{} {
	if vr.CanInterface() {
		return vr.Interface()
	}
	return nil
}

func init() {
	NewType = newType
}

func newType(name, typeDecl string) PType {
	p := CreateParser()
	_, fileName, fileLine, _ := runtime.Caller(1)
	expr, err := p.Parse(fileName, fmt.Sprintf(`type %s = %s`, name, typeDecl), false, true)
	if err != nil {
		err = convertReportedIssue(err, fileName, fileLine)
		panic(err)
	}

	if ta, ok := expr.(*TypeAlias); ok {
		rt, _ := CreateTypeDefinition(ta, RUNTIME_NAME_AUTHORITY)
		registerResolvableType(rt.(ResolvableType))
		return rt.(PType)
	}
	panic(convertReportedIssue(Error2(expr, EVAL_NO_DEFINITION, issue.H{`source`: ``, `type`: TYPE, `name`: name}), fileName, fileLine))
}

func convertReportedIssue(err error, fileName string, lineOffset int) error {
	if ri, ok := err.(*issue.ReportedIssue); ok {
		return ri.OffsetByLocation(issue.NewLocation(fileName, lineOffset, 0))
	}
	return err
}

func CreateTypeDefinition(d Definition, na URI) (interface{}, TypedName) {
	switch d.(type) {
	case *TypeAlias:
		taExpr := d.(*TypeAlias)
		name := taExpr.Name()
		body := taExpr.Type()
		tn := NewTypedName2(TYPE, name, na)
		var ta PType
		switch body.(type) {
		case *QualifiedReference:
			ta = NewTypeAliasType(name, body, nil)
		case *AccessExpression:
			ta = nil
			ae := body.(*AccessExpression)
			if len(ae.Keys()) == 1 {
				arg := ae.Keys()[0]
				if hash, ok := arg.(*LiteralHash); ok {
					if lq, ok := ae.Operand().(*QualifiedReference); ok {
						if lq.Name() == `Object` || lq.Name() == `TypeSet` {
							ta = createMetaType(name, lq.Name(), hash)
						}
					}
				}
			}
			if ta == nil {
				ta = NewTypeAliasType(name, body, nil)
			}
		case *LiteralHash:
			ta = createMetaType(name, ``, body.(*LiteralHash))
		case *LiteralList:
			ll := body.(*LiteralList)
			if len(ll.Elements()) == 1 {
				if hash, ok := ll.Elements()[0].(*LiteralHash); ok {
					ta = createMetaType(name, ``, hash)
				}
			}
		case *KeyedEntry:
			ke := body.(*KeyedEntry)
			if pn, ok := ke.Key().(*QualifiedReference); ok {
				if hash, ok := ke.Value().(*LiteralHash); ok {
					ta = createMetaType(name, pn.Name(), hash)
				}
			}
		}

		if ta == nil {
			panic(fmt.Sprintf(`cannot create object from a %T`, body))
		}
		return ta, tn
	default:
		panic(fmt.Sprintf(`Don't know how to define a %T`, d))
	}
}

func createMetaType(name string, parentName string, hash *LiteralHash) PType {
	if parentName == `` || parentName == `Object` {
		return NewObjectType(name, nil, hash)
	} // TODO else if lq.Name() == `TypeSet`

	return NewObjectType(name, NewTypeReferenceType(parentName), hash)
}
