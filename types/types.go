package types

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"runtime"
	"sync"
	"time"

	"github.com/puppetlabs/go-evaluator/errors"
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/semver"
	"github.com/puppetlabs/go-parser/issue"
	"github.com/puppetlabs/go-parser/parser"
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
	FLOAT_DEC = `(?:` + INTEGER_DEC + OPTIONAL_FRACTION + OPTIONAL_EXPONENT + `)`

	INTEGER_PATTERN = `\A` + SIGN_PREFIX + `(?:` + INTEGER_DEC + `|` + INTEGER_HEX + `|` + INTEGER_OCT + `|` + INTEGER_BIN + `)\z`
	FLOAT_PATTERN = `\A` + SIGN_PREFIX + `(?:` + FLOAT_DEC + `|` + INTEGER_HEX + `|` + INTEGER_OCT + `|` + INTEGER_BIN + `)\z`
)

type objectTypeAndCtor struct {
  typ eval.ObjectType
  ctor eval.DispatchFunction
}

func (rt *objectTypeAndCtor) Type() eval.ObjectType {
	return rt.typ
}

func (rt *objectTypeAndCtor) Creator() eval.DispatchFunction {
	return rt.ctor
}

// isInstance answers if value is an instance of the given puppeType
func isInstance(puppetType eval.PType, value eval.PValue) bool {
	return GuardedIsInstance(puppetType, value, nil)
}

// isAssignable answers if t is assignable to this type
func isAssignable(puppetType eval.PType, other eval.PType) bool {
	return GuardedIsAssignable(puppetType, other, nil)
}

func generalize(a eval.PType) eval.PType {
	if g, ok := a.(eval.Generalizable); ok {
		return g.Generic()
	}
	if g, ok := a.(eval.ParameterizedType); ok {
		return g.Default()
	}
	return a
}

func defaultFor(t eval.PType) eval.PType {
	if g, ok := t.(eval.ParameterizedType); ok {
		return g.Default()
	}
	return t
}

func normalize(t eval.PType) eval.PType {
	// TODO: Implement for ParameterizedType
	return t
}

func GuardedIsInstance(a eval.PType, v eval.PValue, g eval.Guard) bool {
	return a.IsInstance(v, g)
}

