package evaluator

import (
	. "github.com/puppetlabs/go-evaluator/evaluator"
	"testing"
	"github.com/puppetlabs/go-pspec/pspec"
	"github.com/puppetlabs/go-parser/issue"
	"github.com/puppetlabs/go-evaluator/serialization"
	"github.com/puppetlabs/go-evaluator/types"
	"bytes"
	"strings"
)

func TestPSpecs(t *testing.T) {
	pspec.RunPspecTests(t, `testdata`)
}

func init() {
	NewGoFunction(`load_plan`,
		func(d Dispatch) {
			d.Param(`String`)
			d.Function(func(c EvalContext, args []PValue) PValue {
				planName := args[0].String()
				if plan, ok := Load(c.Loader(), NewTypedName(PLAN, planName)); ok {
					return WrapUnknown(plan)
				}
				panic(Error(EVAL_UNKNOWN_PLAN, issue.H{`name`: planName}))
			})
		})

	NewGoFunction(`load_task`,
		func(d Dispatch) {
			d.Param(`String`)
			d.Function(func(c EvalContext, args []PValue) PValue {
				taskName := args[0].String()
				if task, ok := Load(c.Loader(), NewTypedName(TASK, taskName)); ok {
					return task.(PValue)
				}
				panic(Error(EVAL_UNKNOWN_TASK, issue.H{`name`: taskName}))
			})
		})

	NewGoFunction(`to_symbol`,
		func(d Dispatch) {
			d.Param(`String`)
			d.Function(func(c EvalContext, args []PValue) PValue {
				return types.WrapRuntime(serialization.Symbol(args[0].String()))
			})
		})

	NewGoFunction(`to_data`,
		func(d Dispatch) {
			d.Param(`Any`)
			d.OptionalParam(
`Struct[
  Optional['type_by_reference'] => Boolean,
  Optional['local_reference'] => Boolean,
  Optional['symbol_as_string'] => Boolean,
  Optional['rich_data'] => Boolean,
  Optional['message_prefix'] => String
]`)
			d.Function(func(c EvalContext, args []PValue) PValue {
				options := EMPTY_MAP
				if len(args) > 1 {
					options = args[1].(KeyedValue)
				}
				return serialization.NewToDataConverter(options).Convert(args[0])
			})
		})

	NewGoFunction(`from_data`,
		func(d Dispatch) {
			d.Param(`Data`)
			d.OptionalParam(
				`Struct[
					Optional['allow_unresolved'] => Boolean
				]`)
			d.Function(func(c EvalContext, args []PValue) PValue {
				options := EMPTY_MAP
				if len(args) > 1 {
					options = args[1].(KeyedValue)
				}
				return serialization.NewFromDataConverter(c.Loader().(DefiningLoader), options).Convert(args[0])
			})
		})

	NewGoFunction(`data_to_json`,
		func(d Dispatch) {
			d.Param(`Data`)
			d.OptionalParam(
				`Struct[
					Optional['prefix'] => String,
					Optional['indent'] => String
				]`)
			d.Function(func(c EvalContext, args []PValue) PValue {
				options := EMPTY_MAP
				if len(args) > 1 {
					options = args[1].(KeyedValue)
				}
				out := bytes.NewBufferString(``)
				serialization.DataToJson(args[0], out, options)
				return types.WrapString(out.String())
			})
		})

	NewGoFunction(`json_to_data`,
		func(d Dispatch) {
			d.Param(`String`)
			d.Function(func(c EvalContext, args []PValue) PValue {
				return serialization.JsonToData(``, strings.NewReader(args[0].String()))
			})
		})
}