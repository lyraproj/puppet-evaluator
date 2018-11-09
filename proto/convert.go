package proto

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/types"
	"github.com/puppetlabs/data-protobuf/datapb"
)

func ToPBData(v eval.Value) (value *datapb.Data) {
	switch v.(type) {
	case *types.BooleanValue:
		value = &datapb.Data{Kind: &datapb.Data_BooleanValue{v.(*types.BooleanValue).Bool()}}
	case *types.FloatValue:
		value = &datapb.Data{Kind: &datapb.Data_FloatValue{v.(*types.FloatValue).Float()}}
	case *types.IntegerValue:
		value = &datapb.Data{Kind: &datapb.Data_IntegerValue{v.(*types.IntegerValue).Int()}}
	case *types.StringValue:
		value = &datapb.Data{Kind: &datapb.Data_StringValue{v.(*types.StringValue).String()}}
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
			vals[i] = &datapb.DataEntry{entry.Key().String(), ToPBData(entry.Value())}
		})
		value = &datapb.Data{Kind: &datapb.Data_HashValue{&datapb.DataHash{vals}}}
	default:
		value = &datapb.Data{Kind: &datapb.Data_UndefValue{}}
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
			vals[i] = types.WrapHashEntry2(val.Key, FromPBData(val.Value))
		}
		value = types.WrapHash(vals)
	default:
		value = eval.UNDEF
	}
	return
}

