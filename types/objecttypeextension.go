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

var ObjectTypeExtension_Type PType

func init() {
	ObjectTypeExtension_Type = newType(`ObjectTypeExtensionType`,
		`AnyType {]
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
	return ObjectTypeExtension_Type
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

// Checks that the given `paramValues` hash contains all keys present in the `parameters` of
// this instance and that each keyed value is a match for the given parameter. The match is done
// using case expression semantics.
//
// This method is only called when a given type is found to be assignable to the base type of
// this extension.
func (te *ObjectTypeExtension) testAssignable(paramValues *StringHash, g Guard) bool {
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
func (te *ObjectTypeExtension) testInstance(o PValue, g Guard) bool {
	return te.parameters.AllPair(func(key string, v1 interface{}) bool {
		v2, ok := te.baseType.GetValue(key, o)
		return ok && PuppetMatch(v2, v1.(PValue))
	})
}

/*
  #
  # @param o [Object] the instance to test
  # @param guard[RecursionGuard] guard against endless recursion
  # @return [Boolean] true or false to indicate if the value is an instance or not
  # @api public
  def test_instance?(o, guard)
    eval = Parser::EvaluatingParser.singleton.evaluator
    @parameters.keys.all? do |pn|
      begin
        m = o.public_method(pn)
        m.arity == 0 ? eval.match?(m.call, @parameters[pn]) : false
      rescue NameError
        false
      end
    end
  end


 */
