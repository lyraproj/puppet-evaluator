package types_test

import (
	"fmt"
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/types"
	"os"

	// Initialize pcore
	_ "github.com/lyraproj/puppet-evaluator/pcore"
)

func ExampleParse_qName() {
	t, err := types.Parse(`Foo::Bar`)
	if err == nil {
		t.ToString(os.Stdout, eval.PrettyExpanded, nil)
		fmt.Println()
	} else {
		fmt.Println(err)
	}
	// Output: DeferredType(Foo::Bar)
}

func ExampleParse_entry() {
	const src = `# This is scanned code.
    constants => {
      first => 0,
      second => 0x32,
      third => 2e4,
      fourth => 2.3e-2,
      fifth => 'hello',
      sixth => "world",
      type => Foo::Bar[1,'23',Baz[1,2]],
      value => "String\nWith \\Escape",
      array => [a, b, c],
      call => Boo::Bar("with", "args")
    }
  `
	v, err := types.Parse(src)
	if err == nil {
		v.ToString(os.Stdout, eval.PrettyExpanded, nil)
		fmt.Println()
	} else {
		fmt.Println(err)
	}
	// Output:
	// {
	//   'constants' => {
	//     'first' => 0,
	//     'second' => 50,
	//     'third' => 20000.0,
	//     'fourth' => 0.02300,
	//     'fifth' => 'hello',
	//     'sixth' => 'world',
	//     'type' => DeferredType(Foo::Bar, [1, '23', DeferredType(Baz, [1, 2])]),
	//     'value' => "String\nWith \\Escape",
	//     'array' => ['a', 'b', 'c'],
	//     'call' => Deferred(
	//       'name' => 'new',
	//       'arguments' => ['Boo::Bar', 'with', 'args']
	//     )
	//   }
	// }
	//
}
