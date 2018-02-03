package eval

import (
	"bytes"
	"strings"
	"testing"

	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/serialization"
	"github.com/puppetlabs/go-evaluator/types"
	"github.com/puppetlabs/go-parser/issue"
	"github.com/puppetlabs/go-pspec/pspec"
)

func TestPSpecs(t *testing.T) {
	pspec.RunPspecTests(t, `testdata`)
}

func init() {
	eval.NewGoFunction(`load_plan`,
		func(d eval.Dispatch) {
			d.Param(`String`)
			d.Function(func(c eval.EvalContext, args []eval.PValue) eval.PValue {
				planName := args[0].String()
				if plan, ok := eval.Load(c.Loader(), eval.NewTypedName(eval.PLAN, planName)); ok {
					return eval.WrapUnknown(plan)
				}
				panic(eval.Error(eval.EVAL_UNKNOWN_PLAN, issue.H{`name`: planName}))
			})
		})

	eval.NewGoFunction(`load_task`,
		func(d eval.Dispatch) {
			d.Param(`String`)
			d.Function(func(c eval.EvalContext, args []eval.PValue) eval.PValue {
				taskName := args[0].String()
				if task, ok := eval.Load(c.Loader(), eval.NewTypedName(eval.TASK, taskName)); ok {
					return task.(eval.PValue)
				}
				panic(eval.Error(eval.EVAL_UNKNOWN_TASK, issue.H{`name`: taskName}))
			})
		})

	eval.NewGoFunction(`to_symbol`,
		func(d eval.Dispatch) {
			d.Param(`String`)
			d.Function(func(c eval.EvalContext, args []eval.PValue) eval.PValue {
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
			d.Function(func(c eval.EvalContext, args []eval.PValue) eval.PValue {
				options := eval.EMPTY_MAP
				if len(args) > 1 {
					options = args[1].(eval.KeyedValue)
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
			d.Function(func(c eval.EvalContext, args []eval.PValue) eval.PValue {
				options := eval.EMPTY_MAP
				if len(args) > 1 {
					options = args[1].(eval.KeyedValue)
				}
				return serialization.NewFromDataConverter(c.Loader().(eval.DefiningLoader), options).Convert(args[0])
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
			d.Function(func(c eval.EvalContext, args []eval.PValue) eval.PValue {
				options := eval.EMPTY_MAP
				if len(args) > 1 {
					options = args[1].(eval.KeyedValue)
				}
				out := bytes.NewBufferString(``)
				serialization.DataToJson(args[0], out, options)
				return types.WrapString(out.String())
			})
		})

	eval.NewGoFunction(`json_to_data`,
		func(d eval.Dispatch) {
			d.Param(`String`)
			d.Function(func(c eval.EvalContext, args []eval.PValue) eval.PValue {
				return serialization.JsonToData(``, strings.NewReader(args[0].String()))
			})
		})
}
