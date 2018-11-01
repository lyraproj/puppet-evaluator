package types

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/hash"
)

type typeParameter struct {
	attribute
}

var TYPE_TYPE_PARAMETER = NewStructType([]*StructElement{
	NewStructElement2(KEY_TYPE, DefaultTypeType()),
	NewStructElement(NewOptionalType3(KEY_ANNOTATIONS), TYPE_ANNOTATIONS),
})

func (t *typeParameter) initHash() *hash.StringHash {
	hash := t.attribute.initHash()
	hash.Put(KEY_TYPE, hash.Get(KEY_TYPE, nil).(*TypeType).PType())
	if v, ok := hash.Get3(KEY_VALUE); ok && eval.Equals(v, _UNDEF) {
		hash.Delete(KEY_VALUE)
	}
	return hash
}

func (t *typeParameter) InitHash() eval.OrderedMap {
	return WrapStringPValue(t.initHash())
}

func (t *typeParameter) FeatureType() string {
	return `type_parameter`
}
