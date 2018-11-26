package types

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-issues/issue"
	"io"
	"reflect"
	"strings"
)

type typedObject struct {
	typ eval.ObjectType
}

func (o *typedObject) PType() eval.Type {
	return o.typ
}

func (o *typedObject) valuesFromHash(c eval.Context, hash eval.OrderedMap) []eval.Value {
	typ := o.typ.(*objectType)
	va := typ.AttributesInfo().PositionalFromHash(hash)
	if len(va) > 0 && typ.IsParameterized() {
		params := make([]*HashEntry, 0)
		typ.typeParameters(true).EachPair(func(k string, v interface{}) {
			if pv, ok := hash.Get4(k); ok && eval.IsInstance(v.(*typeParameter).typ, pv) {
				params = append(params, WrapHashEntry2(k, pv))
			}
		})
		if len(params) > 0 {
			o.typ = NewObjectTypeExtension(c, typ, []eval.Value{WrapHash(params)})
		}
	}
	return va
}

type attributeSlice struct {
	typedObject
	values []eval.Value
}

func AllocObjectValue(c eval.Context, typ eval.ObjectType) eval.Object {
	if typ.IsMetaType() {
		return AllocObjectType()
	}
	if rf := typ.GoType(); rf != nil {
		if rf.Kind() == reflect.Ptr && rf.Elem().Kind() == reflect.Struct {
			rf = rf.Elem()
		}
		return &reflectedObject{typedObject{typ}, reflect.New(rf).Elem()}
	}
	return &attributeSlice{typedObject{typ}, eval.EMPTY_VALUES}
}

func NewReflectedValue(typ eval.ObjectType, value reflect.Value) eval.Object {
	if value.Kind() == reflect.Func {
		return &reflectedFunc{typedObject{typ}, value}
	}
	return &reflectedObject{typedObject{typ}, value}
}

func NewObjectValue(c eval.Context, typ eval.ObjectType, values []eval.Value) (ov eval.Object) {
	ov = AllocObjectValue(c, typ)
	ov.Initialize(c, values)
	return ov
}

func NewObjectValue2(c eval.Context, typ eval.ObjectType, hash *HashValue) (ov eval.Object) {
	ov = AllocObjectValue(c, typ)
	ov.InitFromHash(c, hash)
	return ov
}

func (o *attributeSlice) Reflect(c eval.Context) reflect.Value {
	ot := o.PType().(eval.ReflectedType)
	if v, ok := ot.ReflectType(c); ok {
		rv := reflect.New(v.Elem())
		o.ReflectTo(c, rv.Elem())
		return rv
	}
	panic(eval.Error(eval.EVAL_UNREFLECTABLE_VALUE, issue.H{`type`: o.PType()}))
}

func (o *attributeSlice) ReflectTo(c eval.Context, value reflect.Value) {
	o.typ.ToReflectedValue(c, o, value)
}

func (o *attributeSlice) Initialize(c eval.Context, values []eval.Value) {
	if len(values) > 0 && o.typ.IsParameterized() {
		o.InitFromHash(c, makeValueHash(o.typ.AttributesInfo(), values))
		return
	}
	fillValueSlice(values, o.typ.AttributesInfo().Attributes())
	o.values = values
}

func (o *attributeSlice) InitFromHash(c eval.Context, hash eval.OrderedMap) {
	o.values = o.valuesFromHash(c, hash)
}

// Ensure that all entries in the value slice that are nil receive default values from the given attributes
func fillValueSlice(values []eval.Value, attrs []eval.Attribute) {
	for ix, v := range values {
		if v == nil {
			at := attrs[ix]
			if !at.HasValue() {
				panic(eval.Error(eval.EVAL_MISSING_REQUIRED_ATTRIBUTE, issue.H{`label`: at.Label()}))
			}
			values[ix] = at.Value()
		}
	}
}

func (o *attributeSlice) Get(key string) (eval.Value, bool) {
	pi := o.typ.AttributesInfo()
	if idx, ok := pi.NameToPos()[key]; ok {
		if idx < len(o.values) {
			return o.values[idx], ok
		}
		return pi.Attributes()[idx].Value(), ok
	}
	return nil, false
}

func (o *attributeSlice) Call(c eval.Context, method eval.ObjFunc, args []eval.Value, block eval.Lambda) (eval.Value, bool) {
	if v, ok := eval.Load(c, NewTypedName(eval.NsFunction, strings.ToLower(o.typ.Name()) + `::` + method.Name())); ok {
		if f, ok := v.(eval.Function); ok {
			return f.Call(c, block, args...), true
		}
	}
	return nil, false
}

