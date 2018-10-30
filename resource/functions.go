package resource

import (
	"strings"

	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/types"
	"github.com/puppetlabs/go-evaluator/yaml2ast"
	"github.com/puppetlabs/go-issues/issue"
)

func createResources(c eval.Context, typ eval.Value, resources eval.OrderedMap, defaults eval.OrderedMap) eval.Value {
	rType, ok := typ.(eval.ObjectType)
	if !ok {
		if ld, ok := eval.Load(c, eval.NewTypedName(eval.TYPE, strings.ToLower(typ.String()))); ok {
			rType = eval.AssertType(`type`, resourceType, ld.(eval.Type)).(eval.ObjectType)
		} else {
			panic(eval.Error(eval.EVAL_UNRESOLVED_TYPE, issue.H{`typeString`: typ}))
		}
	}

	if defaults == nil {
		if dv, ok := resources.Get4(`_defaults`); ok {
			defaults = dv.(eval.OrderedMap)
			resources = resources.RejectPairs(func(k, v eval.Value) bool {
				return k.String() == `_defaults`
			})
		} else {
			defaults = eval.EMPTY_MAP
		}
	}

	location := c.StackTop()
	ctor := rType.Constructor()
	return resources.Map(func(ev eval.Value) eval.Value {
		entry := ev.(eval.MapEntry)
		rh := defaults.Merge(types.SingletonHash2(`title`, entry.Key()).Merge(entry.Value().(eval.OrderedMap)))
		rs := ctor.Call(c, nil, rh).(eval.PuppetObject)
		defineResource(c, rs, location)
		return rs
	})
}

func initResourceFunctions() {
	eval.NewGoFunction(`create_resources`,
		func(d eval.Dispatch) {
			d.Param(`Variant[String,Type[Resource]]`)
			d.Param(`Hash[String,Hash[String,RichData]]`)
			d.OptionalParam(`Hash[String,RichData]`)
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				var defaults eval.OrderedMap
				if len(args) > 2 {
					defaults = args[2].(eval.OrderedMap)
				}
				return createResources(c, args[0], args[1].(eval.OrderedMap), defaults)
			})
		})

	eval.NewGoFunction(`resources`,
		func(d eval.Dispatch) {
			d.Param(`Hash[Variant[String,Type[Resource]], Hash[String,Hash[String,RichData]]]`)
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				ha := args[0].(*types.HashValue)
				rs := make([]*types.HashEntry, ha.Len())
				ha.EachPair(func(k, v eval.Value) {
					rs = append(rs, types.WrapHashEntry(k, createResources(c, k, v.(eval.OrderedMap), nil)))
				})
				return types.WrapHash(rs)
			})
		})

	eval.NewGoFunction(`create_resource_types`,
		// Dispatch that expects a hash where each key is the name of the
		// resource type and the value is the attributes hash
		func(d eval.Dispatch) {
			d.Param(`Hash[String,Hash[String,RichData]]`)
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				hv := args[0].(*types.HashValue)
				rts := make([]eval.Type, 0, hv.Len())
				hv.EachPair(func(k, v eval.Value) {
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

	eval.NewGoFunction(`create_types`,
		// Dispatch that expects a hash where each key is the name of the
		// resource type and the value is the attributes hash
		func(d eval.Dispatch) {
			d.Param(`Hash[String,Hash[String,RichData]]`)
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				hv := args[0].(*types.HashValue)
				rts := make([]eval.Type, 0, hv.Len())
				hv.EachPair(func(k, v eval.Value) {
					rhe := make([]*types.HashEntry, 2)
					rhe[0] = types.WrapHashEntry2(`name`, k)
					rhe[1] = types.WrapHashEntry2(`attributes`, v)
					rts = append(rts, types.NewObjectType(``, nil, types.WrapHash(rhe)))
				})
				c.AddTypes(rts...)
				return eval.UNDEF
			})
		})

	eval.NewGoFunction(`resource`,
		func(d eval.Dispatch) {
			d.Param(`Variant[Type[Resource],String]`)
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				ref := types.WrapString(Reference(args[0]))
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

	eval.NewGoFunction(`reference`,
		func(d eval.Dispatch) {
			d.Param(`Resource`)
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				return types.WrapString(Reference(args[0]))
			})
		},
	)

	eval.NewGoFunction(`evaluate_yaml`,
		func(d eval.Dispatch) {
			d.Param(`String`)
			d.Param(`Binary`)
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				return yaml2ast.EvaluateYaml(c, args[0].String(), args[1].(*types.BinaryValue).Bytes())
			})
		},
		func(d eval.Dispatch) {
			d.Param(`String`)
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				path := args[0].String()
				return yaml2ast.EvaluateYaml(c, path, types.BinaryFromFile(c, path).Bytes())
			})
		})
}
