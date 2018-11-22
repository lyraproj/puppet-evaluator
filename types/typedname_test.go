package types

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"testing"
	"fmt"
)

func ExampleTypedName_IsParent() {
	tn := NewTypedName(eval.NsType, `Foo::Fee::Fum`)
	tp := NewTypedName(eval.NsType, `Foo::Fee`)
	fmt.Println(tp.IsParent(tn))

	// Output: true
}

func ExampleTypedName_IsParent_1() {
	tn := NewTypedName(eval.NsType, `Foo::Fee::Fum`)
	tp := NewTypedName(eval.NsType, `Foo::Bar`)
	fmt.Println(tp.IsParent(tn))

	// Output: false
}

func ExampleTypedName_IsParent_2() {
	tn := NewTypedName(eval.NsType, `Foo::Fee::Fum`)
	tp := NewTypedName(eval.NsType, `Foo::Fee`)
	fmt.Println(tn.IsParent(tp))

	// Output: false
}

func ExampleTypedName_IsParent_3() {
	tn := NewTypedName(eval.NsType, `Foo::Fee::Fum`)
	fmt.Println(tn.IsParent(tn))

	// Output: false
}

func ExampleTypedName_MapKey() {
	tn := NewTypedName(eval.NsType, `Foo::Fee::Fum`)
	fmt.Println(tn.MapKey())

	// Output: http://puppet.com/2016.1/runtime/type/foo::fee::fum
}

func ExampleTypedNameFromMapKey() {
	tn := NewTypedName(eval.NsType, `Foo::Fee::Fum`)
	tn = typedNameFromMapKey(tn.MapKey())
	fmt.Println(tn)

	// Output: TypedName('namespace' => 'type', 'name' => 'foo::fee::fum')
}

func TestParent(t *testing.T) {
	tn := NewTypedName(eval.NsType, `Foo::Fee::Fum`)
	tp := NewTypedName(eval.NsType, `Foo::Fee`)
	if !eval.Equals(tn.Parent(), tp) {
		t.Error(`TypedName.Parent() does not return parent`)
	}
}

func TestChild(t *testing.T) {
	tn := NewTypedName(eval.NsType, `Foo::Fee::Fum`)
	tp := NewTypedName(eval.NsType, `Fee::Fum`)
	ch := tn.Child()
	if !eval.Equals(ch, tp) {
		t.Errorf(`TypedName.Child() does not return child. Expected '%s', got '%s'`, tp, ch)
	}
}