func (o *attributeSlice) Equals(other interface{}, g eval.Guard) bool {
	if ov, ok := other.(*attributeSlice); ok {
		return ov.typ.Equals(ov.typ, g) && eval.GuardedEquals(ov.values, ov.values, g)
	}
	return false
}

func (o *attributeSlice) String() string {
	return eval.ToString(o)
}

func (o *attributeSlice) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	ObjectToString(o, s, b, g)
}

func (o *attributeSlice) InitHash() eval.OrderedMap {
	return makeValueHash(o.typ.AttributesInfo(), o.values)
}

// Turn a positional argument list into a hash. The hash will exclude all values
// that are equal to the default value of the corresponding attribute
func makeValueHash(pi eval.AttributesInfo, values []eval.Value) *HashValue {
	posToName := pi.PosToName()
	entries := make([]*HashEntry, 0, len(posToName))
	for i, v := range values {
		attr := pi.Attributes()[i]
		if !(attr.HasValue() && eval.Equals(v, attr.Value())) {
			entries = append(entries, WrapHashEntry2(attr.Name(), v))
		}
	}
	return WrapHash(entries)
}

type reflectedObject struct {
	typedObject
	value reflect.Value
}

func (o *reflectedObject) Call(c eval.Context, method eval.ObjFunc, args []eval.Value, block eval.Lambda) (eval.Value, bool) {
	m, ok := o.value.Type().MethodByName(method.GoName())
	if !ok {
		return nil, false
	}

	mt := m.Type
	rf := c.Reflector()
	rfArgs := make([]reflect.Value, 1 + len(args))
	rfArgs[0] = o.value
	for i, arg := range args {
		pn := i + 1
		av := reflect.New(mt.In(pn)).Elem()
		rf.ReflectTo(arg, av)
		rfArgs[pn] = av
	}

	result := method.(eval.CallableGoMember).CallGoReflected(c, rfArgs)
	switch len(result) {
	case 0:
		return _UNDEF, true
	case 1:
		r := result[0]
		if r.IsValid() {
			return wrap(c, r), true
		} else {
			return _UNDEF, true
		}
	default:
		rs := make([]eval.Value, len(result))
		for i, r := range result {
			if r.IsValid() {
				rs[i] = wrap(c, r)
			} else {
				rs[i] = _UNDEF
			}
		}
		return WrapValues(rs), true
	}
}

func (o *reflectedObject) Reflect(c eval.Context) reflect.Value {
	return o.value
}

func (o *reflectedObject) ReflectTo(c eval.Context, value reflect.Value) {
	if o.value.Kind() == reflect.Struct && value.Kind() == reflect.Ptr {
		value.Set(o.value.Addr())
	} else {
		value.Set(o.value)
	}
}

func (o *reflectedObject) Initialize(c eval.Context, values []eval.Value) {
	if len(values) > 0 && o.typ.IsParameterized() {
		o.InitFromHash(c, makeValueHash(o.typ.AttributesInfo(), values))
		return
	}
	pi := o.typ.AttributesInfo()
	if len(pi.PosToName()) > 0 {
		attrs := pi.Attributes()
		fillValueSlice(values, attrs)
		o.setValues(c, values)
	} else if len(values) == 1 {
		values[0].(eval.Reflected).ReflectTo(c, o.value)
	}
}

func (o *reflectedObject) InitFromHash(c eval.Context, hash eval.OrderedMap) {
	o.setValues(c, o.valuesFromHash(c, hash))
}

func (o *reflectedObject) setValues(c eval.Context, values []eval.Value) {
	attrs := o.typ.AttributesInfo().Attributes()
	rf := c.Reflector()
	if len(attrs) == 1 && attrs[0].GoName() == KEY_VALUE {
		rf.ReflectTo(values[0], o.value)
	} else {
		oe := o.structVal()
		for i, a := range attrs {
			v := values[i]
			rf.ReflectTo(v, oe.FieldByName(a.GoName()))
		}
	}
}

func (o *reflectedObject) Get(key string) (eval.Value, bool) {
	pi := o.typ.AttributesInfo()
	if idx, ok := pi.NameToPos()[key]; ok {
		attr := pi.Attributes()[idx]
		if attr.GoName() == KEY_VALUE {
			return WrapPrimitive(o.value)
		}
		rf := o.structVal().FieldByName(attr.GoName())
		if rf.IsValid() {
			return wrap(nil, rf), true
		}
		return pi.Attributes()[idx].Value(), ok
	}
	return nil, false
}

