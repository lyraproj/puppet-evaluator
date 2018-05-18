package resource

import (
	"fmt"
	"strings"

	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/types"
	"github.com/puppetlabs/go-issues/issue"
)

func initResourceFunctions() {
	eval.NewGoFunction(`create_resources`,
		func(d eval.Dispatch) {
			d.Param(`Variant[String,Type[Resource]]`)
			d.Param(`Hash[String,Hash[String,RichData]]`)
			d.OptionalParam(`Hash[String,RichData]`)
			d.Function(func(c eval.Context, args []eval.PValue) eval.PValue {
				typ := args[0]
				rType, ok := typ.(eval.ObjectType)
				if !ok {
					if ld, ok := eval.Load(c, eval.NewTypedName(eval.TYPE, strings.ToLower(typ.String()))); ok {
						rType = eval.AssertType(c, `type`, resourceType, ld.(eval.PType)).(eval.ObjectType)
					} else {
						panic(eval.Error(c, eval.EVAL_FAILURE, issue.H{`message`: fmt.Sprintf("'%s' is not a resource", ld)}))
					}
				}
				resources := args[1].(eval.KeyedValue)
				defaults := eval.EMPTY_MAP
				if len(args) > 2 {
					defaults = args[2].(eval.KeyedValue)
				} else {
					if dv, ok := resources.Get4(`_defaults`); ok {
						defaults = dv.(eval.KeyedValue)
						resources = resources.RejectPairs(func(k, v eval.PValue) bool {
							return k.String() == `_defaults`
						})
					}
				}

				location := c.StackTop()
				ctor := rType.Constructor()
				return resources.Map(func(ev eval.PValue) eval.PValue {
					entry := ev.(eval.EntryValue)
					rh := defaults.Merge(types.SingletonHash2(`title`, entry.Key()).Merge(entry.Value().(eval.KeyedValue)))
					rs := ctor.Call(c, nil, rh).(eval.PuppetObject)
					defineResource(c, rs, location)
					return rs
				})
			})
		})

	eval.NewGoFunction(`create_resource_types`,
		// Dispatch that expects a hash where each key is the name of the
		// resource type and the value is the attributes hash
		func(d eval.Dispatch) {
			d.Param(`Hash[String,Hash[String,RichData]]`)
			d.Function(func(c eval.Context, args []eval.PValue) eval.PValue {
				hv := args[0].(*types.HashValue)
				rts := make([]eval.PType, 0, hv.Len())
				hv.EachPair(func(k, v eval.PValue) {
					rhe := make([]*types.HashEntry, 3)
					rhe[0] = types.WrapHashEntry2(`name`, k)
					rhe[1] = types.WrapHashEntry2(`parent`, resourceType)
					rhe[2] = types.WrapHashEntry2(`attributes`, v)
					rts = append(rts, types.NewObjectType(``, nil, types.WrapHash(rhe)))
				})
				c.AddTypes(rts...)
				return eval.UNDEF
			})
		})

	eval.NewGoFunction(`get_resource`,
		func(d eval.Dispatch) {
			d.Param(`Variant[Type[Resource],String]`)
			d.Function(func(c eval.Context, args []eval.PValue) eval.PValue {
				ref := types.WrapString(Reference(c, args[0]))
				if r, ok := Resources(c).Get(ref); ok {
					return r
				}
				if node, ok := FindNode(c, ref); ok {
					return node.Resources().Get2(ref, eval.UNDEF)
				}
				return eval.UNDEF
			})
		},
	)
}
