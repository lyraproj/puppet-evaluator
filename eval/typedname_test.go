package eval

import (
	"testing"
	"fmt"
)

func ExampleTypedName_IsParent() {
	tn := NewTypedName(TYPE, `Foo::Fee::Fum`)
	tp := NewTypedName(TYPE, `Foo::Fee`)
	fmt.Println(tp.IsParent(tn))

	// Output: true
}

func ExampleTypedName_IsParent_1() {
	tn := NewTypedName(TYPE, `Foo::Fee::Fum`)
	tp := NewTypedName(TYPE, `Foo::Bar`)
	fmt.Println(tp.IsParent(tn))

	// Output: false
}

func ExampleTypedName_IsParent_2() {
	tn := NewTypedName(TYPE, `Foo::Fee::Fum`)
	tp := NewTypedName(TYPE, `Foo::Fee`)
	fmt.Println(tn.IsParent(tp))

	// Output: false
}

func ExampleTypedName_IsParent_3() {
	tn := NewTypedName(TYPE, `Foo::Fee::Fum`)
	fmt.Println(tn.IsParent(tn))

	// Output: false
}

func TestParent(t *testing.T) {
	tn := NewTypedName(TYPE, `Foo::Fee::Fum`)
	tp := NewTypedName(TYPE, `Foo::Fee`)
	if !Equals(tn.Parent(), tp) {
		t.Error(`TypedName.Parent() does not return parent`)
	}
}

func TestChild(t *testing.T) {
	tn := NewTypedName(TYPE, `Foo::Fee::Fum`)
	tp := NewTypedName(TYPE, `Fee::Fum`)
	ch := tn.Child()
	if !Equals(ch, tp) {
		t.Errorf(`TypedName.Child() does not return child. Expected '%s', got '%s'`, tp, ch)
	}
}
