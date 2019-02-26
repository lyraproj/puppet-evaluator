package types

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/puppet-evaluator/errors"
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-parser/parser"
	"github.com/lyraproj/semver/semver"
	"strings"
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

	INTEGER_HEX = `(?:0[xX][0-9A-Fa-f]+)`
	INTEGER_OCT = `(?:0[0-7]+)`
	INTEGER_BIN = `(?:0[bB][01]+)`
	INTEGER_DEC = `(?:0|[1-9]\d*)`
	SIGN_PREFIX = `[+-]?\s*`

	OPTIONAL_FRACTION = `(?:\.\d+)?`
	OPTIONAL_EXPONENT = `(?:[eE]-?\d+)?`
	FLOAT_DEC         = `(?:` + INTEGER_DEC + OPTIONAL_FRACTION + OPTIONAL_EXPONENT + `)`

	INTEGER_PATTERN = `\A` + SIGN_PREFIX + `(?:` + INTEGER_DEC + `|` + INTEGER_HEX + `|` + INTEGER_OCT + `|` + INTEGER_BIN + `)\z`
	FLOAT_PATTERN   = `\A` + SIGN_PREFIX + `(?:` + FLOAT_DEC + `|` + INTEGER_HEX + `|` + INTEGER_OCT + `|` + INTEGER_BIN + `)\z`
)

// isInstance answers if value is an instance of the given puppeType
func isInstance(puppetType eval.Type, value eval.Value) bool {
	return GuardedIsInstance(puppetType, value, nil)
}

// isAssignable answers if t is assignable to this type
func isAssignable(puppetType eval.Type, other eval.Type) bool {
	return GuardedIsAssignable(puppetType, other, nil)
}

func generalize(a eval.Type) eval.Type {
	if g, ok := a.(eval.Generalizable); ok {
		return g.Generic()
	}
	if g, ok := a.(eval.ParameterizedType); ok {
		return g.Default()
	}
	return a
}

func defaultFor(t eval.Type) eval.Type {
	if g, ok := t.(eval.ParameterizedType); ok {
		return g.Default()
	}
	return t
}

func normalize(t eval.Type) eval.Type {
	// TODO: Implement for ParameterizedType
	return t
}

func resolve(c eval.Context, t eval.Type) eval.Type {
	if rt, ok := t.(eval.ResolvableType); ok {
		return rt.Resolve(c)
	}
	return t
}

func EachCoreType(fc func(t eval.Type)) {
	keys := make([]string, len(coreTypes))
	i := 0
	for key := range coreTypes {
		keys[i] = key
		i++
	}
	sort.Strings(keys)
	for _, key := range keys {
		fc(coreTypes[key])
	}
}

func GuardedIsInstance(a eval.Type, v eval.Value, g eval.Guard) bool {
	return a.IsInstance(v, g)
}

func GuardedIsAssignable(a eval.Type, b eval.Type, g eval.Guard) bool {
	if a == b || a == anyType_DEFAULT {
		return true
	}
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

func UniqueTypes(types []eval.Type) []eval.Type {
	top := len(types)
	if top < 2 {
		return types
	}

	result := make([]eval.Type, 0, top)
	exists := make(map[eval.HashKey]bool, top)
	for _, t := range types {
		key := eval.ToKey(t)
		if !exists[key] {
			exists[key] = true
			result = append(result, t)
		}
	}
	return result
}

// ValueSlice convert a slice of values that implement the eval.Value interface to []eval.Value. The
// method will panic if the given argument is not a slice or array, or if not all
// elements implement the eval.Value interface
func ValueSlice(slice interface{}) []eval.Value {
	sv := reflect.ValueOf(slice)
	top := sv.Len()
	result := make([]eval.Value, top)
	for idx := 0; idx < top; idx++ {
		result[idx] = sv.Index(idx).Interface().(eval.Value)
	}
	return result
}

func UniqueValues(values []eval.Value) []eval.Value {
	top := len(values)
	if top < 2 {
		return values
	}

	result := make([]eval.Value, 0, top)
	exists := make(map[eval.HashKey]bool, top)
	for _, v := range values {
		key := eval.ToKey(v)
		if !exists[key] {
			exists[key] = true
			result = append(result, v)
		}
	}
	return result
}

func NewIllegalArgumentType2(name string, index int, expected string, actual eval.Value) errors.InstantiationError {
	return errors.NewIllegalArgumentType(name, index, expected, eval.DetailedValueType(actual).String())
}

func TypeToString(t eval.Type, b io.Writer, s eval.FormatContext, g eval.RDetect) {
	f := eval.GetFormat(s.FormatMap(), t.PType())
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
		panic(s.UnsupportedFormat(t.PType(), `sp`, f))
	}
}

