package types

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-issues/issue"
	"reflect"
	"github.com/puppetlabs/go-evaluator/utils"
	"github.com/puppetlabs/go-semver/semver"
	"strings"
)

const tagName = "puppet"

type reflector struct {
	c eval.Context
}

var pValueType = reflect.TypeOf([]eval.PValue{}).Elem()

func NewReflector(c eval.Context) eval.Reflector {
	return &reflector{c}
}

func (r *reflector) StructName(prefix string, t reflect.Type) string {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if name, ok := r.c.ImplementationRegistry().ReflectedToPtype(t); ok {
		if !strings.HasPrefix(name, prefix) {
			panic(eval.Error(r.c, eval.EVAL_TYPESET_ILLEGAL_NAME_PREFIX, issue.H{`name`: name, `expected_prefix`: prefix}))
		}
		return name
	}
	return prefix + t.Name()
}

func (r *reflector) FieldName(f *reflect.StructField) string {
	if tagHash, ok := r.TagHash(f); ok {
		if nv, ok := tagHash.Get4(`name`); ok {
			return nv.String()
		}
	}
	return utils.CamelToSnakeCase(f.Name)
}

func (r *reflector) Reflect(src eval.PValue, rt reflect.Type) reflect.Value {
	if rt != nil && rt.Kind() == reflect.Interface && rt.AssignableTo(pValueType) {
		sv := reflect.ValueOf(src)
		if sv.Type().AssignableTo(rt) {
			return sv
		}
	}
	if sn, ok := src.(eval.PReflected); ok {
		return sn.Reflect(r.c)
	}
	panic(eval.Error(r.c, eval.EVAL_INVALID_SOURCE_FOR_GET, issue.H{`type`: src.Type()}))
}

// ReflectTo assigns the native value of src to dest
func (r *reflector) ReflectTo(src eval.PValue, dest reflect.Value) {
	if dest.Kind() == reflect.Ptr && !dest.CanSet() {
		dest = dest.Elem()
	}
	assertSettable(r.c, dest)
	if dest.Kind() == reflect.Interface && dest.Type().AssignableTo(pValueType) {
		sv := reflect.ValueOf(src)
		if !sv.Type().AssignableTo(dest.Type()) {
			panic(eval.Error(r.c, eval.EVAL_ATTEMPT_TO_SET_WRONG_KIND, issue.H{`expected`: sv.Type().String(), `actual`: dest.Type().String()}))
		}
		dest.Set(sv)
	} else {
		switch src.(type) {
		case eval.PReflected:
			src.(eval.PReflected).ReflectTo(r.c, dest)
		case eval.PuppetObject:
			po := src.(eval.PuppetObject)
			po.Type().(eval.ObjectType).ToReflectedValue(r.c, po, dest)
		default:
			panic(eval.Error(r.c, eval.EVAL_INVALID_SOURCE_FOR_SET, issue.H{`type`: src.Type()}))
		}
	}
}

func (r *reflector) ReflectType(src eval.PType) (reflect.Type, bool) {
	return ReflectType(src)
}

func ReflectType(src eval.PType) (reflect.Type, bool) {
	if sn, ok := src.(eval.PReflectedType); ok {
		return sn.ReflectType()
	}
	return nil, false
}

func (r *reflector) TagHash(f *reflect.StructField) (eval.KeyedValue, bool) {
	if tag := f.Tag.Get(tagName); tag != `` {
		tagExpr := r.c.ParseAndValidate(``, `{`+tag+`}`, true)
		if tagHash, ok := r.c.Evaluate(tagExpr).(eval.KeyedValue); ok {
			return tagHash, true
		}
	}
	return nil, false
}

