package functions

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/types"
	"github.com/puppetlabs/go-parser/issue"
)

func init() {
	eval.NewGoFunction(`dig`,
		func(d eval.Dispatch) {
			d.Param(`Optional[Collection]`)
			d.RepeatedParam(`Any`)
			d.Function(func(c eval.EvalContext, args []eval.PValue) eval.PValue {
				walkedPath := []eval.PValue{}
				return types.WrapArray(args).Reduce(func(d, k eval.PValue) eval.PValue {
					if eval.Equals(eval.UNDEF, k) {
						return eval.UNDEF
					}
					switch d.(type) {
					case *types.UndefValue:
						return eval.UNDEF
					case *types.HashValue:
						walkedPath = append(walkedPath, k)
						return d.(*types.HashValue).Get2(k, eval.UNDEF)
					case *types.ArrayValue:
						walkedPath = append(walkedPath, k)
						if idx, ok := k.(*types.IntegerValue); ok {
							return d.(*types.ArrayValue).At(int(idx.Int()))
						}
						return eval.UNDEF
					default:
						panic(eval.Error(c, eval.EVAL_NOT_COLLECTION_AT, issue.H{`walked_path`: types.WrapArray(walkedPath), `klass`: d.Type().String()}))
					}
				})
			})
		})
}