func basicTypeToString(t eval.Type, b io.Writer, s eval.FormatContext, g eval.RDetect) {
	name := t.Name()
	if ex, ok := s.Property(`expanded`); !(ok && ex == `true`) {
		switch t.(type) {
		case *TypeAliasType:
			if ts, ok := s.Property(`typeSet`); ok {
				name = stripTypeSetName(ts, name)
			}
			_, err := io.WriteString(b, name)
			if err != nil {
				panic(err)
			}
			return
		}
	}
	_, err := io.WriteString(b, name)
	if err != nil {
		panic(err)
	}
	if pt, ok := t.(eval.ParameterizedType); ok {
		params := pt.Parameters()
		if len(params) > 0 {
			si := s.Indentation()
			if si.Breaks() {
				// Never break between the type and the start array marker
				s = newFormatContext2(newIndentation(si.IsIndenting(), si.Level()), s.FormatMap(), s.Properties())
			}
			WrapValues(params).ToString(b, s, g)
		}
	}
}

func stripTypeSetName(tsName, name string) string {
	tsName = tsName + `::`
	if strings.HasPrefix(name, tsName) {
		// Strip name and two colons
		return name[len(tsName):]
	}
	return name
}

type alterFunc func(t eval.Type) eval.Type

func alterTypes(types []eval.Type, function alterFunc) []eval.Type {
	al := make([]eval.Type, len(types))
	for idx, t := range types {
		al[idx] = function(t)
	}
	return al
}

func toTypes(types eval.List) ([]eval.Type, int) {
	top := types.Len()
	if top == 1 {
		if a, ok := types.At(0).(eval.List); ok {
			if _, ok = a.(stringValue); !ok {
				ts, f := toTypes(a)
				if f >= 0 {
					return nil, 0
				}
				return ts, 0
			}
		}
	}
	result := make([]eval.Type, 0, top)
	if types.All(func(t eval.Value) bool {
		if pt, ok := t.(eval.Type); ok {
			result = append(result, pt)
			return true
		}
		return false
	}) {
		return result, -1
	}
	return nil, 0
}

func DefaultDataType() *TypeAliasType {
	return dataType_DEFAULT
}

func DefaultRichDataType() *TypeAliasType {
	return richDataType_DEFAULT
}

func NilAs(dflt, t eval.Type) eval.Type {
	if t == nil {
		t = dflt
	}
	return t
}

func CopyAppend(types []eval.Type, t eval.Type) []eval.Type {
	top := len(types)
	tc := make([]eval.Type, top+1, top+1)
	copy(tc, types)
	tc[top] = t
	return tc
}

var dataArrayType_DEFAULT = &ArrayType{IntegerTypePositive, &TypeReferenceType{`Data`}}
var dataHashType_DEFAULT = &HashType{IntegerTypePositive, stringTypeDefault, &TypeReferenceType{`Data`}}
var dataType_DEFAULT = &TypeAliasType{name: `Data`, resolvedType: &VariantType{[]eval.Type{scalarDataType_DEFAULT, undefType_DEFAULT, dataArrayType_DEFAULT, dataHashType_DEFAULT}}}

