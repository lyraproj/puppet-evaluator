package serialization

import (
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/types"
)

// ToDataConverter is deprecated. Use Serializer instead
type ToDataConverter struct {
	serializer Serializer
}

var localRefSym = types.WrapString(PCORE_LOCAL_REF_SYMBOL)

// NewToDataConverter is deprecated. Use NewSerializer instead
func NewToDataConverter(options eval.OrderedMap) *ToDataConverter {
	return &ToDataConverter{serializer: NewSerializer(options)}
}

// Convert is deprecated. Use a Serializer instead
func (t *ToDataConverter) Convert(value eval.Value) eval.Value {
	cl := NewCollector()
	t.serializer.Convert(value, cl)
	return cl.Value()
}
