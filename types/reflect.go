package types

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/utils"
	"github.com/puppetlabs/go-issues/issue"
	"reflect"
	"github.com/puppetlabs/go-semver/semver"
)

const tagName = "puppet"

func FieldName(c eval.Context, f *reflect.StructField) string {
	if tagHash, ok := TagHash(c, f); ok {
		if nv, ok := tagHash.Get4(`name`); ok {
			return nv.String()
		}
	}
	return utils.CamelToSnakeCase(f.Name)
}

func TagHash(c eval.Context, f *reflect.StructField) (eval.KeyedValue, bool) {
	if tag := f.Tag.Get(tagName); tag != `` {
		tagExpr := c.ParseAndValidate(``, `{`+tag+`}`, true)
		if tagHash, ok := c.Evaluate(tagExpr).(eval.KeyedValue); ok {
			return tagHash, true
		}
	}
	return nil, false
}

func TypeSetFromReflect(c eval.Context, typeSetName string, version semver.Version, structTypes ...reflect.Type) *TypeSetType {
	types := make([]*HashEntry, 0)
	for _, structType := range structTypes {
		var parent eval.PType
		structType = checkStructType(c, structType)
		nf := structType.NumField()
		if nf > 0 {
			f := structType.Field(0)
			if f.Anonymous && f.Type.Kind() == reflect.Struct {
				parent = NewTypeReferenceType(typeSetName + `::` + f.Type.Name())
			}
		}
		types = append(types, WrapHashEntry2(
			structType.Name(),
			ObjectTypeFromReflect(c, typeSetName + `::` + structType.Name(), parent, structType)))
	}

	es := make([]*HashEntry, 0)
	es = append(es, WrapHashEntry2(eval.KEY_PCORE_URI, WrapString(string(eval.PCORE_URI))))
	es = append(es, WrapHashEntry2(eval.KEY_PCORE_VERSION, WrapSemVer(eval.PCORE_VERSION)))
	es = append(es, WrapHashEntry2(KEY_VERSION, WrapSemVer(version)))
	es = append(es, WrapHashEntry2(KEY_TYPES, WrapHash(types)))
	return NewTypeSetType(eval.RUNTIME_NAME_AUTHORITY, typeSetName, WrapHash(es))
}

func checkStructType(c eval.Context, t reflect.Type) reflect.Type {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		panic(eval.Error(c, eval.EVAL_ATTEMPT_TO_SET_WRONG_KIND, issue.H{`expected`: `Struct`, `actual`: t.Kind().String()}))
	}
	return t
}

func ObjectTypeFromReflect(c eval.Context, typeName string, parent eval.PType, structType reflect.Type) eval.ObjectType {
	structType = checkStructType(c, structType)
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

		if fh, ok := TagHash(c, &f); ok {
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
			typ = eval.WrapType(c, f.Type)
		}
		// Convenience. If a value is declared as being undef, then make the
		// type Optional if undef isn't an acceptable value
		if val != nil && eval.Equals(val, eval.UNDEF) {
			if !typ.IsInstance(c, eval.UNDEF, nil) {
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
	c.ImplementationRegistry().RegisterType(c, ot, structType)
	return ot
}