var richKeyType_DEFAULT = &VariantType{[]eval.Type{stringTypeDefault, numericType_DEFAULT}}
var richDataArrayType_DEFAULT = &ArrayType{IntegerTypePositive, &TypeReferenceType{`RichData`}}
var richDataHashType_DEFAULT = &HashType{IntegerTypePositive, richKeyType_DEFAULT, &TypeReferenceType{`RichData`}}
var richDataType_DEFAULT = &TypeAliasType{`RichData`, nil, &VariantType{
	[]eval.Type{scalarType_DEFAULT,
		binaryType_DEFAULT,
		defaultType_DEFAULT,
		objectType_DEFAULT,
		typeType_DEFAULT,
		typeSetType_DEFAULT,
		undefType_DEFAULT,
		richDataArrayType_DEFAULT,
		richDataHashType_DEFAULT}}, nil}

type Mapping struct {
	T eval.Type
	R reflect.Type
}

var resolvableTypes = make([]eval.ResolvableType, 0, 16)
var resolvableMappings = make([]Mapping, 0, 16)
var resolvableTypesLock sync.Mutex

type BuildFunctionArgs struct {
	Name       string
	LocalTypes eval.LocalTypesCreator
	Creators   []eval.DispatchCreator
}

var constructorsDecls = make([]*BuildFunctionArgs, 0, 16)

func init() {
	// "resolve" the dataType and richDataType
	dataArrayType_DEFAULT.typ = dataType_DEFAULT
	dataHashType_DEFAULT.valueType = dataType_DEFAULT
	richDataArrayType_DEFAULT.typ = richDataType_DEFAULT
	richDataHashType_DEFAULT.valueType = richDataType_DEFAULT

	eval.DefaultFor = defaultFor
	eval.Generalize = generalize
	eval.Normalize = normalize
	eval.IsAssignable = isAssignable
	eval.IsInstance = isInstance
	eval.New = new

	eval.DetailedValueType = func(value eval.Value) eval.Type {
		if dt, ok := value.(eval.DetailedTypeValue); ok {
			return dt.DetailedType()
		}
		return value.PType()
	}

	eval.GenericType = func(t eval.Type) eval.Type {
		if g, ok := t.(eval.Generalizable); ok {
			return g.Generic()
		}
		return t
	}

	eval.GenericValueType = func(value eval.Value) eval.Type {
		return eval.GenericType(value.PType())
	}

	eval.ToArray = func(elements []eval.Value) eval.List {
		return WrapValues(elements)
	}

	eval.ToKey = func(value eval.Value) eval.HashKey {
		if hk, ok := value.(eval.HashKeyValue); ok {
			return hk.ToKey()
		}
		b := bytes.NewBuffer([]byte{})
		appendKey(b, value)
		return eval.HashKey(b.String())
	}

	eval.IsTruthy = func(tv eval.Value) bool {
		switch tv.(type) {
		case *UndefValue:
			return false
		case booleanValue:
			return tv.(booleanValue).Bool()
		default:
			return true
		}
	}

	eval.NewObjectType = newObjectType
	eval.NewGoObjectType = newGoObjectType
	eval.NewTypeAlias = newTypeAlias
	eval.NewTypeSet = newTypeSet
	eval.NewGoType = newGoType
	eval.RegisterResolvableType = registerResolvableType
	eval.NewGoConstructor = newGoConstructor
	eval.NewGoConstructor2 = newGoConstructor2
	eval.Wrap = wrap
	eval.WrapReflected = wrapReflected
	eval.WrapReflectedType = wrapReflectedType
}

func canSerializeAsString(t eval.Type) bool {
	if t == nil {
		// true because nil members will not participate
		return true
	}
	if st, ok := t.(eval.SerializeAsString); ok {
		return st.CanSerializeAsString()
	}
	return false
}

