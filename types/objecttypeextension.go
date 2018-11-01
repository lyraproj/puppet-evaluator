package types

import (
	"io"

	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/hash"
	"github.com/puppetlabs/go-issues/issue"
	"reflect"
)

type objectTypeExtension struct {
	baseType   *objectType
	parameters *hash.StringHash
}

var ObjectTypeExtension_Type eval.ObjectType

func init() {
	ObjectTypeExtension_Type = newObjectType(`Pcore::ObjectTypeExtensionType`,
		`Pcore::AnyType {
			attributes => {
				base_type => Type,
				init_parameters => Array
			}
		}`)
}

func NewObjectTypeExtension(c eval.Context, baseType eval.ObjectType, initParameters []eval.Value) *objectTypeExtension {
	o := &objectTypeExtension{}
	o.initialize(c, baseType.(*objectType), initParameters)
	return o
}

func (te *objectTypeExtension) Accept(v eval.Visitor, g eval.Guard) {
	v(te)
	te.baseType.Accept(v, g)
}

func (te *objectTypeExtension) Constructor() eval.Function {
	return te.baseType.Constructor()
}

func (te *objectTypeExtension) Default() eval.Type {
	return te.baseType.Default()
}

func (te *objectTypeExtension) Equals(other interface{}, g eval.Guard) bool {
	op, ok := other.(*objectTypeExtension)
	return ok && te.baseType.Equals(op.baseType, g) && te.parameters.Equals(op.parameters, g)
}

func (te *objectTypeExtension) Generalize() eval.Type {
	return te.baseType
}

func (te *objectTypeExtension) Get(key string) (eval.Value, bool) {
	return te.baseType.Get(key)
}

func (te *objectTypeExtension) HasHashConstructor() bool {
	return te.baseType.HasHashConstructor()
}

func (te *objectTypeExtension) IsAssignable(t eval.Type, g eval.Guard) bool {
	if ote, ok := t.(*objectTypeExtension); ok {
		return te.baseType.IsAssignable(ote.baseType, g) && te.testAssignable(ote.parameters, g)
	}
	if ot, ok := t.(*objectType); ok {
		return te.baseType.IsAssignable(ot, g) && te.testAssignable(hash.EMPTY_STRINGHASH, g)
	}
	return false
}

func (te *objectTypeExtension) IsMetaType() bool {
	return te.baseType.IsMetaType()
}

func (te *objectTypeExtension) IsParameterized() bool {
	return true
}

func (te *objectTypeExtension) IsInstance(v eval.Value, g eval.Guard) bool {
	return te.baseType.IsInstance(v, g) && te.testInstance(v, g)
}

func (te *objectTypeExtension) Member(name string) (eval.CallableMember, bool) {
	return te.baseType.Member(name)
}

func (t *objectTypeExtension) MetaType() eval.ObjectType {
	return ObjectTypeExtension_Type
}

func (te *objectTypeExtension) Name() string {
	return te.baseType.Name()
}

func (te *objectTypeExtension) Parameters() []eval.Value {
	pts := te.baseType.typeParameters(true)
	n := pts.Len()
	if n > 2 {
		return []eval.Value{WrapStringPValue(te.parameters)}
	}
	params := make([]eval.Value, 0, n)
	top := 0
	idx := 0
	pts.EachKey(func(k string) {
		v, ok := te.parameters.Get3(k)
		if ok {
			top = idx + 1
		} else {
			v = WrapDefault()
		}
		params = append(params, v.(eval.Value))
		idx++
	})
	return params[:top]
}

func (te *objectTypeExtension) FromReflectedValue(c eval.Context, src reflect.Value) eval.PuppetObject {
	return te.baseType.FromReflectedValue(c, src)
}

func (te *objectTypeExtension) Parent() eval.Type {
	return te.baseType.Parent()
}

func (te *objectTypeExtension) ToReflectedValue(c eval.Context, src eval.PuppetObject, dest reflect.Value) {
	te.baseType.ToReflectedValue(c, src, dest)
}

