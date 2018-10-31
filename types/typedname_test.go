package types

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"testing"
	"fmt"
)

func ExampleTypedName_IsParent() {
	tn := newTypedName(eval.TYPE, `Foo::Fee::Fum`)
	tp := newTypedName(eval.TYPE, `Foo::Fee`)
	fmt.Println(tp.IsParent(tn))

	// Output: true
}

func ExampleTypedName_IsParent_1() {
	tn := newTypedName(eval.TYPE, `Foo::Fee::Fum`)
	tp := newTypedName(eval.TYPE, `Foo::Bar`)
	fmt.Println(tp.IsParent(tn))

	// Output: false
}

func ExampleTypedName_IsParent_2() {
	tn := newTypedName(eval.TYPE, `Foo::Fee::Fum`)
	tp := newTypedName(eval.TYPE, `Foo::Fee`)
	fmt.Println(tn.IsParent(tp))

	// Output: false
}

func ExampleTypedName_IsParent_3() {
	tn := newTypedName(eval.TYPE, `Foo::Fee::Fum`)
	fmt.Println(tn.IsParent(tn))

	// Output: false
}

func TestParent(t *testing.T) {
	tn := newTypedName(eval.TYPE, `Foo::Fee::Fum`)
	tp := newTypedName(eval.TYPE, `Foo::Fee`)
	if !eval.Equals(tn.Parent(), tp) {
		t.Error(`TypedName.Parent() does not return parent`)
	}
}

func TestChild(t *testing.T) {
	tn := newTypedName(eval.TYPE, `Foo::Fee::Fum`)
	tp := newTypedName(eval.TYPE, `Fee::Fum`)
	ch := tn.Child()
	if !eval.Equals(ch, tp) {
		t.Errorf(`TypedName.Child() does not return child. Expected '%s', got '%s'`, tp, ch)
	}
}