// New creates a new instance of type t
func new(c eval.Context, receiver eval.Value, args ...eval.Value) eval.Value {
	name := ``
	typ, ok := receiver.(eval.Type)
	if ok {
		name = typ.Name()
	} else {
		// Type might be in string form
		_, ok = receiver.(stringValue)
		if !ok {
			// Only types or names of types can be used
			panic(eval.Error(eval.EVAL_INSTANCE_DOES_NOT_RESPOND, issue.H{`type`: receiver.PType(), `message`: `new`}))
		}

		name = receiver.String()
		var t interface{}
		if t, ok = eval.Load(c, NewTypedName(eval.NsType, name)); ok {
			typ = t.(eval.Type)
		}
	}

	if nb, ok := typ.(eval.Newable); ok {
		return nb.New(c, args)
	}

	var ctor eval.Function
	var ct eval.Creatable
	ct, ok = typ.(eval.Creatable)
	if ok {
		ctor = ct.Constructor(c)
	}

	if ctor == nil {
		tn := NewTypedName(eval.NsConstructor, name)
		if t, ok := eval.Load(c, tn); ok {
			ctor = t.(eval.Function)
		}
	}

	if ctor == nil {
		panic(eval.Error(eval.EVAL_INSTANCE_DOES_NOT_RESPOND, issue.H{`type`: name, `message`: `new`}))
	}

	r := ctor.(eval.Function).Call(c, nil, args...)
	if typ != nil {
		eval.AssertInstance(`new`, typ, r)
	}
	return r
}

func newGoConstructor(typeName string, creators ...eval.DispatchCreator) {
	registerGoConstructor(&BuildFunctionArgs{typeName, nil, creators})
}

func newGoConstructor2(typeName string, localTypes eval.LocalTypesCreator, creators ...eval.DispatchCreator) {
	registerGoConstructor(&BuildFunctionArgs{typeName, localTypes, creators})
}

func newGoConstructor3(typeNames []string, localTypes eval.LocalTypesCreator, creators ...eval.DispatchCreator) {
	for _, tn := range typeNames {
		registerGoConstructor(&BuildFunctionArgs{tn, localTypes, creators})
	}
}

func PopDeclaredTypes() (types []eval.ResolvableType) {
	resolvableTypesLock.Lock()
	types = resolvableTypes
	if len(types) > 0 {
		resolvableTypes = make([]eval.ResolvableType, 0, 16)
	}
	resolvableTypesLock.Unlock()
	return
}

func PopDeclaredMappings() (types []Mapping) {
	resolvableTypesLock.Lock()
	types = resolvableMappings
	if len(types) > 0 {
		resolvableMappings = make([]Mapping, 0, 16)
	}
	resolvableTypesLock.Unlock()
	return
}

func PopDeclaredConstructors() (ctorDecls []*BuildFunctionArgs) {
	resolvableTypesLock.Lock()
	ctorDecls = constructorsDecls
	if len(ctorDecls) > 0 {
		constructorsDecls = make([]*BuildFunctionArgs, 0, 16)
	}
	resolvableTypesLock.Unlock()
	return
}

func registerGoConstructor(ctorDecl *BuildFunctionArgs) {
	resolvableTypesLock.Lock()
	constructorsDecls = append(constructorsDecls, ctorDecl)
	resolvableTypesLock.Unlock()
}

func newGoType(name string, zeroValue interface{}) eval.ObjectType {
	t := NewObjectType(name, nil, zeroValue)
	registerResolvableType(t)
	return t
}

func registerResolvableType(tp eval.ResolvableType) {
	resolvableTypesLock.Lock()
	resolvableTypes = append(resolvableTypes, tp)
	resolvableTypesLock.Unlock()
}

func registerMapping(t eval.Type, r reflect.Type) {
	resolvableTypesLock.Lock()
	resolvableMappings = append(resolvableMappings, Mapping{t, r})
	resolvableTypesLock.Unlock()
}

