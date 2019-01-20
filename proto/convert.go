package proto

import (
	"github.com/lyraproj/data-protobuf/datapb"
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/serialization"
	"github.com/lyraproj/puppet-evaluator/types"
)

// A ProtoConsumer consumes values and produces a datapb.Data
type ProtoConsumer interface {
	serialization.ValueConsumer

	// Value returns the created value. Must not be called until the consumption
	// of values is complete.
	Value() *datapb.Data
}

type protoConsumer struct {
	stack [][]*datapb.Data
}

// NewProtoConsumer creates a new ProtoConsumer
func NewProtoConsumer() ProtoConsumer {
	return &protoConsumer{stack: make([][]*datapb.Data, 1, 8)}
}

func (pc *protoConsumer) CanDoBinary() bool {
	return true
}

func (pc *protoConsumer) CanDoComplexKeys() bool {
	return true
}

func (pc *protoConsumer) StringDedupThreshold() int {
	return 0
}

func (pc *protoConsumer) AddArray(cap int, doer eval.Doer) {
	top := len(pc.stack)
	pc.stack = append(pc.stack, make([]*datapb.Data, 0, cap))
	doer()
	els := pc.stack[top]
	pc.stack = pc.stack[0:top]
	pc.add(&datapb.Data{Kind: &datapb.Data_ArrayValue{&datapb.DataArray{els}}})
}

func (pc *protoConsumer) AddHash(cap int, doer eval.Doer) {
	top := len(pc.stack)
	pc.stack = append(pc.stack, make([]*datapb.Data, 0, cap*2))
	doer()
	els := pc.stack[top]
	pc.stack = pc.stack[0:top]

	top = len(els)
	vals := make([]*datapb.DataEntry, top/2)
	for i := 0; i < top; i += 2 {
		vals[i/2] = &datapb.DataEntry{els[i], els[i+1]}
	}
	pc.add(&datapb.Data{Kind: &datapb.Data_HashValue{&datapb.DataHash{vals}}})
}

func (pc *protoConsumer) Add(v eval.Value) {
	pc.add(ToPBData(v))
}

func (pc *protoConsumer) AddRef(ref int) {
	pc.add(&datapb.Data{Kind: &datapb.Data_Reference{int64(ref)}})
}

func (pc *protoConsumer) Value() *datapb.Data {
	bs := pc.stack[0]
	if len(bs) > 0 {
		return bs[0]
	}
	return nil
}

func (pc *protoConsumer) add(value *datapb.Data) {
	top := len(pc.stack) - 1
	pc.stack[top] = append(pc.stack[top], value)
}

func ToPBData(v eval.Value) (value *datapb.Data) {
	switch v.(type) {
	case *types.BooleanValue:
		value = &datapb.Data{Kind: &datapb.Data_BooleanValue{v.(*types.BooleanValue).Bool()}}
	case *types.FloatValue:
		value = &datapb.Data{Kind: &datapb.Data_FloatValue{v.(*types.FloatValue).Float()}}
	case *types.IntegerValue:
		value = &datapb.Data{Kind: &datapb.Data_IntegerValue{v.(*types.IntegerValue).Int()}}
	case eval.StringValue:
		value = &datapb.Data{Kind: &datapb.Data_StringValue{v.String()}}
	case *types.UndefValue:
		value = &datapb.Data{Kind: &datapb.Data_UndefValue{}}
	case *types.ArrayValue:
		av := v.(*types.ArrayValue)
		vals := make([]*datapb.Data, av.Len())
		av.EachWithIndex(func(elem eval.Value, i int) {
			vals[i] = ToPBData(elem)
		})
		value = &datapb.Data{Kind: &datapb.Data_ArrayValue{&datapb.DataArray{vals}}}
	case *types.HashValue:
		av := v.(*types.HashValue)
		vals := make([]*datapb.DataEntry, av.Len())
		av.EachWithIndex(func(elem eval.Value, i int) {
			entry := elem.(*types.HashEntry)
			vals[i] = &datapb.DataEntry{ToPBData(entry.Key()), ToPBData(entry.Value())}
		})
		value = &datapb.Data{Kind: &datapb.Data_HashValue{&datapb.DataHash{vals}}}
	case *types.BinaryValue:
		value = &datapb.Data{Kind: &datapb.Data_BinaryValue{v.(*types.BinaryValue).Bytes()}}
	default:
		value = &datapb.Data{Kind: &datapb.Data_UndefValue{}}
	}
	return
}

// ConsumePBData converts a datapb.Data into stream of values that are sent to a
// serialization.ValueConsumer.
func ConsumePBData(v *datapb.Data, consumer serialization.ValueConsumer) {
	switch v.Kind.(type) {
	case *datapb.Data_BooleanValue:
		consumer.Add(types.WrapBoolean(v.GetBooleanValue()))
	case *datapb.Data_FloatValue:
		consumer.Add(types.WrapFloat(v.GetFloatValue()))
	case *datapb.Data_IntegerValue:
		consumer.Add(types.WrapInteger(v.GetIntegerValue()))
	case *datapb.Data_StringValue:
		consumer.Add(types.WrapString(v.GetStringValue()))
	case *datapb.Data_UndefValue:
		consumer.Add(eval.UNDEF)
	case *datapb.Data_ArrayValue:
		av := v.GetArrayValue().GetValues()
		consumer.AddArray(len(av), func() {
			for _, elem := range av {
				ConsumePBData(elem, consumer)
			}
		})
	case *datapb.Data_HashValue:
		av := v.GetHashValue().Entries
		consumer.AddHash(len(av), func() {
			for _, val := range av {
				ConsumePBData(val.Key, consumer)
				ConsumePBData(val.Value, consumer)
			}
		})
	case *datapb.Data_BinaryValue:
		consumer.Add(types.WrapBinary(v.GetBinaryValue()))
	case *datapb.Data_Reference:
		consumer.AddRef(int(v.GetReference()))
	default:
		consumer.Add(eval.UNDEF)
	}
	return
}

func FromPBData(v *datapb.Data) (value eval.Value) {
	switch v.Kind.(type) {
	case *datapb.Data_BooleanValue:
		value = types.WrapBoolean(v.GetBooleanValue())
	case *datapb.Data_FloatValue:
		value = types.WrapFloat(v.GetFloatValue())
	case *datapb.Data_IntegerValue:
		value = types.WrapInteger(v.GetIntegerValue())
	case *datapb.Data_StringValue:
		value = types.WrapString(v.GetStringValue())
	case *datapb.Data_UndefValue:
		value = eval.UNDEF
	case *datapb.Data_ArrayValue:
		av := v.GetArrayValue().GetValues()
		vals := make([]eval.Value, len(av))
		for i, elem := range av {
			vals[i] = FromPBData(elem)
		}
		value = types.WrapValues(vals)
	case *datapb.Data_HashValue:
		av := v.GetHashValue().Entries
		vals := make([]*types.HashEntry, len(av))
		for i, val := range av {
			vals[i] = types.WrapHashEntry(FromPBData(val.Key), FromPBData(val.Value))
		}
		value = types.WrapHash(vals)
	default:
		value = eval.UNDEF
	}
	return
}