func GuardedIsAssignable(a eval.PType, b eval.PType, g eval.Guard) bool {
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

func UniqueTypes(types []eval.PType) []eval.PType {
	top := len(types)
	if top < 2 {
		return types
	}

	result := make([]eval.PType, 0, top)
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

// ValueSlice convert a slice of values that implement the eval.PValue interface to []eval.PValue. The
// method will panic if the given argument is not a slice or array, or if not all
// elements implement the eval.PValue interface
func ValueSlice(slice interface{}) []eval.PValue {
	sv := reflect.ValueOf(slice)
	top := sv.Len()
	result := make([]eval.PValue, top)
	for idx := 0; idx < top; idx++ {
		result[idx] = sv.Index(idx).Interface().(eval.PValue)
	}
	return result
}

func UniqueValues(values []eval.PValue) []eval.PValue {
	top := len(values)
	if top < 2 {
		return values
	}

	result := make([]eval.PValue, 0, top)
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

func NewIllegalArgumentType2(name string, index int, expected string, actual eval.PValue) errors.InstantiationError {
	return errors.NewIllegalArgumentType(name, index, expected, eval.DetailedValueType(actual).String())
}

func TypeToString(t eval.PType, b io.Writer, s eval.FormatContext, g eval.RDetect) {
	f := eval.GetFormat(s.FormatMap(), t.Type())
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

func basicTypeToString(t eval.PType, b io.Writer, s eval.FormatContext, g eval.RDetect) {
	io.WriteString(b, t.Name())
	if s != EXPANDED {
		switch t.(type) {
		case *objectType, *TypeAliasType, *TypeSetType:
			return
		}
	}
	if pt, ok := t.(eval.ParameterizedType); ok {
		params := pt.Parameters()
		if len(params) > 0 {
			WrapArray(params).ToString(b, s, g)
		}
	}
}

type alterFunc func(t eval.PType) eval.PType

func alterTypes(types []eval.PType, function alterFunc) []eval.PType {
	al := make([]eval.PType, len(types))
	for idx, t := range types {
		al[idx] = function(t)
	}
	return al
}

func toTypes(types eval.IndexedValue) ([]eval.PType, int) {
	top := types.Len()
	if top == 1 {
		if a, ok := types.At(0).(eval.IndexedValue); ok {
			return toTypes(a)
		}
	}
	result := make([]eval.PType, 0, top)
	if types.All(func(t eval.PValue) bool {
		if pt, ok := t.(eval.PType); ok {
			result = append(result, pt)
			return true
		}
		return false
	}) {
		return result, -1
	}
	return nil, len(result)
}

func DefaultDataType() *TypeAliasType {
	return dataType_DEFAULT
}

func DefaultRichDataType() *TypeAliasType {
	return richDataType_DEFAULT
}

func NilAs(dflt, t eval.PType) eval.PType {
	if t == nil {
		t = dflt
	}
	return t
}

func CopyAppend(types []eval.PType, t eval.PType) []eval.PType {
	top := len(types)
	tc := make([]eval.PType, top+1, top+1)
	copy(tc, types)
	tc[top] = t
	return tc
}

var dataArrayType_DEFAULT = &ArrayType{integerType_POSITIVE, &TypeReferenceType{`Data`}}
var dataHashType_DEFAULT = &HashType{integerType_POSITIVE, stringType_DEFAULT, &TypeReferenceType{`Data`}}
var dataType_DEFAULT = &TypeAliasType{name: `Data`, resolvedType: &VariantType{[]eval.PType{scalarDataType_DEFAULT, undefType_DEFAULT, dataArrayType_DEFAULT, dataHashType_DEFAULT}}}

var richKeyType_DEFAULT = &VariantType{[]eval.PType{stringType_DEFAULT, numericType_DEFAULT}}
var richDataArrayType_DEFAULT = &ArrayType{integerType_POSITIVE, &TypeReferenceType{`RichData`}}
var richDataHashType_DEFAULT = &HashType{integerType_POSITIVE, richKeyType_DEFAULT, &TypeReferenceType{`RichData`}}
var richDataType_DEFAULT = &TypeAliasType{`RichData`, nil, &VariantType{
	[]eval.PType{scalarType_DEFAULT, binaryType_DEFAULT, defaultType_DEFAULT, typeType_DEFAULT, undefType_DEFAULT, richDataArrayType_DEFAULT, richDataHashType_DEFAULT}}, nil}

var resolvableTypes = make([]eval.ResolvableType, 0, 16)
var resolvableTypesLock sync.Mutex

type BuildFunctionArgs struct {
	Name string
	LocalTypes eval.LocalTypesCreator
	Creators []eval.DispatchCreator
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

	eval.DetailedValueType = func(value eval.PValue) eval.PType {
		if dt, ok := value.(eval.DetailedTypeValue); ok {
			return dt.DetailedType()
		}
		return value.Type()
	}

	eval.GenericType = func(t eval.PType) eval.PType {
		if g, ok := t.(eval.Generalizable); ok {
			return g.Generic()
		}
		return t
	}

	eval.GenericValueType = func(value eval.PValue) eval.PType {
		return eval.GenericType(value.Type())
	}

	eval.ToArray = func(elements []eval.PValue) eval.IndexedValue {
		return WrapArray(elements)
	}

	eval.ToKey = func(value eval.PValue) eval.HashKey {
		if hk, ok := value.(eval.HashKeyValue); ok {
			return hk.ToKey()
		}
		b := bytes.NewBuffer([]byte{})
		appendKey(b, value)
		return eval.HashKey(b.String())
	}

	eval.IsTruthy = func(tv eval.PValue) bool {
		switch tv.(type) {
		case *UndefValue:
			return false
		case *BooleanValue:
			return tv.(*BooleanValue).Bool()
		default:
			return true
		}
	}

	eval.RegisterResolvableType = registerResolvableType
	eval.NewGoConstructor = newGoConstructor
	eval.NewGoConstructor2 = newGoConstructor2
	eval.WrapUnknown = wrap
}

func newGoConstructor(typeName string, creators ...eval.DispatchCreator) {
	registerGoConstructor(&BuildFunctionArgs{typeName, nil, creators})
}

func newGoConstructor2(typeName string, localTypes eval.LocalTypesCreator, creators ...eval.DispatchCreator) {
	registerGoConstructor(&BuildFunctionArgs{typeName, localTypes, creators})
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

func registerResolvableType(tp eval.ResolvableType) {
	resolvableTypesLock.Lock()
	resolvableTypes = append(resolvableTypes, tp)
	resolvableTypesLock.Unlock()
}

func appendKey(b *bytes.Buffer, v eval.PValue) {
	if hk, ok := v.(eval.StreamHashKeyValue); ok {
		hk.ToKey(b)
	} else if pt, ok := v.(eval.PType); ok {
		b.WriteByte(1)
		b.WriteByte(HK_TYPE)
		b.Write([]byte(pt.Name()))
		if ppt, ok := pt.(eval.ParameterizedType); ok {
			for _, p := range ppt.Parameters() {
				appendKey(b, p)
			}
		}
	} else if hk, ok := v.(eval.HashKeyValue); ok {
		b.Write([]byte(hk.ToKey()))
	} else {
		panic(NewIllegalArgumentType2(`ToKey`, 0, `value used as hash key`, v))
	}
}

func wrap(v interface{}) (pv eval.PValue) {
	switch v.(type) {
	case nil:
		pv = _UNDEF
	case eval.PValue:
		pv = v.(eval.PValue)
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
	case []eval.PValue:
		pv = WrapArray(v.([]eval.PValue))
	case map[string]interface{}:
		pv = WrapHash4(v.(map[string]interface{}))
	case map[string]eval.PValue:
		pv = WrapHash3(v.(map[string]eval.PValue))
	case json.Number:
		if i, err := v.(json.Number).Int64(); err == nil {
			pv = WrapInteger(i)
		} else {
			f, _ := v.(json.Number).Float64()
			pv = WrapFloat(f)
		}
	default:
		// Can still be an alias, slice, or map in which case reflection conversion will work
		pv = wrapValue(reflect.ValueOf(v))
	}
	return pv
}

func wrapValue(vr reflect.Value) (pv eval.PValue) {
	switch vr.Kind() {
	case reflect.String:
		pv = WrapString(vr.String())
	case reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8:
		pv = WrapInteger(vr.Int())
	case reflect.Uint, reflect.Uint64, reflect.Uint32, reflect.Uint16, reflect.Uint8:
		pv = WrapInteger(int64(vr.Uint())) // Possible loss for very large numbers
	case reflect.Bool:
		pv = WrapBoolean(vr.Bool())
	case reflect.Float64, reflect.Float32:
		pv = WrapFloat(vr.Float())
	case reflect.Slice, reflect.Array:
		top := vr.Len()
		els := make([]eval.PValue, top)
		for i := 0; i < top; i++ {
			els[i] = wrap(interfaceOrNil(vr.Index(i)))
		}
		pv = WrapArray(els)
	case reflect.Map:
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

func interfaceOrNil(vr reflect.Value) interface{} {
	if vr.CanInterface() {
		return vr.Interface()
	}
	return nil
}

func init() {
	eval.NewObjectType = newObjectType
}

func newAliasType(name, typeDecl string) eval.PType {
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
	panic(convertReported(eval.Error2(expr, eval.EVAL_NO_DEFINITION, issue.H{`source`: ``, `type`: eval.TYPE, `name`: name}), fileName, fileLine))
}

func newObjectType(name, typeDecl string, creators ...eval.DispatchFunction) eval.ObjectType {
	p := parser.CreateParser()
	_, fileName, fileLine, _ := runtime.Caller(1)
	expr, err := p.Parse(fileName, fmt.Sprintf(`type %s = %s`, name, typeDecl), true)
	if err != nil {
		err = convertReported(err, fileName, fileLine)
		panic(err)
	}

	if ta, ok := expr.(*parser.TypeAlias); ok {
		rt, _ := CreateTypeDefinition(ta, eval.RUNTIME_NAME_AUTHORITY)
		ot := rt.(*objectType)
		ot.setCreators(creators...)
		registerResolvableType(ot)
		return ot
	}
	panic(convertReported(eval.Error2(expr, eval.EVAL_NO_DEFINITION, issue.H{`source`: ``, `type`: eval.TYPE, `name`: name}), fileName, fileLine))
}

func convertReported(err error, fileName string, lineOffset int) error {
	if ri, ok := err.(*issue.Reported); ok {
		return ri.OffsetByLocation(issue.NewLocation(fileName, lineOffset, 0))
	}
	return err
}

func CreateTypeDefinition(d parser.Definition, na eval.URI) (interface{}, eval.TypedName) {
	switch d.(type) {
	case *parser.TypeAlias:
		taExpr := d.(*parser.TypeAlias)
		name := taExpr.Name()
		body := taExpr.Type()
		tn := eval.NewTypedName2(eval.TYPE, name, na)
		var ta eval.PType
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
						if lq.Name() == `Object` || lq.Name() == `TypeSet` {
							ta = createMetaType(name, lq.Name(), hash)
						}
					}
				}
			}
			if ta == nil {
				ta = NewTypeAliasType(name, body, nil)
			}
		case *parser.LiteralHash:
			ta = createMetaType(name, ``, body.(*parser.LiteralHash))
		case *parser.LiteralList:
			ll := body.(*parser.LiteralList)
			if len(ll.Elements()) == 1 {
				if hash, ok := ll.Elements()[0].(*parser.LiteralHash); ok {
					ta = createMetaType(name, ``, hash)
				}
			}
		case *parser.KeyedEntry:
			ke := body.(*parser.KeyedEntry)
			if pn, ok := ke.Key().(*parser.QualifiedReference); ok {
				if hash, ok := ke.Value().(*parser.LiteralHash); ok {
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

func createMetaType(name string, parentName string, hash *parser.LiteralHash) eval.PType {
	if parentName == `` || parentName == `Object` {
		return NewObjectType(name, nil, hash)
	} // TODO else if lq.Name() == `TypeSet`

	return NewObjectType(name, NewTypeReferenceType(parentName), hash)
}