func appendKey(b *bytes.Buffer, v eval.Value) {
	if hk, ok := v.(eval.StreamHashKeyValue); ok {
		hk.ToKey(b)
	} else if pt, ok := v.(eval.Type); ok {
		b.WriteByte(1)
		b.WriteByte(HK_TYPE)
		b.Write([]byte(pt.Name()))
		if ppt, ok := pt.(eval.ParameterizedType); ok {
			for _, p := range ppt.Parameters() {
				appendTypeParamKey(b, p)
			}
		}
	} else if hk, ok := v.(eval.HashKeyValue); ok {
		b.Write([]byte(hk.ToKey()))
	} else {
		panic(NewIllegalArgumentType2(`ToKey`, 0, `value used as hash key`, v))
	}
}

// Special hash key generation for type parameters which might be hashes
// using string keys
func appendTypeParamKey(b *bytes.Buffer, v eval.Value) {
	if h, ok := v.(*HashValue); ok {
		b.WriteByte(2)
		h.EachPair(func(k, v eval.Value) {
			b.Write([]byte(k.String()))
			b.WriteByte(3)
			appendTypeParamKey(b, v)
		})
	} else {
		appendKey(b, v)
	}
}

func wrap(c eval.Context, v interface{}) (pv eval.Value) {
	switch v.(type) {
	case nil:
		pv = _UNDEF
	case eval.Value:
		pv = v.(eval.Value)
	case string:
		pv = stringValue(v.(string))
	case int8:
		pv = integerValue(int64(v.(int8)))
	case int16:
		pv = integerValue(int64(v.(int16)))
	case int32:
		pv = integerValue(int64(v.(int32)))
	case int64:
		pv = integerValue(v.(int64))
	case byte:
		pv = integerValue(int64(v.(byte)))
	case int:
		pv = integerValue(int64(v.(int)))
	case float64:
		pv = floatValue(v.(float64))
	case bool:
		pv = booleanValue(v.(bool))
	case *regexp.Regexp:
		pv = WrapRegexp2(v.(*regexp.Regexp))
	case []byte:
		pv = WrapBinary(v.([]byte))
	case semver.Version:
		pv = WrapSemVer(v.(semver.Version))
	case semver.VersionRange:
		pv = WrapSemVerRange(v.(semver.VersionRange))
	case time.Duration:
		pv = WrapTimespan(v.(time.Duration))
	case time.Time:
		pv = WrapTimestamp(v.(time.Time))
	case []int:
		pv = WrapInts(v.([]int))
	case []string:
		pv = WrapStrings(v.([]string))
	case []eval.Value:
		pv = WrapValues(v.([]eval.Value))
	case []eval.Type:
		pv = WrapTypes(v.([]eval.Type))
	case []interface{}:
		return WrapInterfaces(c, v.([]interface{}))
	case map[string]interface{}:
		pv = WrapStringToInterfaceMap(c, v.(map[string]interface{}))
	case map[string]string:
		pv = WrapStringToStringMap(v.(map[string]string))
	case map[string]eval.Value:
		pv = WrapStringToValueMap(v.(map[string]eval.Value))
	case map[string]eval.Type:
		pv = WrapStringToTypeMap(v.(map[string]eval.Type))
	case json.Number:
		if i, err := v.(json.Number).Int64(); err == nil {
			pv = integerValue(i)
		} else {
			f, _ := v.(json.Number).Float64()
			pv = floatValue(f)
		}
	case reflect.Value:
		pv = wrapReflected(c, v.(reflect.Value))
	case reflect.Type:
		var err error
		if pv, err = wrapReflectedType(c, v.(reflect.Type)); err != nil {
			panic(err)
		}
	default:
		// Can still be an alias, slice, or map in which case reflection conversion will work
		pv = wrapReflected(c, reflect.ValueOf(v))
	}
	return pv
}