func (te *objectTypeExtension) String() string {
	return eval.ToString2(te, NONE)
}

func (te *objectTypeExtension) ToString(bld io.Writer, format eval.FormatContext, g eval.RDetect) {
	TypeToString(te, bld, format, g)
}

func (te *objectTypeExtension) PType() eval.Type {
	return &TypeType{te}
}

func (te *objectTypeExtension) initialize(c eval.Context, baseType *objectType, initParameters []eval.Value) {
	pts := baseType.typeParameters(true)
	pvs := pts.Values()
	if pts.IsEmpty() {
		panic(eval.Error(eval.EVAL_NOT_PARAMETERIZED_TYPE, issue.H{`type`: baseType.Label()}))
	}
	te.baseType = baseType
	namedArgs := false
	if len(initParameters) == 1 {
		_, namedArgs = initParameters[0].(*HashValue)
	}

	if namedArgs {
		namedArgs = pts.Len() >= 1 && !eval.IsInstance(pvs[0].(*typeParameter).PType(), initParameters[0])
	}

	checkParam := func(tp *typeParameter, v eval.Value) eval.Value {
		return eval.AssertInstance(func() string { return tp.Label() }, tp.PType(), v)
	}

	byName := hash.NewStringHash(pts.Len())
	if namedArgs {
		hash := initParameters[0].(*HashValue)
		hash.EachPair(func(k, pv eval.Value) {
			pn := k.String()
			tp := pts.Get(pn, nil)
			if tp == nil {
				panic(eval.Error(eval.EVAL_MISSING_TYPE_PARAMETER, issue.H{`name`: pn, `label`: baseType.Label()}))
			}
			if !eval.Equals(pv, WrapDefault()) {
				byName.Put(pn, checkParam(tp.(*typeParameter), pv))
			}
		})
	} else {
		for idx, t := range pvs {
			if idx < len(initParameters) {
				tp := t.(*typeParameter)
				pv := initParameters[idx]
				if !eval.Equals(pv, WrapDefault()) {
					byName.Put(tp.Name(), checkParam(tp, pv))
				}
			}
		}
	}
	if byName.IsEmpty() {
		panic(eval.Error(eval.EVAL_EMPTY_TYPE_PARAMETER_LIST, issue.H{`label`: baseType.Label()}))
	}
	te.parameters = byName
}

func (te *objectTypeExtension) AttributesInfo() eval.AttributesInfo {
	return te.baseType.AttributesInfo()
}

// Checks that the given `paramValues` hash contains all keys present in the `parameters` of
// this instance and that each keyed value is a match for the given parameter. The match is done
// using case expression semantics.
//
// This method is only called when a given type is found to be assignable to the base type of
// this extension.
func (te *objectTypeExtension) testAssignable(paramValues *hash.StringHash, g eval.Guard) bool {
	// Default implementation performs case expression style matching of all parameter values
	// provided that the value exist (this should always be the case, since all defaults have
	// been assigned at this point)
	return te.parameters.AllPair(func(key string, v1 interface{}) bool {
		if v2, ok := paramValues.Get3(key); ok {
			a := v2.(eval.Value)
			b := v1.(eval.Value)
			if eval.PuppetMatch(nil, a, b) {
				return true
			}
			if at, ok := a.(eval.Type); ok {
				if bt, ok := b.(eval.Type); ok {
					return eval.IsAssignable(bt, at)
				}
			}
		}
		return false
	})
}

// Checks that the given instance `o` has one attribute for each key present in the `parameters` of
// this instance and that each attribute value is a match for the given parameter. The match is done
// using case expression semantics.
//
// This method is only called when the given value is found to be an instance of the base type of
// this extension.
func (te *objectTypeExtension) testInstance(o eval.Value, g eval.Guard) bool {
	c := eval.CurrentContext()
	return te.parameters.AllPair(func(key string, v1 interface{}) bool {
		v2, ok := te.baseType.GetValue(key, o)
		return ok && eval.PuppetMatch(c, v2, v1.(eval.Value))
	})
}
