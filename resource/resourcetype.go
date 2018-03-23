package resource

import "github.com/puppetlabs/go-evaluator/eval"

func init() {
	eval.NewObjectType(`Resource`, `{
    type_parameters => {
      title => String[1]
    },
    attributes => {
      title => String[1]
    }
 }`)
}
