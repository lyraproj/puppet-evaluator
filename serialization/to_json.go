package serialization

import (
	"encoding/json"
	"io"

	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/types"
	"github.com/puppetlabs/go-parser/issue"
)

func DataToNative(c eval.Context, value eval.PValue) interface{} {
	switch value.(type) {
	case *types.IntegerValue:
		return value.(*types.IntegerValue).Int()
	case *types.FloatValue:
		return value.(*types.FloatValue).Float()
	case *types.BooleanValue:
		return value.(*types.BooleanValue).Bool()
	case *types.UndefValue:
		return nil
	case *types.StringValue:
		return value.String()
	case *types.ArrayValue:
		av := value.(*types.ArrayValue)
		result := make([]interface{}, av.Len())
		av.EachWithIndex(func(elem eval.PValue, idx int) { result[idx] = DataToNative(c, elem) })
		return result
	case *types.HashValue:
		hv := value.(*types.HashValue)
		result := make(map[string]interface{}, hv.Len())
		hv.EachPair(func(k, v eval.PValue) { result[assertString(c, k)] = DataToNative(c, v) })
		return result
	default:
		panic(eval.Error(c, eval.EVAL_TYPE_MISMATCH, issue.H{`detail`: eval.DescribeMismatch(``, types.DefaultDataType(), value.Type())}))
	}
}

func DataToJson(c eval.Context, value eval.PValue, out io.Writer, options eval.KeyedValue) {
	e := json.NewEncoder(out)
	prefix := options.Get5(`prefix`, eval.EMPTY_STRING).String()
	indent := options.Get5(`indent`, eval.EMPTY_STRING).String()
	if !(prefix == `` && indent == ``) {
		e.SetIndent(prefix, indent)
	}
	e.Encode(DataToNative(c, value))
}

func JsonToData(c eval.Context, path string, in io.Reader) eval.PValue {
	d := json.NewDecoder(in)
	d.UseNumber()
	var parsedValue interface{}
	err := d.Decode(&parsedValue)
	if err == nil {
		return NativeToData(parsedValue)
	}
	panic(eval.Error(c, eval.EVAL_TASK_BAD_JSON, issue.H{`path`: path, `detail`: err}))
}

func NativeToData(value interface{}) eval.PValue {
	return eval.WrapUnknown(value)
}

func assertString(c eval.Context, value eval.PValue) string {
	if s, ok := value.(*types.StringValue); ok {
		return s.String()
	}
	panic(eval.Error(c, eval.EVAL_TYPE_MISMATCH, issue.H{`detail`: eval.DescribeMismatch(``, types.DefaultStringType(), value.Type())}))
}
