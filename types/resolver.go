package types

import (
	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/puppet-evaluator/eval"
)

var coreTypes map[string]eval.Type

func Resolve(c eval.Context, qref string) eval.Type {
	pt := coreTypes[qref]
	if pt != nil {
		return pt
	}
	return loadType(c, qref)
}

func ResolveWithParams(c eval.Context, name string, args []eval.Value) eval.Type {
	t := Resolve(c, name)
	if oo, ok := t.(eval.ObjectType); ok && oo.IsParameterized() {
		return NewObjectTypeExtension(c, oo, args)
	}
	if pt, ok := t.(eval.ParameterizedType); ok {
		mt := pt.MetaType().(*objectType)
		if mt.creators != nil {
			if posCtor := mt.creators[0]; posCtor != nil {
				return posCtor(c, args).(eval.Type)
			}
		}
	}
	panic(eval.Error(eval.EVAL_NOT_PARAMETERIZED_TYPE, issue.H{`type`: name}))
}

func loadType(c eval.Context, name string) eval.Type {
	tn := newTypedName2(eval.NsType, name, c.Loader().NameAuthority())
	found, ok := eval.Load(c, tn)
	if ok {
		return found.(eval.Type)
	}
	return NewTypeReferenceType(name)
}
