package eval_test

import (
	"fmt"
	"github.com/puppetlabs/go-evaluator/eval"
	// Initialize pcore
	_ "github.com/puppetlabs/go-evaluator/pcore"
)

func ExampleWrap() {
	// Wrap native Go types
	str := eval.Wrap("hello")
	idx := eval.Wrap(23)
	flt := eval.Wrap(123.34)
	bl := eval.Wrap(true)
	und := eval.UNDEF

	fmt.Printf("'%s' is a %s\n", str, str.Type())
	fmt.Printf("'%s' is a %s\n", idx, idx.Type())
	fmt.Printf("'%s' is a %s\n", flt, flt.Type())
	fmt.Printf("'%s' is a %s\n", bl, bl.Type())
	fmt.Printf("'%s' is a %s\n", und, und.Type())

	// Output:
	// 'hello' is a String
	// '23' is a Integer[23, 23]
	// '123.34' is a Float[123.340, 123.340]
	// 'true' is a Boolean
	// 'undef' is a Undef
}

func ExampleWrap_slice() {
	// Wrap native Go slice
	arr := eval.Wrap([]interface{}{1, 2.2, true, nil, "hello"})
	fmt.Printf("%s is an %s\n", arr, arr.Type())

	// Output:
	// [1, 2.20000, true, undef, 'hello'] is an Array[5, 5]
}

func ExampleWrap_hash() {
	// Wrap native Go hash
	hsh := eval.Wrap(map[string]interface{}{
		"first":  1,
		"second": 2.2,
		"third":  "three",
		"nested": []string{"hello", "world"},
	})
	nst, _ := hsh.(eval.KeyedValue).Get4("nested")
	fmt.Printf("'%s' is a %s\n", hsh, hsh.Type())
	fmt.Printf("hsh['nested'] == %s, an instance of %s\n", nst, nst.Type())

	// Output:
	// '{'first' => 1, 'nested' => ['hello', 'world'], 'second' => 2.20000, 'third' => 'three'}' is a Hash[Enum['first', 'nested', 'second', 'third'], Data, 4, 4]
	// hsh['nested'] == ['hello', 'world'], an instance of Array[Enum['hello', 'world'], 2, 2]
}

func ExamplePcore_ParseType() {
	eval.Puppet.Do(func(ctx eval.Context) error {
		pcoreType := ctx.ParseType2("Enum[foo,fee,fum]")
		fmt.Printf("%s is an instance of %s\n", pcoreType, pcoreType.Type())
		return nil
	})
	// Output:
	// Enum['foo', 'fee', 'fum'] is an instance of Type[Enum['foo', 'fee', 'fum']]
}

func ExamplePcore_IsInstance() {
	eval.Puppet.Do(func(ctx eval.Context) error {
		pcoreType := ctx.ParseType2("Enum[foo,fee,fum]")
		fmt.Println(eval.IsInstance(ctx, pcoreType, eval.Wrap("foo")))
		fmt.Println(eval.IsInstance(ctx, pcoreType, eval.Wrap("bar")))
		return nil
	})
	// Output:
	// true
	// false
}

func ExamplePcore_ParseType_error() {
	err := eval.Puppet.Do(func(ctx eval.Context) error {
		ctx.ParseType2("Enum[foo") // Missing end bracket
		return nil
	})
	if err != nil {
		fmt.Println(err)
	}
	// Output: expected one of ',' or ']', got 'EOF' (line: 1, column: 9)
}
