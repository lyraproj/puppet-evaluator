package functions

import (
	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/types"
)

func init() {
	eval.NewGoFunction(`dig`,
		func(d eval.Dispatch) {
			d.Param(`Optional[Collection]`)
			d.RepeatedParam(`Any`)
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				walkedPath := make([]eval.Value, 0)
				return types.WrapValues(args).Reduce(func(d, k eval.Value) eval.Value {
					if eval.Equals(eval.Undef, k) {
						return eval.Undef
					}
					switch d := d.(type) {
					case *types.UndefValue:
						return eval.Undef
					case *types.HashValue:
						walkedPath = append(walkedPath, k)
						return d.Get2(k, eval.Undef)
					case *types.ArrayValue:
						walkedPath = append(walkedPath, k)
						if idx, ok := k.(eval.IntegerValue); ok {
							return d.At(int(idx.Int()))
						}
						return eval.Undef
					default:
						panic(eval.Error(eval.NotCollectionAt, issue.H{`walked_path`: types.WrapValues(walkedPath), `klass`: d.PType().String()}))
					}
				})
			})
		})
}
