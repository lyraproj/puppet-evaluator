package functions

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"gopkg.in/yaml.v2"
	"github.com/puppetlabs/go-evaluator/types"
	"github.com/puppetlabs/go-issues/issue"
)

func init() {
	eval.NewGoFunction(`parse_yaml`,
		func(d eval.Dispatch) {
			d.Param(`String`)
			d.Function(func(c eval.Context, args []eval.PValue) eval.PValue {
				return UnmarshalYaml(c, []byte(args[0].(*types.StringValue).String()))
			})
		},

		func(d eval.Dispatch) {
			d.Param(`Binary`)
			d.Function(func(c eval.Context, args []eval.PValue) eval.PValue {
				return UnmarshalYaml(c, args[0].(*types.BinaryValue).Bytes())
			})
		})
}

func UnmarshalYaml(c eval.Context, data []byte) eval.PValue {
	ms := make(yaml.MapSlice, 0)
	err := yaml.Unmarshal([]byte(data), &ms)
	if err != nil {
		var itm interface{}
		err2 := yaml.Unmarshal([]byte(data), &itm)
		if err2 != nil {
			panic(eval.Error(c, eval.EVAL_PARSE_ERROR, issue.H{`language`: `YAML`, `detail`: err.Error()}))
		}
		return eval.Wrap(itm)
	}
	return eval.Wrap(ms)
}
