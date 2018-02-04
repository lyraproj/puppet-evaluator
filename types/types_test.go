package types

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/puppetlabs/go-evaluator/eval"
)

func TestUnique(t *testing.T) {
	x := WrapString(`hello`)
	y := WrapInteger(32)
	UniqueValues([]eval.PValue{x, y})

	z := WrapString(`hello`)
	svec := []*StringValue{x, z}
	UniqueValues([]eval.PValue{svec[0], svec[1]})
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
	tuple := NewTupleType([]eval.PType{DefaultStringType(), DefaultIntegerType()}, nil)
	t.Log(tuple.String())
}

func TestWrapAliasedString(t *testing.T) {
	v := wrap(eval.FUNCTION)
	s, ok := v.(*StringValue)
	if !(ok && s.String() == `function`) {
		t.Errorf("Namespace got wrapped as %T %s", v, v.String())
	}
}

func TestWrapMapOfInterface(t *testing.T) {
	type M map[string]interface{}
	a := wrap(M{
		`foo`: 23,
		`fee`: `hello`,
		`fum`: M{
			`x`: `1`,
			`y`: []int{1, 2, 3},
			`z`: regexp.MustCompile(`^[a-z]+$`)}})

	e := WrapHash([]*HashEntry{
		WrapHashEntry2(`foo`, WrapInteger(23)),
		WrapHashEntry2(`fee`, WrapString(`hello`)),
		WrapHashEntry2(`fum`, WrapHash([]*HashEntry{
			WrapHashEntry2(`x`, WrapString(`1`)),
			WrapHashEntry2(`y`, WrapArray([]eval.PValue{
				WrapInteger(1), WrapInteger(2), WrapInteger(3)})),
			WrapHashEntry2(`z`, WrapRegexp(`^[a-z]+$`))}))})

	if !eval.Equals(e, a) {
		t.Errorf(`Expected '%s', got '%s'`, e, a)
	}
}