func (o *reflectedObject) Equals(other interface{}, g eval.Guard) bool {
	if ov, ok := other.(*reflectedObject); ok {
		return o.typ.Equals(ov.typ, g) && reflect.DeepEqual(o.value.Interface(), ov.value.Interface())
	}
	return false
}

func (o *reflectedObject) String() string {
	return eval.ToString(o)
}

func (o *reflectedObject) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	ObjectToString(o, s, b, g)
}

func (o *reflectedObject) InitHash() eval.OrderedMap {
	pi := o.typ.AttributesInfo()
	nc := len(pi.PosToName())
	if nc == 0 {
		return eval.EMPTY_MAP
	}

	if nc == 1 {
		attr := pi.Attributes()[0]
		if attr.GoName() == KEY_VALUE {
			pv, _ := WrapPrimitive(o.value)
			return SingletonHash2(`value`, pv)
		}
	}

	entries := make([]*HashEntry, 0, nc)
	oe := o.structVal()
	for _, attr := range pi.Attributes() {
		gn := attr.GoName()
		if gn != `` {
			v := wrap(nil, oe.FieldByName(gn))
			if !(attr.HasValue() && eval.Equals(v, attr.Value())) {
				entries = append(entries, WrapHashEntry2(attr.Name(), v))
			}
		}
	}
	return WrapHash(entries)
}

func (o *reflectedObject) structVal() reflect.Value {
	oe := o.value
	if oe.Kind() == reflect.Ptr {
		oe = oe.Elem()
	}
	return oe
}

type reflectedFunc struct {
	typedObject
	function reflect.Value
}

func (o *reflectedFunc) Call(c eval.Context, method eval.ObjFunc, args []eval.Value, block eval.Lambda) (eval.Value, bool) {
	mt := o.function.Type()
	rf := c.Reflector()
	rfArgs := make([]reflect.Value, len(args))
	for i, arg := range args {
		av := reflect.New(mt.In(i)).Elem()
		rf.ReflectTo(arg, av)
		rfArgs[i] = av
	}

	pc := mt.NumIn()
	if pc != len(args) {
		panic(eval.Error(eval.EVAL_TYPE_MISMATCH, issue.H{`detail`: eval.DescribeSignatures(
			[]eval.Signature{method.CallableType().(*CallableType)}, NewTupleType([]eval.Type{}, NewIntegerType(int64(pc-1), int64(pc-1))), nil)}))
	}
	result := o.function.Call(rfArgs)
	oc := mt.NumOut()

	if method.ReturnsError() {
		oc--
		err := result[oc].Interface()
		if err != nil {
			if re, ok := err.(issue.Reported); ok {
				panic(re)
			}
			panic(eval.Error(eval.EVAL_GO_FUNCTION_ERROR, issue.H{`name`: mt.Name(), `error`: err}))
		}
		result = result[:oc]
	}

	switch len(result) {
	case 0:
		return _UNDEF, true
	case 1:
		r := result[0]
		if r.IsValid() {
			return wrap(c, r), true
		} else {
			return _UNDEF, true
		}
	default:
		rs := make([]eval.Value, len(result))
		for i, r := range result {
			if r.IsValid() {
				rs[i] = wrap(c, r)
			} else {
				rs[i] = _UNDEF
			}
		}
		return WrapValues(rs), true
	}
}

func (o *reflectedFunc) Reflect(c eval.Context) reflect.Value {
	return o.function
}

func (o *reflectedFunc) ReflectTo(c eval.Context, value reflect.Value) {
	value.Set(o.function)
}

func (o *reflectedFunc) Initialize(c eval.Context, values []eval.Value) {
}

func (o *reflectedFunc) InitFromHash(c eval.Context, hash eval.OrderedMap) {
}

func (o *reflectedFunc) Get(key string) (eval.Value, bool) {
	return nil, false
}

func (o *reflectedFunc) Equals(other interface{}, g eval.Guard) bool {
	if ov, ok := other.(*reflectedFunc); ok {
		return o.typ.Equals(ov.typ, g) && o.function == ov.function
	}
	return false
}

func (o *reflectedFunc) String() string {
	return eval.ToString(o)
}

func (o *reflectedFunc) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	ObjectToString(o, s, b, g)
}

func (o *reflectedFunc) InitHash() eval.OrderedMap {
	return eval.EMPTY_MAP
}
