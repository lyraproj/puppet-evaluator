package proto

import (
	"github.com/lyraproj/data-protobuf/datapb"
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/types"
)

// A Consumer consumes values and produces a datapb.Data
type Consumer interface {
	eval.ValueConsumer

	// Value returns the created value. Must not be called until the consumption
	// of values is complete.
	Value() *datapb.Data
}

type protoConsumer struct {
	stack [][]*datapb.Data
}

// NewProtoConsumer creates a new Consumer
func NewProtoConsumer() Consumer {
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
	pc.add(&datapb.Data{Kind: &datapb.Data_ArrayValue{ArrayValue: &datapb.DataArray{Values: els}}})
}

func (pc *protoConsumer) AddHash(cap int, doer eval.Doer) {
	top := len(pc.stack)
	pc.stack = append(pc.stack, make([]*datapb.Data, 0, cap*2))
	doer()
	els := pc.stack[top]
	pc.stack = pc.stack[0:top]

	top = len(els)
	vs := make([]*datapb.DataEntry, top/2)
	for i := 0; i < top; i += 2 {
		vs[i/2] = &datapb.DataEntry{Key: els[i], Value: els[i+1]}
	}
	pc.add(&datapb.Data{Kind: &datapb.Data_HashValue{HashValue: &datapb.DataHash{Entries: vs}}})
}

func (pc *protoConsumer) Add(v eval.Value) {
	pc.add(ToPBData(v))
}

func (pc *protoConsumer) AddRef(ref int) {
	pc.add(&datapb.Data{Kind: &datapb.Data_Reference{Reference: int64(ref)}})
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
	switch v := v.(type) {
	case eval.BooleanValue:
		value = &datapb.Data{Kind: &datapb.Data_BooleanValue{BooleanValue: v.Bool()}}
	case eval.FloatValue:
		value = &datapb.Data{Kind: &datapb.Data_FloatValue{FloatValue: v.Float()}}
	case eval.IntegerValue:
		value = &datapb.Data{Kind: &datapb.Data_IntegerValue{IntegerValue: v.Int()}}
	case eval.StringValue:
		value = &datapb.Data{Kind: &datapb.Data_StringValue{StringValue: v.String()}}
	case *types.UndefValue:
		value = &datapb.Data{Kind: &datapb.Data_UndefValue{}}
	case *types.ArrayValue:
		vs := make([]*datapb.Data, v.Len())
		v.EachWithIndex(func(elem eval.Value, i int) {
			vs[i] = ToPBData(elem)
		})
		value = &datapb.Data{Kind: &datapb.Data_ArrayValue{ArrayValue: &datapb.DataArray{Values: vs}}}
	case *types.HashValue:
		vs := make([]*datapb.DataEntry, v.Len())
		v.EachWithIndex(func(elem eval.Value, i int) {
			entry := elem.(*types.HashEntry)
			vs[i] = &datapb.DataEntry{Key: ToPBData(entry.Key()), Value: ToPBData(entry.Value())}
		})
		value = &datapb.Data{Kind: &datapb.Data_HashValue{HashValue: &datapb.DataHash{Entries: vs}}}
	case *types.BinaryValue:
		value = &datapb.Data{Kind: &datapb.Data_BinaryValue{BinaryValue: v.Bytes()}}
	default:
		value = &datapb.Data{Kind: &datapb.Data_UndefValue{}}
	}
	return
}

// ConsumePBData converts a datapb.Data into stream of values that are sent to a
// serialization.ValueConsumer.
func ConsumePBData(v *datapb.Data, consumer eval.ValueConsumer) {
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
		consumer.Add(eval.Undef)
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
		consumer.Add(eval.Undef)
	}
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
		value = eval.Undef
	case *datapb.Data_ArrayValue:
		av := v.GetArrayValue().GetValues()
		vs := make([]eval.Value, len(av))
		for i, elem := range av {
			vs[i] = FromPBData(elem)
		}
		value = types.WrapValues(vs)
	case *datapb.Data_HashValue:
		av := v.GetHashValue().Entries
		vs := make([]*types.HashEntry, len(av))
		for i, val := range av {
			vs[i] = types.WrapHashEntry(FromPBData(val.Key), FromPBData(val.Value))
		}
		value = types.WrapHash(vs)
	default:
		value = eval.Undef
	}
	return
}
