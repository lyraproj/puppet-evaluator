package resource

import (
	"github.com/puppetlabs/go-evaluator/eval"
	_ "github.com/puppetlabs/go-evaluator/loader"
	"github.com/puppetlabs/go-evaluator/types"
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

var resultType, resultSetType eval.Type

func InitBuiltinResources(c eval.Context) {
	initResourceType(c)

	resultType = eval.NewObjectType(`Result`, `{
    attributes => {
      'id' => ScalarData,
      'value' => { type => RichData, value => undef },
      'message' => { type => Optional[String], value => undef },
      'error' => { type => Boolean, kind => derived },
      'ok' => { type => Boolean, kind => derived }
    }
  }`, func(ctx eval.Context, args []eval.Value) eval.Value {
		return NewResult2(args...)
	}, func(ctx eval.Context, args []eval.Value) eval.Value {
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
  }`, func(ctx eval.Context, args []eval.Value) eval.Value {
		return NewResultSet(args...)
	}, func(ctx eval.Context, args []eval.Value) eval.Value {
		return NewResultSetFromHash(args[0].(*types.HashValue))
	})

	for _, br := range builtinResourceTypes {
		func(name string) {
			eval.NewObjectType(name, `Resource { attributes => { values => Hash[String,Any] }}`, nil)
			eval.NewGoConstructor(name,
				func(d eval.Dispatch) {
					d.Param(`Hash[String,Any]`)
					d.Function(func(c eval.Context, args []eval.Value) eval.Value {
						typ, _ := eval.Load(c, eval.NewTypedName(eval.TYPE, name))
						hash := args[0].(*types.HashValue)
						return types.NewObjectValue2(c, typ.(eval.ObjectType), types.WrapHashSorted(
							map[string]eval.Value{
								`title`: hash.Get5(`title`, eval.EMPTY_STRING),
								`values`: hash.RejectPairs(func(k, v eval.Value) bool {
									return k.String() == `title`
								})}))
					})
				})
		}(br)
	}

	initResourceFunctions()
}
