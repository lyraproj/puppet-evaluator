package yaml

import (
	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/types"
	ym "gopkg.in/yaml.v2"
)

func Unmarshal(c eval.Context, data []byte) eval.Value {
	ms := make(ym.MapSlice, 0)
	err := ym.Unmarshal([]byte(data), &ms)
	if err != nil {
		var itm interface{}
		err2 := ym.Unmarshal([]byte(data), &itm)
		if err2 != nil {
			panic(eval.Error(eval.ParseError, issue.H{`language`: `YAML`, `detail`: err.Error()}))
		}
		return wrapValue(c, itm)
	}
	return wrapSlice(c, ms)
}

func wrapSlice(c eval.Context, ms ym.MapSlice) eval.Value {
	es := make([]*types.HashEntry, len(ms))
	for i, me := range ms {
		es[i] = types.WrapHashEntry(wrapValue(c, me.Key), wrapValue(c, me.Value))
	}
	return types.WrapHash(es)
}

func wrapValue(c eval.Context, v interface{}) eval.Value {
	switch v := v.(type) {
	case ym.MapSlice:
		return wrapSlice(c, v)
	case []interface{}:
		vs := make([]eval.Value, len(v))
		for i, y := range v {
			vs[i] = wrapValue(c, y)
		}
		return types.WrapValues(vs)
	default:
		return eval.Wrap(c, v)
	}
}
