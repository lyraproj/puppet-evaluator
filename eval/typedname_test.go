package eval

import "testing"

func TestParent(t *testing.T) {
	tn := NewTypedName(TYPE, `Foo::Fee::Fum`)
	tp := NewTypedName(TYPE, `Foo::Fee`)
	if !Equals(tn.Parent(), tp) {
		t.Error(`TypedName.Parent() does not return parent`)
	}
}
