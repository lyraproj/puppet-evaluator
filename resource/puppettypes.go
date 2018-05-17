package resource

import (
	"github.com/puppetlabs/go-evaluator/eval"
	_ "github.com/puppetlabs/go-evaluator/loader"
	"github.com/puppetlabs/go-evaluator/types"
	"strings"
	"fmt"
	"github.com/puppetlabs/go-issues/issue"
)

var builtinResourceTypes = [...]string{
	`Augeas`,
	`Component`,
	`Computer`,
	`Cron`,
	`Exec`,
	`File`,
	`Filebucket`,
	`Group`,
	`Host`,
	`Interface`,
	`K5login`,
	`Macauthorization`,
	`Mailalias`,
	`Maillist`,
	`Mcx`,
	`Mount`,
	`Nagios_command`,
	`Nagios_contact`,
	`Nagios_contactgroup`,
	`Nagios_host`,
	`Nagios_hostdependency`,
	`Nagios_hostescalation`,
	`Nagios_hostgroup`,
	`Nagios_hostextinfo`,
	`Nagios_service`,
	`Nagios_servicedependency`,
	`Nagios_serviceescalation`,
	`Nagios_serviceextinfo`,
	`Nagios_servicegroup`,
	`Nagios_timeperiod`,
	`Node`,
	`Notify`,
	`Package`,
	`Resources`,
	`Router`,
	`Schedule`,
	`Scheduled_task`,
	`Selboolean`,
	`Selmodule`,
	`Service`,
	`Ssh_authorized_key`,
	`Sshkey`,
	`Stage`,
	`Tidy`,
	`User`,
	`Vlan`,
	`Whit`,
	`Yumrepo`,
	`Zfs`,
	`Zone`,
	`Zpool`,
}

var resourceType, resultType, resultSetType eval.PType

func InitBuiltinResources() {
	resourceType = eval.NewObjectType(`Resource`, `{
    type_parameters => {
      title => String[1]
    },
    attributes => {
      title => String[1]
    }}`)

	resultType = eval.NewObjectType(`Result`, `{
    attributes => {
      'id' => ScalarData,
      'value' => { type => RichData, value => undef },
      'message' => { type => Optional[String], value => undef },
      'error' => { type => Boolean, kind => derived },
      'ok' => { type => Boolean, kind => derived }
    }
  }`, func(ctx eval.Context, args []eval.PValue) eval.PValue {
		return NewResult2(args...)
	}, func(ctx eval.Context, args []eval.PValue) eval.PValue {
		return NewResultFromHash(args[0].(*types.HashValue))
	})

	resultSetType = eval.NewObjectType(`ResultSet`, `{
    attributes => {
      'results' => Array[Result],
    },
    functions => {
      count => Callable[[], Integer],
      empty => Callable[[], Boolean],
      error_set => Callable[[], ResultSet],
      first => Callable[[], Optional[Result]],
      ids => Callable[[], Array[ScalarData]],
      ok => Callable[[], Boolean],
      ok_set => Callable[[], ResultSet],
      '[]' => Callable[[Variant[ScalarData, Type[Resource]]], Optional[Result]],
    }
  }`, func(ctx eval.Context, args []eval.PValue) eval.PValue {
		return NewResultSet(args...)
	}, func(ctx eval.Context, args []eval.PValue) eval.PValue {
		return NewResultSetFromHash(args[0].(*types.HashValue))
	})

	for _, br := range builtinResourceTypes {
		func(name string) {
			eval.NewObjectType(name, `Resource { attributes => { values => Hash[String,Any] }}`, nil)
			eval.NewGoConstructor(name,
				func(d eval.Dispatch) {
					d.Param(`Hash[String,Any]`)
					d.Function(func(c eval.Context, args []eval.PValue) eval.PValue {
						typ, _ := eval.Load(c, eval.NewTypedName(eval.TYPE, name))
						hash := args[0].(*types.HashValue)
						return types.NewObjectValue2(c, typ.(eval.ObjectType), types.WrapHash3(
							map[string]eval.PValue{
								`title`: hash.Get5(`title`, eval.EMPTY_STRING),
								`values`: hash.RejectPairs(func(k, v eval.PValue) bool {
									return k.String() == `title`
								})}))
					})
				})
		}(br)
	}

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
