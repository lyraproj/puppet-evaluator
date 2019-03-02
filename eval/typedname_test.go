package eval

import (
	"fmt"
	"testing"
)

func ExampleTypedName_IsParent() {
	tn := NewTypedName(NsType, `Foo::Fee::Fum`)
	tp := NewTypedName(NsType, `Foo::Fee`)
	fmt.Println(tp.IsParent(tn))

	// Output: true
}

func ExampleTypedName_IsParent_ofDifferentBranch() {
	tn := NewTypedName(NsType, `Foo::Fee::Fum`)
	tp := NewTypedName(NsType, `Foo::Bar`)
	fmt.Println(tp.IsParent(tn))

	// Output: false
}

func ExampleTypedName_IsParent_ofParent() {
	tn := NewTypedName(NsType, `Foo::Fee::Fum`)
	tp := NewTypedName(NsType, `Foo::Fee`)
	fmt.Println(tn.IsParent(tp))

	// Output: false
}

func ExampleTypedName_IsParent_ofItself() {
	tn := NewTypedName(NsType, `Foo::Fee::Fum`)
	fmt.Println(tn.IsParent(tn))

	// Output: false
}

func ExampleTypedName_MapKey() {
	tn := NewTypedName(NsType, `Foo::Fee::Fum`)
	fmt.Println(tn.MapKey())

	// Output: http://puppet.com/2016.1/runtime/type/foo::fee::fum
}

func ExampleTypedNameFromMapKey() {
	tn := NewTypedName(NsType, `Foo::Fee::Fum`)
	tn = TypedNameFromMapKey(tn.MapKey())
	fmt.Println(tn)

	// Output: TypedName('namespace' => 'type', 'name' => 'foo::fee::fum')
}

func TestParent(t *testing.T) {
	tn := NewTypedName(NsType, `Foo::Fee::Fum`)
	tp := NewTypedName(NsType, `Foo::Fee`)
	if !Equals(tn.Parent(), tp) {
		t.Error(`TypedName.Parent() does not return parent`)
	}
}

func TestChild(t *testing.T) {
	tn := NewTypedName(NsType, `Foo::Fee::Fum`)
	tp := NewTypedName(NsType, `Fee::Fum`)
	ch := tn.Child()
	if !Equals(ch, tp) {
		t.Errorf(`TypedName.Child() does not return child. Expected '%s', got '%s'`, tp, ch)
	}
}
