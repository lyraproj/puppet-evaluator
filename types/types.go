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

	"strings"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/puppet-evaluator/errors"
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-parser/parser"
	"github.com/lyraproj/semver/semver"
)

const (
	NoString = "\x00"

	HkBinary       = byte('B')
	HkBoolean      = byte('b')
	HkDefault      = byte('d')
	HkFloat        = byte('f')
	HkInteger      = byte('i')
	HkRegexp       = byte('r')
	HkTimespan     = byte('D')
	HkTimestamp    = byte('T')
	HkType         = byte('t')
	HkUndef        = byte('u')
	HkUri          = byte('U')
	HkVersion      = byte('v')
	HkVersionRange = byte('R')

	IntegerHex = `(?:0[xX][0-9A-Fa-f]+)`
	IntegerOct = `(?:0[0-7]+)`
	IntegerBin = `(?:0[bB][01]+)`
	IntegerDec = `(?:0|[1-9]\d*)`
	SignPrefix = `[+-]?\s*`

	OptionalFraction = `(?:\.\d+)?`
	OptionalExponent = `(?:[eE]-?\d+)?`
	FloatDec         = `(?:` + IntegerDec + OptionalFraction + OptionalExponent + `)`

	IntegerPattern = `\A` + SignPrefix + `(?:` + IntegerDec + `|` + IntegerHex + `|` + IntegerOct + `|` + IntegerBin + `)\z`
	FloatPattern   = `\A` + SignPrefix + `(?:` + FloatDec + `|` + IntegerHex + `|` + IntegerOct + `|` + IntegerBin + `)\z`
)

