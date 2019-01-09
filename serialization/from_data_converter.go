package serialization

import (
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/types"
)

// FromDataConverter is deprecated. Use Deserializer instead
type FromDataConverter struct {
	context      eval.Context
	deserializer Collector
}

// NewFromDataConverter is deprecated. Use NewDeserializer instead
func NewFromDataConverter(ctx eval.Context, options eval.OrderedMap) *FromDataConverter {
	return &FromDataConverter{context: ctx, deserializer: NewDeserializer(ctx, options)}
}

// Convert is deprecated. Use a Deserializer instead
func (f *FromDataConverter) Convert(value eval.Value) eval.Value {
	he := make([]*types.HashEntry, 0, 2)
	he = append(he, types.WrapHashEntry2(`rich_data`, types.Boolean_FALSE))
	he = append(he, types.WrapHashEntry2(`dedup_level`, types.WrapInteger(NoDedup)))
	NewSerializer(f.context, types.WrapHash(he)).Convert(value, f.deserializer)
	return f.deserializer.Value()
}
