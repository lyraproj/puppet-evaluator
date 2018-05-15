package types_test

import (
	"fmt"
	"regexp"

	"github.com/puppetlabs/go-evaluator/eval"
	_ "github.com/puppetlabs/go-evaluator/pcore"
	"github.com/puppetlabs/go-evaluator/types"
)

func ExampleUniqueValues() {
	x := types.WrapString(`hello`)
	y := types.WrapInteger(32)
	types.UniqueValues([]eval.PValue{x, y})

	z := types.WrapString(`hello`)
	svec := []*types.StringValue{x, z}
	fmt.Println(types.UniqueValues([]eval.PValue{svec[0], svec[1]}))
	// Output: [hello]
}

func ExampleNewCallableType2() {
	cc := types.NewCallableType2(types.ZERO, types.WrapDefault())
	fmt.Println(cc)
	// Output: Callable[0, default]
}

func ExampleNewTupleType() {
	tuple := types.NewTupleType([]eval.PType{types.DefaultStringType(), types.DefaultIntegerType()}, nil)
	fmt.Println(tuple)
	// Output: Tuple[String, Integer]
}

func ExampleWrapHash() {
	type M map[string]interface{}
	a := eval.Wrap(M{
		`foo`: 23,
		`fee`: `hello`,
		`fum`: M{
			`x`: `1`,
			`y`: []int{1, 2, 3},
			`z`: regexp.MustCompile(`^[a-z]+$`)}})

	e := types.WrapHash([]*types.HashEntry{
		types.WrapHashEntry2(`foo`, types.WrapInteger(23)),
		types.WrapHashEntry2(`fee`, types.WrapString(`hello`)),
		types.WrapHashEntry2(`fum`, types.WrapHash([]*types.HashEntry{
			types.WrapHashEntry2(`x`, types.WrapString(`1`)),
			types.WrapHashEntry2(`y`, types.WrapArray([]eval.PValue{
				types.WrapInteger(1), types.WrapInteger(2), types.WrapInteger(3)})),
			types.WrapHashEntry2(`z`, types.WrapRegexp(`^[a-z]+$`))}))})

	fmt.Println(eval.Equals(e, a))
	// Output: true
}
