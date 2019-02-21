package types_test

import (
	"fmt"
	"github.com/lyraproj/puppet-evaluator/eval"
	// Initialize pcore
	_ "github.com/lyraproj/puppet-evaluator/pcore"
)

func ExampleObject_Initialize() {
	eval.Puppet.Do(func(c eval.Context) {
		t := c.ParseType2(`Object[
      name => 'Address',
      attributes => {
        'annotations' => {
          'type' => Optional[Hash[String, String]],
          'value' => undef
        },
        'lineOne' => {
          'type' => String,
          'value' => ''
        }
      }
    ]`)
		c.AddTypes(t)
		c.ResolveDefinitions()

		v := eval.New(c, t, eval.Wrap(c, map[string]string{`lineOne`: `30 East 60th Street`}))
		fmt.Println(v.String())
	})

	// Output: Address('lineOne' => '30 East 60th Street')
}