func wrapReflected(c eval.Context, vr reflect.Value) (pv eval.Value) {
	if c == nil {
		c = eval.CurrentContext()
	}

	// Invalid shouldn't happen, but needs a check
	if !vr.IsValid() {
		return _UNDEF
	}

	vi := vr

	// Check for nil
	switch vr.Kind() {
	case reflect.Ptr, reflect.Slice, reflect.Array, reflect.Map, reflect.Interface:
		if vr.IsNil() {
			return _UNDEF
		}

		if vi.Kind() == reflect.Interface {
			// Need implementation here.
			vi = vi.Elem()
		}
	}

	if _, ok := wellknowns[vr.Type()]; ok {
		iv := vr.Interface()
		if pv, ok = iv.(eval.Value); ok {
			return
		}
		// A well-known that isn't an eval.Value just yet
		return wrap(c, iv)
	}

	if t, ok := loadFromImplRegistry(c, vi.Type()); ok {
		if pt, ok := t.(eval.ObjectType); ok {
			pv = pt.FromReflectedValue(c, vi)
			return
		}
	}

	pv, ok := WrapPrimitive(vr)
	if ok {
		return pv
	}

	switch vr.Kind() {
	case reflect.Slice, reflect.Array:
		top := vr.Len()
		els := make([]eval.Value, top)
		for i := 0; i < top; i++ {
			els[i] = wrap(c, interfaceOrNil(vr.Index(i)))
		}
		pv = WrapValues(els)
	case reflect.Map:
		keys := vr.MapKeys()
		els := make([]*HashEntry, len(keys))
		for i, k := range keys {
			els[i] = WrapHashEntry(wrap(c, interfaceOrNil(k)), wrap(c, interfaceOrNil(vr.MapIndex(k))))
		}
		pv = sortedMap(els)
	case reflect.Ptr:
		return wrapReflected(c, vr.Elem())
	default:
		if vr.IsValid() && vr.CanInterface() {
			ix := vr.Interface()
			pv, ok = ix.(eval.Value)
			if ok {
				return pv
			}
			pv = WrapRuntime(vr.Interface())
		} else {
			pv = _UNDEF
		}
	}
	return pv
}

func WrapPrimitive(vr reflect.Value) (pv eval.Value, ok bool) {
	ok = true
	switch vr.Kind() {
	case reflect.String:
		pv = stringValue(vr.String())
	case reflect.Int, reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8:
		pv = integerValue(vr.Int())
	case reflect.Uint, reflect.Uint64, reflect.Uint32, reflect.Uint16, reflect.Uint8:
		pv = integerValue(int64(vr.Uint())) // Possible loss for very large numbers
	case reflect.Bool:
		pv = booleanValue(vr.Bool())
	case reflect.Float64, reflect.Float32:
		pv = floatValue(vr.Float())
	default:
		ok = false
	}
	return
}

func loadFromImplRegistry(c eval.Context, vt reflect.Type) (eval.Type, bool) {
	if t, ok := c.ImplementationRegistry().ReflectedToType(vt); ok {
		return t, true
	}
	return nil, false
}

var evalValueType = reflect.TypeOf((*eval.Value)(nil)).Elem()
var evalTypeType = reflect.TypeOf((*eval.Type)(nil)).Elem()
var evalObjectTypeType = reflect.TypeOf((*eval.ObjectType)(nil)).Elem()
var evalTypeSetType = reflect.TypeOf((*eval.TypeSet)(nil)).Elem()

var wellknowns map[reflect.Type]eval.Type

func wrapReflectedType(c eval.Context, vt reflect.Type) (pt eval.Type, err error) {
	if c == nil {
		c = eval.CurrentContext()
	}

	var ok bool
	if pt, ok = wellknowns[vt]; ok {
		return
	}

	kind := vt.Kind()
	if pt, ok = loadFromImplRegistry(c, vt); ok {
		if kind == reflect.Ptr {
			pt = NewOptionalType(pt)
		}
		return
	}

	var t eval.Type
	switch kind {
	case reflect.Slice, reflect.Array:
		if t, err = wrapReflectedType(c, vt.Elem()); err == nil {
			pt = NewArrayType(t, nil)
		}
	case reflect.Map:
		if t, err = wrapReflectedType(c, vt.Key()); err == nil {
			var v eval.Type
			if v, err = wrapReflectedType(c, vt.Elem()); err == nil {
				pt = NewHashType(t, v, nil)
			}
		}
	case reflect.Ptr:
		if t, err = wrapReflectedType(c, vt.Elem()); err == nil {
			pt = NewOptionalType(t)
		}
	default:
		pt, ok = primitivePTypes[vt.Kind()]
		if !ok {
			err = eval.Error(eval.EVAL_UNREFLECTABLE_TYPE, issue.H{`type`: vt.String()})
		}
	}
	return
}

