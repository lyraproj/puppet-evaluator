package types

import (
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/hash"
)

type typeParameter struct {
	attribute
}

var TypeTypeParameter = NewStructType([]*StructElement{
	newStructElement2(keyType, DefaultTypeType()),
	NewStructElement(newOptionalType3(keyAnnotations), typeAnnotations),
})

func (t *typeParameter) initHash() *hash.StringHash {
	h := t.attribute.initHash()
	h.Put(keyType, h.Get(keyType, nil).(*TypeType).PType())
	if v, ok := h.Get3(keyValue); ok && eval.Equals(v, undef) {
		h.Delete(keyValue)
	}
	return h
}

func (t *typeParameter) Equals(o interface{}, g eval.Guard) bool {
	if ot, ok := o.(*typeParameter); ok {
		return t.attribute.Equals(&ot.attribute, g)
	}
	return false
}

func (t *typeParameter) InitHash() eval.OrderedMap {
	return WrapStringPValue(t.initHash())
}

func (t *typeParameter) FeatureType() string {
	return `type_parameter`
}
