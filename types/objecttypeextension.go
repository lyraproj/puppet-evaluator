package types

import (
	. "github.com/puppetlabs/go-evaluator/hash"
	. "github.com/puppetlabs/go-evaluator/evaluator"
	"github.com/puppetlabs/go-parser/issue"
	"io"
)

type ObjectTypeExtension struct {
	baseType *ObjectType
	parameters *StringHash
}

func init() {
	ObjectTypeExtensionType = newType(`ObjectTypeExtensionType`,
		`AnyType {
			attributes => {
				base_type => Type,
				init_parameters => Array
			}
		}`)
}

func NewObjectTypeExtension(baseType *ObjectType, initParameters []PValue) *ObjectTypeExtension {
  o := &ObjectTypeExtension{}
  o.initialize(baseType, initParameters)
  return o
}

func (te *ObjectTypeExtension) Accept(v Visitor, g Guard) {
	v(te)
	te.baseType.Accept(v, g)
}

func (te *ObjectTypeExtension) Default() PType {
	return te.baseType.Default()
}

func (te *ObjectTypeExtension) Equals(other interface{}, g Guard) bool {
	op, ok := other.(*ObjectTypeExtension)
	return ok && te.baseType.Equals(op.baseType, g) && te.parameters.Equals(op.parameters, g)
}

func (te *ObjectTypeExtension) Generalize() PType {
  return te.baseType
}

func (te *ObjectTypeExtension) IsAssignable(t PType, g Guard) bool {
	panic("implement me")
}

func (te *ObjectTypeExtension) IsInstance(v PValue, g Guard) bool {
	panic("implement me")
}

func (te *ObjectTypeExtension) Name() string {
	return te.baseType.Name()
}

func (te *ObjectTypeExtension) Parameters() []PValue {
	n := te.parameters.Size()
	if n > 2 {
		return []PValue{WrapHash5(te.parameters)}
	}
	params := make([]PValue, 0, n)
	te.parameters.EachValue(func(v interface{}) { params = append(params, v.(PValue)) })
	return params
}

func (te *ObjectTypeExtension) String() string {
	return ToString2(te, NONE)
}

func (te *ObjectTypeExtension) ToString(bld io.Writer, format FormatContext, g RDetect) {
	TypeToString(te, bld, format, g)
}

func (te *ObjectTypeExtension) Type() PType {
	return ObjectTypeExtensionType
}

func (te *ObjectTypeExtension) initialize(baseType *ObjectType, initParameters []PValue) {
	pts := baseType.typeParameters(true)
	pvs := pts.Values()
	if pts.IsEmpty() {
		Error(EVAL_NOT_PARAMETERIZED_TYPE, issue.H{`type`: baseType.Label()})
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
			tp := pts.Get(pn, nil).(*typeParameter)
			if tp == nil {
				Error(EVAL_MISSING_TYPE_PARAMETER, issue.H{`name`: pn, `label`: baseType.Label()})
			}
			if !Equals(pv, WrapDefault()) {
			  byName.Put(pn, checkParam(tp, pv))
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
		Error(EVAL_EMPTY_TYPE_PARAMETER_LIST, issue.H{`label`: baseType.Label()})
	}
	te.parameters = byName
}

var ObjectTypeExtensionType PType
