package types

import "github.com/puppetlabs/go-evaluator/eval"

type typeParameter struct {
	attribute
}

var TYPE_TYPE_PARAMETER = NewStructType([]*StructElement{
	NewStructElement2(KEY_TYPE, DefaultTypeType()),
	NewStructElement(NewOptionalType3(KEY_ANNOTATIONS), TYPE_ANNOTATIONS),
})

func (t *typeParameter) initHash() map[string]eval.PValue {
	hash := t.attribute.initHash()
	hash[KEY_TYPE] = hash[KEY_TYPE].(*TypeType).Type()
	if v, ok := hash[KEY_VALUE]; ok && eval.Equals(v, _UNDEF) {
		delete(hash, KEY_VALUE)
	}
	return hash
}

func (t *typeParameter) InitHash() eval.KeyedValue {
	return WrapHash3(t.initHash())
}

func (t *typeParameter) FeatureType() string {
	return `type_parameter`
}
