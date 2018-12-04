package functions

import (
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/types"
	"github.com/lyraproj/issue/issue"
)

func init() {
	eval.NewGoFunction(`dig`,
		func(d eval.Dispatch) {
			d.Param(`Optional[Collection]`)
			d.RepeatedParam(`Any`)
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				walkedPath := []eval.Value{}
				return types.WrapValues(args).Reduce(func(d, k eval.Value) eval.Value {
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
						panic(eval.Error(eval.EVAL_NOT_COLLECTION_AT, issue.H{`walked_path`: types.WrapValues(walkedPath), `klass`: d.PType().String()}))
					}
				})
			})
		})
}
