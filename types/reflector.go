package types

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-issues/issue"
	"reflect"
	"github.com/puppetlabs/go-semver/semver"
	"strings"
)

const tagName = "puppet"

type reflector struct {
	c eval.Context
}

var pValueType = reflect.TypeOf((*eval.Value)(nil)).Elem()

func NewReflector(c eval.Context) eval.Reflector {
	return &reflector{c}
}

func Methods(t reflect.Type) []reflect.Method {
	if t.Kind() == reflect.Ptr {
		// Pointer may have methods
		if t.NumMethod() == 0 {
			t = t.Elem()
		}
	}
	nm := t.NumMethod()
	ms := make([]reflect.Method, nm)
	for i := 0; i < nm; i++ {
		ms[i] = t.Method(i)
	}
	return ms
}

func Fields(t reflect.Type) []reflect.StructField {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	nf := 0
	if t.Kind() == reflect.Struct {
		nf = t.NumField()
	}
	fs := make([]reflect.StructField, nf)
	for i := 0; i < nf; i++ {
		fs[i] = t.Field(i)
	}
	return fs
}

// NormalizeType ensures that pointers to interface is converted to interface and that struct is converted to
// pointer to struct
func NormalizeType(rt reflect.Type) reflect.Type {
	switch rt.Kind() {
	case reflect.Struct:
		rt = reflect.PtrTo(rt)
	case reflect.Ptr:
		re := rt.Elem()
		if re.Kind() == reflect.Interface {
			rt = re
		}
	}
	return rt
}

func (r *reflector) Methods(t reflect.Type) []reflect.Method {
	return Methods(t)
}

func (r *reflector) Fields(t reflect.Type) []reflect.StructField {
	return Fields(t)
}

func (r *reflector) FieldName(f *reflect.StructField) string {
	if tagHash, ok := r.TagHash(f); ok {
		if nv, ok := tagHash.Get4(`name`); ok {
			return nv.String()
		}
	}
	return issue.CamelToSnakeCase(f.Name)
}

func (r *reflector) Reflect(src eval.Value) reflect.Value {
	if sn, ok := src.(eval.Reflected); ok {
		return sn.Reflect(r.c)
	}
	panic(eval.Error(eval.EVAL_UNREFLECTABLE_VALUE, issue.H{`type`: src.PType()}))
}

func (r *reflector) Reflect2(src eval.Value, rt reflect.Type) reflect.Value {
	if rt != nil && rt.Kind() == reflect.Interface && rt.AssignableTo(pValueType) {
		sv := reflect.ValueOf(src)
		if sv.Type().AssignableTo(rt) {
			return sv
		}
	}
	return r.Reflect(src)
}

// ReflectTo assigns the native value of src to dest
func (r *reflector) ReflectTo(src eval.Value, dest reflect.Value) {
	assertSettable(&dest)
	if dest.Kind() == reflect.Interface && dest.Type().AssignableTo(pValueType) {
		sv := reflect.ValueOf(src)
		if !sv.Type().AssignableTo(dest.Type()) {
			panic(eval.Error(eval.EVAL_ATTEMPT_TO_SET_WRONG_KIND, issue.H{`expected`: sv.Type().String(), `actual`: dest.Type().String()}))
		}
		dest.Set(sv)
	} else {
		switch src.(type) {
		case eval.Reflected:
			src.(eval.Reflected).ReflectTo(r.c, dest)
		case eval.PuppetObject:
			po := src.(eval.PuppetObject)
			po.PType().(eval.ObjectType).ToReflectedValue(r.c, po, dest)
		default:
			panic(eval.Error(eval.EVAL_INVALID_SOURCE_FOR_SET, issue.H{`type`: src.PType()}))
		}
	}
}

func (r *reflector) ReflectType(src eval.Type) (reflect.Type, bool) {
	return ReflectType(r.c, src)
}

