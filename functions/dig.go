package functions

import (
	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
	"github.com/lyraproj/puppet-evaluator/pdsl"
)

func init() {
	px.NewGoFunction(`dig`,
		func(d px.Dispatch) {
			d.Param(`Optional[Collection]`)
			d.RepeatedParam(`Any`)
			d.Function(func(c px.Context, args []px.Value) px.Value {
				walkedPath := make([]px.Value, 0)
				return types.WrapValues(args).Reduce(func(d, k px.Value) px.Value {
					if px.Undef.Equals(k, nil) {
						return px.Undef
					}
					switch d := d.(type) {
					case *types.UndefValue:
						return px.Undef
					case *types.Hash:
						walkedPath = append(walkedPath, k)
						return d.Get2(k, px.Undef)
					case *types.Array:
						walkedPath = append(walkedPath, k)
						if idx, ok := k.(px.Integer); ok {
							return d.At(int(idx.Int()))
						}
						return px.Undef
					default:
						panic(px.Error(pdsl.NotCollectionAt, issue.H{`walked_path`: types.WrapValues(walkedPath), `klass`: d.PType().String()}))
					}
				})
			})
		})
}
