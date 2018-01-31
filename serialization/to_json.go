package serialization

import (
	"encoding/json"
	. "github.com/puppetlabs/go-evaluator/evaluator"
	. "github.com/puppetlabs/go-evaluator/types"
	"github.com/puppetlabs/go-parser/issue"
	"io"
)

func DataToNative(value PValue) interface{} {
	switch value.(type) {
	case *IntegerValue:
		return value.(*IntegerValue).Int()
	case *FloatValue:
		return value.(*FloatValue).Float()
	case *BooleanValue:
		return value.(*BooleanValue).Bool()
	case *UndefValue:
		return nil
	case *StringValue:
		return value.String()
	case *ArrayValue:
		av := value.(*ArrayValue)
		result := make([]interface{}, av.Len())
		av.EachWithIndex(func(elem PValue, idx int) { result[idx] = DataToNative(elem) })
		return result
	case *HashValue:
		hv := value.(*HashValue)
		result := make(map[string]interface{}, hv.Len())
		hv.EachPair(func(k, v PValue) { result[assertString(k)] = DataToNative(v) })
		return result
	default:
		panic(Error(EVAL_TYPE_MISMATCH, issue.H{`detail`: DescribeMismatch(``, DefaultDataType(), value.Type())}))
	}
}

func DataToJson(value PValue, out io.Writer, options KeyedValue) {
	e := json.NewEncoder(out)
	prefix := options.Get5(`prefix`, EMPTY_STRING).String()
	indent := options.Get5(`indent`, EMPTY_STRING).String()
	if !(prefix == `` && indent == ``) {
		e.SetIndent(prefix, indent)
	}
	e.Encode(DataToNative(value))
}

func JsonToData(path string, in io.Reader) PValue {
	d := json.NewDecoder(in)
	d.UseNumber()
	var parsedValue interface{}
	err := d.Decode(&parsedValue)
	if err == nil {
		return NativeToData(parsedValue)
	}
	panic(Error(EVAL_TASK_BAD_JSON, issue.H{`path`: path, `detail`: err}))
}

func NativeToData(value interface{}) PValue {
	return WrapUnknown(value)
}

func assertString(value PValue) string {
	if s, ok := value.(*StringValue); ok {
		return s.String()
	}
	panic(Error(EVAL_TYPE_MISMATCH, issue.H{`detail`: DescribeMismatch(``, DefaultStringType(), value.Type())}))
}