func ReflectType(c eval.Context, src eval.Type) (reflect.Type, bool) {
	if sn, ok := src.(eval.ReflectedType); ok {
		return sn.ReflectType(c)
	}
	return nil, false
}

func (r *reflector) TagHash(f *reflect.StructField) (eval.OrderedMap, bool) {
	if tag := f.Tag.Get(tagName); tag != `` {
		tagExpr := r.c.ParseAndValidate(``, `{`+tag+`}`, true)
		if tagHash, ok := eval.Evaluate(r.c, tagExpr).(eval.OrderedMap); ok {
			return tagHash, true
		}
	}
	return nil, false
}

var errorType = reflect.TypeOf((*error)(nil)).Elem()

func (r *reflector) ObjectTypeFromReflect(typeName string, parent eval.Type, rf reflect.Type) eval.ObjectType {
	return NewObjectType3(typeName, parent, func(obj eval.ObjectType) *HashValue {
		obj.(*objectType).goType = rf

		r.c.ImplementationRegistry().RegisterType(r.c, obj, rf)

		ie := make([]*HashEntry, 0, 2)
		var pt reflect.Type

		fs := r.Fields(rf)
		nf := len(fs)
		if nf > 0 {
			es := make([]*HashEntry, 0, nf)
			for i, f := range fs {
				if i == 0 && f.Anonymous {
					// Parent
					pt = reflect.PtrTo(f.Type)
					continue
				}
				if f.PkgPath != `` {
					// Unexported
					continue
				}

				name, decl := r.ReflectFieldTags(&f)
				es = append(es, WrapHashEntry2(name, decl))
			}
			ie = append(ie, WrapHashEntry2(`attributes`, WrapHash(es)))
		}

		ms := r.Methods(rf)
		nm := len(ms)
		if nm > 0 {
			es := make([]*HashEntry, 0, nm)
			for _, m := range ms {
				if m.PkgPath != `` {
					// Not exported struct method
					continue
				}

				if pt != nil {
					if _, ok := pt.MethodByName(m.Name); ok {
						// Redeclarations of parent method are not included
						continue
					}
				}

				mt := m.Type
				returnsError := false
				var rt eval.Type
				switch mt.NumOut() {
				case 0:
					rt = DefaultAnyType()
				case 1:
					ot := mt.Out(0)
					if ot.AssignableTo(errorType) {
						returnsError = true
					} else {
						rt = wrapReflectedType(r.c, mt.Out(0))
					}
				case 2:
					rt = wrapReflectedType(r.c, mt.Out(0))
					ot := mt.Out(1)
					if ot.AssignableTo(errorType) {
						returnsError = true
						break
					}
					fallthrough
				default:
					panic(eval.Error(eval.EVAL_UNREFLECTABLE_RETURN, issue.H{`type`: rf.Name(), `method`: m.Name}))
				}

				var pt *TupleType
				pc := mt.NumIn()
				ix := 0
				if rf.Kind() != reflect.Interface {
					// First argumnet is the receiver itself
					ix = 1
				}

				if pc == ix {
					pt = EmptyTupleType()
				} else {
					ps := make([]eval.Type, pc - ix)
					for p := ix; p < pc; p++ {
						ps[p-ix] = wrapReflectedType(r.c, mt.In(p))
					}
					pt = NewTupleType(ps, nil)
				}
				ne := 2
				if returnsError {
					ne++
				}
				ds := make([]*HashEntry, ne)
				ds[0] = WrapHashEntry2(KEY_TYPE, NewCallableType(pt, rt, nil))
				ds[1] = WrapHashEntry2(KEY_GONAME, WrapString(m.Name))
				if returnsError {
					ds[2] = WrapHashEntry2(KEY_RETURNS_ERROR, Boolean_TRUE)
				}
				es = append(es, WrapHashEntry2(issue.CamelToSnakeCase(m.Name), WrapHash(ds)))
			}
			ie = append(ie, WrapHashEntry2(`functions`, WrapHash(es)))
		}
		return WrapHash(ie)
	})
}

