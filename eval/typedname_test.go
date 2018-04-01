package eval

import "testing"

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
