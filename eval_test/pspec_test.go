package eval

import (
	"bytes"
	"strings"
	"testing"

	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/serialization"
	"github.com/puppetlabs/go-evaluator/types"
	"github.com/puppetlabs/go-issues/issue"
	"github.com/puppetlabs/go-pspec/pspec"
)

func TestPSpecs(t *testing.T) {
	pspec.RunPspecTests(t, `testdata`, func() eval.DefiningLoader {
		eval.NewGoFunction(`load_plan`,
			func(d eval.Dispatch) {
				d.Param(`String`)
				d.Function(func(c eval.Context, args []eval.Value) eval.Value {
					planName := args[0].String()
					if plan, ok := eval.Load(c, eval.NewTypedName(eval.PLAN, planName)); ok {
						return eval.Wrap(nil, plan)
					}
					panic(eval.Error(eval.EVAL_UNKNOWN_PLAN, issue.H{`name`: planName}))
				})
			})

		eval.NewGoFunction(`load_task`,
			func(d eval.Dispatch) {
				d.Param(`String`)
				d.Function(func(c eval.Context, args []eval.Value) eval.Value {
					taskName := args[0].String()
					if task, ok := eval.Load(c, eval.NewTypedName(eval.TASK, taskName)); ok {
						return task.(eval.Value)
					}
					panic(eval.Error(eval.EVAL_UNKNOWN_TASK, issue.H{`name`: taskName}))
				})
			})

		eval.NewGoFunction(`to_symbol`,
			func(d eval.Dispatch) {
				d.Param(`String`)
				d.Function(func(c eval.Context, args []eval.Value) eval.Value {
					return types.WrapRuntime(serialization.Symbol(args[0].String()))
				})
			})

		eval.NewGoFunction(`to_data`,
			func(d eval.Dispatch) {
				d.Param(`Any`)
				d.OptionalParam(
					`Struct[
  Optional['type_by_reference'] => Boolean,
  Optional['local_reference'] => Boolean,
  Optional['symbol_as_string'] => Boolean,
  Optional['rich_data'] => Boolean,
  Optional['message_prefix'] => String
]`)
				d.Function(func(c eval.Context, args []eval.Value) eval.Value {
					options := eval.EMPTY_MAP
					if len(args) > 1 {
						options = args[1].(eval.OrderedMap)
					}
					return serialization.NewToDataConverter(options).Convert(args[0])
				})
			})

		eval.NewGoFunction(`from_data`,
			func(d eval.Dispatch) {
				d.Param(`Data`)
				d.OptionalParam(
					`Struct[
					Optional['allow_unresolved'] => Boolean
				]`)
				d.Function(func(c eval.Context, args []eval.Value) eval.Value {
					options := eval.EMPTY_MAP
					if len(args) > 1 {
						options = args[1].(eval.OrderedMap)
					}
					return serialization.NewFromDataConverter(c, options).Convert(args[0])
				})
			})

		eval.NewGoFunction(`data_to_json`,
			func(d eval.Dispatch) {
				d.Param(`Data`)
				d.OptionalParam(
					`Struct[
					Optional['prefix'] => String,
					Optional['indent'] => String
				]`)
				d.Function(func(c eval.Context, args []eval.Value) eval.Value {
					options := eval.EMPTY_MAP
					if len(args) > 1 {
						options = args[1].(eval.OrderedMap)
					}
					out := bytes.NewBufferString(``)
					serialization.DataToJson(c, args[0], out, options)
					return types.WrapString(out.String())
				})
			})

		eval.NewGoFunction(`json_to_data`,
			func(d eval.Dispatch) {
				d.Param(`String`)
				d.Function(func(c eval.Context, args []eval.Value) eval.Value {
					return serialization.JsonToData(c, ``, strings.NewReader(args[0].String()))
				})
			})

		return eval.StaticLoader().(eval.DefiningLoader)
	})
}