func (r *reflector) ObjectTypeFromReflect(typeName string, parent eval.PType, structType reflect.Type) eval.ObjectType {
	structType = r.checkStructType(structType)
	nf := structType.NumField()
	es := make([]*HashEntry, 0, nf)
	for i := 0; i < nf; i++ {
		f := structType.Field(i)
		if i == 0 && f.Anonymous {
			// Parent
			continue
		}

		name := ``
		var typ eval.PType
		var val eval.PValue

		as := make([]*HashEntry, 0)

		if fh, ok := r.TagHash(&f); ok {
			if v, ok := fh.Get4(`name`); ok {
				name = v.String()
			}
			if v, ok := fh.GetEntry(`kind`); ok {
				as = append(as, v.(*HashEntry))
			}
			if v, ok := fh.GetEntry(`value`); ok {
				val = v.Value()
				as = append(as, v.(*HashEntry))
			}
			if v, ok := fh.Get4(`type`); ok {
				if t, ok := v.(eval.PType); ok {
					typ = t
				}
			}
		}

		if typ == nil {
			typ = eval.WrapType(r.c, f.Type)
		}
		// Convenience. If a value is declared as being undef, then make the
		// type Optional if undef isn't an acceptable value
		if val != nil && eval.Equals(val, eval.UNDEF) {
			if !typ.IsInstance(r.c, eval.UNDEF, nil) {
				typ = NewOptionalType(typ)
			}
		}
		as = append(as, WrapHashEntry2(`type`, typ))
		if name == `` {
			name = utils.CamelToSnakeCase(f.Name)
		}

		es = append(es, WrapHashEntry2(name, WrapHash(as)))
	}
	ot := NewObjectType(typeName, parent, SingletonHash2(`attributes`, WrapHash(es)))
	r.c.ImplementationRegistry().RegisterType(r.c, typeName, structType)
	return ot
}

func (r *reflector) TypeSetFromReflect(typeSetName string, version semver.Version, structTypes ...reflect.Type) eval.TypeSet {
	types := make([]*HashEntry, 0)
	prefix := typeSetName + `::`
	for _, structType := range structTypes {
		var parent eval.PType
		structType = r.checkStructType(structType)
		nf := structType.NumField()
		if nf > 0 {
			f := structType.Field(0)
			if f.Anonymous && f.Type.Kind() == reflect.Struct {
				parent = NewTypeReferenceType(r.StructName(prefix, f.Type))
			}
		}
		name := r.StructName(prefix, structType)
		types = append(types, WrapHashEntry2(
			name[strings.LastIndex(name, `::`) + 2:],
			r.ObjectTypeFromReflect(name, parent, structType)))
	}

	es := make([]*HashEntry, 0)
	es = append(es, WrapHashEntry2(eval.KEY_PCORE_URI, WrapString(string(eval.PCORE_URI))))
	es = append(es, WrapHashEntry2(eval.KEY_PCORE_VERSION, WrapSemVer(eval.PCORE_VERSION)))
	es = append(es, WrapHashEntry2(KEY_VERSION, WrapSemVer(version)))
	es = append(es, WrapHashEntry2(KEY_TYPES, WrapHash(types)))
	return NewTypeSetType(eval.RUNTIME_NAME_AUTHORITY, typeSetName, WrapHash(es))
}

func assertSettable(c eval.Context, value reflect.Value) {
	if !value.CanSet() {
		panic(eval.Error(c, eval.EVAL_ATTEMPT_TO_SET_UNSETTABLE, issue.H{`kind`: value.Type().String()}))
	}
}

func assertKind(c eval.Context, value reflect.Value, kind reflect.Kind) {
	vk := value.Kind()
	if vk != kind && vk != reflect.Interface {
		panic(eval.Error(c, eval.EVAL_ATTEMPT_TO_SET_WRONG_KIND, issue.H{`expected`: kind.String(), `actual`: vk.String()}))
	}
}

func (r *reflector) checkStructType(t reflect.Type) reflect.Type {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		panic(eval.Error(r.c, eval.EVAL_ATTEMPT_TO_SET_WRONG_KIND, issue.H{`expected`: `Struct`, `actual`: t.Kind().String()}))
	}
	return t
}