// isInstance answers if value is an instance of the given puppetType
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
	if a == b || a == anyTypeDefault {
		return true
	}
	switch b := b.(type) {
	case nil:
		return false
	case *UnitType:
		return true
	case *NotUndefType:
		nt := b.typ
		if !GuardedIsAssignable(nt, undefTypeDefault, g) {
			return GuardedIsAssignable(a, nt, g)
		}
	case *OptionalType:
		if GuardedIsAssignable(a, undefTypeDefault, g) {
			ot := b.typ
			return ot == nil || GuardedIsAssignable(a, ot, g)
		}
		return false
	case *TypeAliasType:
		return GuardedIsAssignable(a, b.resolvedType, g)
	case *VariantType:
		return b.allAssignableTo(a, g)
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

func NewIllegalArgumentType(name string, index int, expected string, actual eval.Value) errors.InstantiationError {
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
			WrapValues(params).ToString(b, s.Subsequent(), g)
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
	return dataTypeDefault
}

func DefaultRichDataType() *TypeAliasType {
	return richDataTypeDefault
}

func NilAs(dflt, t eval.Type) eval.Type {
	if t == nil {
		t = dflt
	}
	return t
}

func CopyAppend(types []eval.Type, t eval.Type) []eval.Type {
	top := len(types)
	tc := make([]eval.Type, top+1)
	copy(tc, types)
	tc[top] = t
	return tc
}

var dataArrayTypeDefault = &ArrayType{IntegerTypePositive, &TypeReferenceType{`Data`}}
var dataHashTypeDefault = &HashType{IntegerTypePositive, stringTypeDefault, &TypeReferenceType{`Data`}}
var dataTypeDefault = &TypeAliasType{name: `Data`, resolvedType: &VariantType{[]eval.Type{scalarDataTypeDefault, undefTypeDefault, dataArrayTypeDefault, dataHashTypeDefault}}}

var richKeyTypeDefault = &VariantType{[]eval.Type{stringTypeDefault, numericTypeDefault}}
var richDataArrayTypeDefault = &ArrayType{IntegerTypePositive, &TypeReferenceType{`RichData`}}
var richDataHashTypeDefault = &HashType{IntegerTypePositive, richKeyTypeDefault, &TypeReferenceType{`RichData`}}
var richDataTypeDefault = &TypeAliasType{`RichData`, nil, &VariantType{
	[]eval.Type{scalarTypeDefault,
		binaryTypeDefault,
		defaultTypeDefault,
		objectTypeDefault,
		typeTypeDefault,
		typeSetTypeDefault,
		undefTypeDefault,
		richDataArrayTypeDefault,
		richDataHashTypeDefault}}, nil}

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
	dataArrayTypeDefault.typ = dataTypeDefault
	dataHashTypeDefault.valueType = dataTypeDefault
	richDataArrayTypeDefault.typ = richDataTypeDefault
	richDataHashTypeDefault.valueType = richDataTypeDefault

	eval.DefaultFor = defaultFor
	eval.Generalize = generalize
	eval.Normalize = normalize
	eval.IsAssignable = isAssignable
	eval.IsInstance = isInstance
	eval.New = newInstance

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
		switch tv := tv.(type) {
		case *UndefValue:
			return false
		case booleanValue:
			return tv.Bool()
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
func newInstance(c eval.Context, receiver eval.Value, args ...eval.Value) eval.Value {
	var name string
	typ, ok := receiver.(eval.Type)
	if ok {
		name = typ.Name()
	} else {
		// Type might be in string form
		_, ok = receiver.(stringValue)
		if !ok {
			// Only types or names of types can be used
			panic(eval.Error(eval.InstanceDoesNotRespond, issue.H{`type`: receiver.PType(), `message`: `new`}))
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
		panic(eval.Error(eval.InstanceDoesNotRespond, issue.H{`type`: name, `message`: `new`}))
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
	obj := AllocObjectType()
	obj.name = name
	obj.initHashExpression = zeroValue
	registerResolvableType(obj)
	return obj
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
		b.WriteByte(HkType)
		b.Write([]byte(pt.Name()))
		if ppt, ok := pt.(eval.ParameterizedType); ok {
			for _, p := range ppt.Parameters() {
				appendTypeParamKey(b, p)
			}
		}
	} else if hk, ok := v.(eval.HashKeyValue); ok {
		b.Write([]byte(hk.ToKey()))
	} else {
		panic(NewIllegalArgumentType(`ToKey`, 0, `value used as hash key`, v))
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
	switch v := v.(type) {
	case nil:
		pv = undef
	case eval.Value:
		pv = v
	case string:
		pv = stringValue(v)
	case int8:
		pv = integerValue(int64(v))
	case int16:
		pv = integerValue(int64(v))
	case int32:
		pv = integerValue(int64(v))
	case int64:
		pv = integerValue(v)
	case byte:
		pv = integerValue(int64(v))
	case int:
		pv = integerValue(int64(v))
	case float64:
		pv = floatValue(v)
	case bool:
		pv = booleanValue(v)
	case *regexp.Regexp:
		pv = WrapRegexp2(v)
	case []byte:
		pv = WrapBinary(v)
	case semver.Version:
		pv = WrapSemVer(v)
	case semver.VersionRange:
		pv = WrapSemVerRange(v)
	case time.Duration:
		pv = WrapTimespan(v)
	case time.Time:
		pv = WrapTimestamp(v)
	case []int:
		pv = WrapInts(v)
	case []string:
		pv = WrapStrings(v)
	case []eval.Value:
		pv = WrapValues(v)
	case []eval.Type:
		pv = WrapTypes(v)
	case []interface{}:
		return WrapInterfaces(c, v)
	case map[string]interface{}:
		pv = WrapStringToInterfaceMap(c, v)
	case map[string]string:
		pv = WrapStringToStringMap(v)
	case map[string]eval.Value:
		pv = WrapStringToValueMap(v)
	case map[string]eval.Type:
		pv = WrapStringToTypeMap(v)
	case json.Number:
		if i, err := v.Int64(); err == nil {
			pv = integerValue(i)
		} else {
			f, _ := v.Float64()
			pv = floatValue(f)
		}
	case reflect.Value:
		pv = wrapReflected(c, v)
	case reflect.Type:
		var err error
		if pv, err = wrapReflectedType(c, v); err != nil {
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
		return undef
	}

	vi := vr

	// Check for nil
	switch vr.Kind() {
	case reflect.Ptr, reflect.Slice, reflect.Array, reflect.Map, reflect.Interface:
		if vr.IsNil() {
			return undef
		}

		if vi.Kind() == reflect.Interface {
			// Need implementation here.
			vi = vi.Elem()
		}
	}

	if _, ok := wellKnown[vr.Type()]; ok {
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
			pv = undef
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

var wellKnown map[reflect.Type]eval.Type

func wrapReflectedType(c eval.Context, vt reflect.Type) (pt eval.Type, err error) {
	if c == nil {
		c = eval.CurrentContext()
	}

	var ok bool
	if pt, ok = wellKnown[vt]; ok {
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
			err = eval.Error(eval.UnreflectableType, issue.H{`type`: vt.String()})
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
		rt, _ := CreateTypeDefinition(ta, eval.RuntimeNameAuthority)
		at := rt.(*TypeAliasType)
		registerResolvableType(at)
		return at
	}
	panic(convertReported(eval.Error2(expr, eval.NoDefinition, issue.H{`source`: ``, `type`: eval.NsType, `name`: name}), fileName, fileLine))
}

func convertReported(err error, fileName string, lineOffset int) error {
	if ri, ok := err.(issue.Reported); ok {
		return ri.OffsetByLocation(issue.NewLocation(fileName, lineOffset, 0))
	}
	return err
}

func CreateTypeDefinition(d parser.Definition, na eval.URI) (interface{}, eval.TypedName) {
	switch d := d.(type) {
	case *parser.TypeAlias:
		name := d.Name()
		return createTypeDefinition(na, name, d.Type()), eval.NewTypedName2(eval.NsType, name, na)
	default:
		panic(fmt.Sprintf(`Don't know how to define a %T`, d))
	}
}

func createTypeDefinition(na eval.URI, name string, body parser.Expression) eval.Type {
	var ta eval.Type
	switch body := body.(type) {
	case *parser.QualifiedReference:
		ta = NewTypeAliasType(name, body, nil)
	case *parser.AccessExpression:
		ta = nil
		if len(body.Keys()) == 1 {
			arg := body.Keys()[0]
			if hash, ok := arg.(*parser.LiteralHash); ok {
				if lq, ok := body.Operand().(*parser.QualifiedReference); ok {
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
		ta = createMetaType(na, name, `Object`, extractParentName(body), body)
	}

	if ta == nil {
		panic(fmt.Sprintf(`cannot create object from a %T`, body))
	}
	return ta
}

func NamedType(na eval.URI, name string, value eval.Value) eval.Type {
	var ta eval.Type
	if na == `` {
		na = eval.RuntimeNameAuthority
	}
	if dt, ok := value.(*DeferredType); ok {
		if len(dt.params) == 1 {
			if hash, ok := dt.params[0].(eval.OrderedMap); ok && dt.Name() != `Struct` {
				if dt.Name() == `Object` {
					ta = createMetaType2(na, name, dt.Name(), extractParentName2(hash), hash)
				} else if dt.Name() == `TypeSet` {
					ta = createMetaType2(na, name, dt.Name(), ``, hash)
				} else {
					ta = createMetaType2(na, name, `Object`, dt.Name(), hash)
				}
			}
		}
		if ta == nil {
			ta = NewTypeAliasType(name, dt, nil)
		}
	} else if h, ok := value.(eval.OrderedMap); ok {
		ta = createMetaType2(na, name, `Object`, ``, h)
	} else {
		panic(fmt.Sprintf(`cannot create object from a %s`, dt.String()))
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

func extractParentName2(hash eval.OrderedMap) string {
	if p, ok := hash.Get4(keyParent); ok {
		if dt, ok := p.(*DeferredType); ok {
			return dt.Name()
		}
		if s, ok := p.(eval.StringValue); ok {
			return s.String()
		}
	}
	return ``
}

func createMetaType(na eval.URI, name string, typeName string, parentName string, hash *parser.LiteralHash) eval.Type {
	if parentName == `` {
		switch typeName {
		case `Object`:
			return ObjectTypeFromAST(name, nil, hash)
		default:
			return NewTypeSetType(na, name, hash)
		}
	}

	return ObjectTypeFromAST(name, NewTypeReferenceType(parentName), hash)
}

func createMetaType2(na eval.URI, name string, typeName string, parentName string, hash eval.OrderedMap) eval.Type {
	if parentName == `` {
		switch typeName {
		case `Object`:
			return newObjectType3(name, nil, hash)
		default:
			return newTypeSetType3(na, name, hash)
		}
	}

	return newObjectType3(name, NewTypeReferenceType(parentName), hash)
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
		return emptyMap
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
			panic(eval.Error(eval.InvalidUri, issue.H{`str`: str, `detail`: err.Error()}))
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
			panic(eval.Error(eval.InvalidVersion, issue.H{`str`: string(s), `detail`: err.Error()}))
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
			panic(eval.Error(eval.InvalidVersionRange, issue.H{`str`: string(s), `detail`: err.Error()}))
		}
		return sr
	}
	if sv, ok := v.(*SemVerRangeValue); ok {
		return sv.VersionRange()
	}
	panic(argError(DefaultSemVerType(), v))
}