func (r *reflector) ReflectFieldTags(f *reflect.StructField) (name string, decl eval.OrderedMap) {
	as := make([]*HashEntry, 0)
	var val eval.Value
	var typ eval.Type

	if fh, ok := r.TagHash(f); ok {
		if v, ok := fh.Get4(KEY_NAME); ok {
			name = v.String()
		}
		if v, ok := fh.GetEntry(KEY_KIND); ok {
			as = append(as, v.(*HashEntry))
		}
		if v, ok := fh.GetEntry(KEY_VALUE); ok {
			val = v.Value()
			as = append(as, v.(*HashEntry))
		}
		if v, ok := fh.Get4(KEY_TYPE); ok {
			if t, ok := v.(eval.Type); ok {
				typ = t
			}
		}
	}

	if typ == nil {
		typ = eval.WrapReflectedType(r.c, f.Type)
	}
	// Convenience. If a value is declared as being undef, then make the
	// type Optional if undef isn't an acceptable value
	if val != nil && eval.Equals(val, eval.UNDEF) {
		if !typ.IsInstance(eval.UNDEF, nil) {
			typ = NewOptionalType(typ)
		}
	}
	as = append(as, WrapHashEntry2(KEY_TYPE, typ))
	as = append(as, WrapHashEntry2(KEY_GONAME, WrapString(f.Name)))
	if name == `` {
		name = issue.CamelToSnakeCase(f.Name)
	}
	return name, WrapHash(as)
}

func (r *reflector) TypeSetFromReflect(typeSetName string, version semver.Version, aliases map[string]string, rTypes ...reflect.Type) eval.TypeSet {
	types := make([]*HashEntry, 0)
	prefix := typeSetName + `::`
	for _, rt := range rTypes {
		var parent eval.Type
		fs := r.Fields(rt)
		nf := len(fs)
		if nf > 0 {
			f := fs[0]
			if f.Anonymous && f.Type.Kind() == reflect.Struct {
				parent = NewTypeReferenceType(typeName(prefix, aliases, f.Type))
			}
		}
		name := typeName(prefix, aliases, rt)
		types = append(types, WrapHashEntry2(
			name[strings.LastIndex(name, `::`) + 2:],
			r.ObjectTypeFromReflect(name, parent, rt)))
	}

	es := make([]*HashEntry, 0)
	es = append(es, WrapHashEntry2(eval.KEY_PCORE_URI, WrapString(string(eval.PCORE_URI))))
	es = append(es, WrapHashEntry2(eval.KEY_PCORE_VERSION, WrapSemVer(eval.PCORE_VERSION)))
	es = append(es, WrapHashEntry2(KEY_VERSION, WrapSemVer(version)))
	es = append(es, WrapHashEntry2(KEY_TYPES, WrapHash(types)))
	return NewTypeSetType(eval.RUNTIME_NAME_AUTHORITY, typeSetName, WrapHash(es))
}

func ParentType(t reflect.Type) reflect.Type {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() == reflect.Struct && t.NumField() > 0 {
		f := t.Field(0)
		if f.Anonymous && f.Type.Kind() == reflect.Struct {
			return f.Type
		}
	}
	return nil
}

func typeName(prefix string, aliases map[string]string, rt reflect.Type) string {
	if rt.Kind() == reflect.Ptr {
		// Pointers have no names
		rt = rt.Elem()
	}
	name := rt.Name()
	if aliases != nil {
		if alias, ok := aliases[name]; ok {
			name = alias
		}
	}
	return prefix + name
}

func assertSettable(value *reflect.Value) {
	if !value.CanSet() {
		panic(eval.Error(eval.EVAL_ATTEMPT_TO_SET_UNSETTABLE, issue.H{`kind`: value.Type().String()}))
	}
}