var primitivePTypes map[reflect.Kind]eval.Type

func PrimitivePType(vt reflect.Type) (pt eval.Type, ok bool) {
	pt, ok = primitivePTypes[vt.Kind()]
	return
}

func interfaceOrNil(vr reflect.Value) interface{} {
	if vr.CanInterface() {
		return vr.Interface()
	}
	return nil
}

func newTypeAlias(name, typeDecl string) eval.Type {
	p := parser.CreateParser()
	_, fileName, fileLine, _ := runtime.Caller(1)
	expr, err := p.Parse(fileName, fmt.Sprintf(`type %s = %s`, name, typeDecl), true)
	if err != nil {
		err = convertReported(err, fileName, fileLine)
		panic(err)
	}

	if ta, ok := expr.(*parser.TypeAlias); ok {
		rt, _ := CreateTypeDefinition(ta, eval.RUNTIME_NAME_AUTHORITY)
		at := rt.(*TypeAliasType)
		registerResolvableType(at)
		return at
	}
	panic(convertReported(eval.Error2(expr, eval.EVAL_NO_DEFINITION, issue.H{`source`: ``, `type`: eval.NsType, `name`: name}), fileName, fileLine))
}

func convertReported(err error, fileName string, lineOffset int) error {
	if ri, ok := err.(issue.Reported); ok {
		return ri.OffsetByLocation(issue.NewLocation(fileName, lineOffset, 0))
	}
	return err
}

func CreateTypeDefinition(d parser.Definition, na eval.URI) (interface{}, eval.TypedName) {
	switch d.(type) {
	case *parser.TypeAlias:
		taExpr := d.(*parser.TypeAlias)
		name := taExpr.Name()
		return createTypeDefinition(na, name, taExpr.Type()), eval.NewTypedName2(eval.NsType, name, na)
	default:
		panic(fmt.Sprintf(`Don't know how to define a %T`, d))
	}
}

func createTypeDefinition(na eval.URI, name string, body parser.Expression) eval.Type {
	var ta eval.Type
	switch body.(type) {
	case *parser.QualifiedReference:
		ta = NewTypeAliasType(name, body, nil)
	case *parser.AccessExpression:
		ta = nil
		ae := body.(*parser.AccessExpression)
		if len(ae.Keys()) == 1 {
			arg := ae.Keys()[0]
			if hash, ok := arg.(*parser.LiteralHash); ok {
				if lq, ok := ae.Operand().(*parser.QualifiedReference); ok {
					if lq.Name() == `Object` {
						ta = createMetaType(na, name, lq.Name(), extractParentName(hash), hash)
					} else if lq.Name() == `TypeSet` {
						ta = createMetaType(na, name, lq.Name(), ``, hash)
					}
				}
			}
		}
		if ta == nil {
			ta = NewTypeAliasType(name, body, nil)
		}
	case *parser.LiteralHash:
		hash := body.(*parser.LiteralHash)
		ta = createMetaType(na, name, `Object`, extractParentName(hash), hash)
	}

	if ta == nil {
		panic(fmt.Sprintf(`cannot create object from a %T`, body))
	}
	return ta
}

func extractParentName(hash *parser.LiteralHash) string {
	for _, he := range hash.Entries() {
		ke := he.(*parser.KeyedEntry)
		if k, ok := ke.Key().(*parser.LiteralString); ok && k.StringValue() == `parent` {
			if pr, ok := ke.Value().(*parser.QualifiedReference); ok {
				return pr.Name()
			}
		}
	}
	return ``
}

