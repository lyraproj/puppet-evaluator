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

func InitBuiltinResources() {
	eval.NewObjectType(`Resource`, `{
    type_parameters => {
      title => String[1]
    },
    attributes => {
      title => String[1]
    }}`)

	for _, br := range builtinResourceTypes {
		func(name string) {
			eval.NewObjectType(name, `Resource { attributes => { values => Hash[String,Any] }}`, nil)
			eval.NewGoConstructor(name,
				func(d eval.Dispatch) {
					d.Param(`Hash[String,Any]`)
					d.Function(func(c eval.EvalContext, args []eval.PValue) eval.PValue {
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
	}
