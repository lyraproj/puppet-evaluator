package values

import (
	"testing"

	"fmt"

	"github.com/puppetlabs/go-evaluator/eval/values/api"
)

func TestUnique(t *testing.T) {
	x := WrapString(`hello`)
	y := WrapInteger(32)
	UniqueValues([]api.PValue{x, y})

	z := WrapString(`hello`)
	svec := []*StringValue{x, z}
	UniqueValues([]api.PValue{svec[0], svec[1]})
}

func TestFloat(t *testing.T) {
	fmt.Printf(`%#g`, 18.0)
}

func TestInteger(t *testing.T) {
	v := NewIntegerType(0, 0)
	if x, ok := toInt(v); ok {
		fmt.Errorf("Oh no. Integer[0,0] is castable to int %d", x)
	}
	if x, ok := toInt(ZERO); ok {
		fmt.Errorf("Oh no. 0 is not castable to int %d", x)
	}
}

func TestCallable(t *testing.T) {
	cc := NewCallableType2(ZERO, WrapDefault())
	t.Log(cc.String())

}

func TestTuple(t *testing.T) {
	tuple := tupleFromArgs(false, []api.PValue{DefaultStringType(), DefaultIntegerType()})
	t.Log(tuple.String())

}