func createMetaType(na eval.URI, name string, typeName string, parentName string, hash *parser.LiteralHash) eval.Type {
	if parentName == `` {
		switch typeName {
		case `Object`:
			return NewObjectType(name, nil, hash)
		default:
			return NewTypeSetType(na, name, hash)
		}
	}

	return NewObjectType(name, NewTypeReferenceType(parentName), hash)
}

func argError(e eval.Type, a eval.Value) errors.InstantiationError {
	return errors.NewArgumentsError(``, eval.DescribeMismatch(`assert`, e, a.PType()))
}

func typeArg(hash eval.OrderedMap, key string, d eval.Type) eval.Type {
	v := hash.Get5(key, nil)
	if v == nil {
		return d
	}
	if t, ok := v.(eval.Type); ok {
		return t
	}
	panic(argError(DefaultTypeType(), v))
}

func hashArg(hash eval.OrderedMap, key string) *HashValue {
	v := hash.Get5(key, nil)
	if v == nil {
		return _EMPTY_MAP
	}
	if t, ok := v.(*HashValue); ok {
		return t
	}
	panic(argError(DefaultHashType(), v))
}

func boolArg(hash eval.OrderedMap, key string, d bool) bool {
	v := hash.Get5(key, nil)
	if v == nil {
		return d
	}
	if t, ok := v.(booleanValue); ok {
		return t.Bool()
	}
	panic(argError(DefaultBooleanType(), v))
}

type LazyType interface {
	LazyIsInstance(v eval.Value, g eval.Guard) int
}

func LazyIsInstance(a eval.Type, b eval.Value, g eval.Guard) int {
	if lt, ok := a.(LazyType); ok {
		return lt.LazyIsInstance(b, g)
	}
	if a.IsInstance(b, g) {
		return 1
	}
	return -1
}

func stringArg(hash eval.OrderedMap, key string, d string) string {
	v := hash.Get5(key, nil)
	if v == nil {
		return d
	}
	if t, ok := v.(stringValue); ok {
		return string(t)
	}
	panic(argError(DefaultStringType(), v))
}

func uriArg(hash eval.OrderedMap, key string, d eval.URI) eval.URI {
	v := hash.Get5(key, nil)
	if v == nil {
		return d
	}
	if t, ok := v.(stringValue); ok {
		str := string(t)
		if _, err := ParseURI2(str, true); err != nil {
			panic(eval.Error(eval.EVAL_INVALID_URI, issue.H{`str`: str, `detail`: err.Error()}))
		}
		return eval.URI(str)
	}
	if t, ok := v.(*UriValue); ok {
		return eval.URI(t.URL().String())
	}
	panic(argError(DefaultUriType(), v))
}

func versionArg(hash eval.OrderedMap, key string, d semver.Version) semver.Version {
	v := hash.Get5(key, nil)
	if v == nil {
		return d
	}
	if s, ok := v.(stringValue); ok {
		sv, err := semver.ParseVersion(string(s))
		if err != nil {
			panic(eval.Error(eval.EVAL_INVALID_VERSION, issue.H{`str`: string(s), `detail`: err.Error()}))
		}
		return sv
	}
	if sv, ok := v.(*SemVerValue); ok {
		return sv.Version()
	}
	panic(argError(DefaultSemVerType(), v))
}

func versionRangeArg(hash eval.OrderedMap, key string, d semver.VersionRange) semver.VersionRange {
	v := hash.Get5(key, nil)
	if v == nil {
		return d
	}
	if s, ok := v.(stringValue); ok {
		sr, err := semver.ParseVersionRange(string(s))
		if err != nil {
			panic(eval.Error(eval.EVAL_INVALID_VERSION_RANGE, issue.H{`str`: string(s), `detail`: err.Error()}))
		}
		return sr
	}
	if sv, ok := v.(*SemVerRangeValue); ok {
		return sv.VersionRange()
	}
	panic(argError(DefaultSemVerType(), v))
}
