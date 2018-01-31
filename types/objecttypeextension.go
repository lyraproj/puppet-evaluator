package types

import (
	. "github.com/puppetlabs/go-evaluator/evaluator"
	. "github.com/puppetlabs/go-evaluator/hash"
	"github.com/puppetlabs/go-parser/issue"
	"io"
)

type objectTypeExtension struct {
	baseType   *objectType
	parameters *StringHash
}

var ObjectTypeExtension_Type PType

func init() {
	ObjectTypeExtension_Type = newType(`ObjectTypeExtensionType`,
		`AnyType {
			attributes => {
				base_type => Type,
				init_parameters => Array
			}
		}`)
}

func NewObjectTypeExtension(baseType ObjectType, initParameters []PValue) *objectTypeExtension {
	o := &objectTypeExtension{}
	o.initialize(baseType.(*objectType), initParameters)
	return o
}

func (te *objectTypeExtension) Accept(v Visitor, g Guard) {
	v(te)
	te.baseType.Accept(v, g)
}

func (te *objectTypeExtension) Default() PType {
	return te.baseType.Default()
}

func (te *objectTypeExtension) Equals(other interface{}, g Guard) bool {
	op, ok := other.(*objectTypeExtension)
	return ok && te.baseType.Equals(op.baseType, g) && te.parameters.Equals(op.parameters, g)
}

func (te *objectTypeExtension) Generalize() PType {
	return te.baseType
}

func (te *objectTypeExtension) IsAssignable(t PType, g Guard) bool {
	if ote, ok := t.(*objectTypeExtension); ok {
		return te.baseType.IsAssignable(ote.baseType, g) && te.testAssignable(ote.parameters, g)
	}
	if ot, ok := t.(*objectType); ok {
		return te.baseType.IsAssignable(ot, g) && te.testAssignable(EMPTY_STRINGHASH, g)
	}
	return false
}

func (te *objectTypeExtension) IsParameterized() bool {
	return true
}

func (te *objectTypeExtension) IsInstance(v PValue, g Guard) bool {
	return te.baseType.IsInstance(v, g) && te.testInstance(v, g)
}

func (te *objectTypeExtension) Member(name string) (CallableMember, bool) {
	return te.baseType.Member(name)
}

func (te *objectTypeExtension) Name() string {
	return te.baseType.Name()
}

func (te *objectTypeExtension) Parameters() []PValue {
	pts := te.baseType.typeParameters(true)
	n := pts.Size()
	if n > 2 {
		return []PValue{WrapHash5(te.parameters)}
	}
	params := make([]PValue, 0, n)
	top := 0
	idx := 0
	pts.EachKey(func(k string) {
		v, ok := te.parameters.Get3(k)
		if ok {
			top = idx + 1
		} else {
			v = WrapDefault()
		}
		params = append(params, v.(PValue))
		idx++
	})
	return params[:top]
}

func (te *objectTypeExtension) String() string {
	return ToString2(te, NONE)
}

func (te *objectTypeExtension) ToString(bld io.Writer, format FormatContext, g RDetect) {
	TypeToString(te, bld, format, g)
}

func (te *objectTypeExtension) Type() PType {
	return ObjectTypeExtension_Type
}

func (te *objectTypeExtension) initialize(baseType *objectType, initParameters []PValue) {
	pts := baseType.typeParameters(true)
	pvs := pts.Values()
	if pts.IsEmpty() {
		panic(Error(EVAL_NOT_PARAMETERIZED_TYPE, issue.H{`type`: baseType.Label()}))
	}
	te.baseType = baseType
	namedArgs := false
	if len(initParameters) == 1 {
		_, namedArgs = initParameters[0].(*HashValue)
	}

	if namedArgs {
		namedArgs = pts.Size() >= 1 && !IsInstance(pvs[0].(*typeParameter).Type(), initParameters[0])
	}

	checkParam := func(tp *typeParameter, v PValue) PValue {
		return AssertInstance(func() string { return tp.Label() }, tp.Type(), v)
	}

	byName := NewStringHash(pts.Size())
	if namedArgs {
		hash := initParameters[0].(*HashValue)
		hash.EachPair(func(k, pv PValue) {
			pn := k.String()
			tp := pts.Get(pn, nil)
			if tp == nil {
				panic(Error(EVAL_MISSING_TYPE_PARAMETER, issue.H{`name`: pn, `label`: baseType.Label()}))
			}
			if !Equals(pv, WrapDefault()) {
				byName.Put(pn, checkParam(tp.(*typeParameter), pv))
			}
		})
	} else {
		for idx, t := range pvs {
			if idx < len(initParameters) {
				tp := t.(*typeParameter)
				pv := initParameters[idx]
				if !Equals(pv, WrapDefault()) {
					byName.Put(tp.Name(), checkParam(tp, pv))
				}
			}
		}
	}
	if byName.IsEmpty() {
		panic(Error(EVAL_EMPTY_TYPE_PARAMETER_LIST, issue.H{`label`: baseType.Label()}))
	}
	te.parameters = byName
}

func (o *objectTypeExtension) AttributesInfo() AttributesInfo {
	return o.baseType.AttributesInfo()
}

// Checks that the given `paramValues` hash contains all keys present in the `parameters` of
// this instance and that each keyed value is a match for the given parameter. The match is done
// using case expression semantics.
//
// This method is only called when a given type is found to be assignable to the base type of
// this extension.
func (te *objectTypeExtension) testAssignable(paramValues *StringHash, g Guard) bool {
	// Default implementation performs case expression style matching of all parameter values
	// provided that the value exist (this should always be the case, since all defaults have
	// been assigned at this point)
	return te.parameters.AllPair(func(key string, v1 interface{}) bool {
		if v2, ok := paramValues.Get3(key); ok {
			a := v2.(PValue)
			b := v1.(PValue)
			if PuppetMatch(a, b) {
				return true
			}
			if at, ok := a.(PType); ok {
				if bt, ok := b.(PType); ok {
					return IsAssignable(bt, at)
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
func (te *objectTypeExtension) testInstance(o PValue, g Guard) bool {
	return te.parameters.AllPair(func(key string, v1 interface{}) bool {
		v2, ok := te.baseType.GetValue(key, o)
		return ok && PuppetMatch(v2, v1.(PValue))
	})
}
