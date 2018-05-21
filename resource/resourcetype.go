package resource

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"reflect"
)

var resourceType eval.PType

type Resource struct {
	title string
}

func initResourceType(c eval.Context) {
	resourceType = eval.NewObjectType(`Resource`, `{
    type_parameters => {
      title => String[1]
    },
    attributes => {
      title => String[1]
    }}`)

	// Enable Resource as parent in Go structures
	c.ImplementationRegistry().RegisterType(c, `Resource`, reflect.TypeOf(&Resource{}))
}
