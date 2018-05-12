package resource

import "testing"

func TestRefSplitOK(t *testing.T) {
	if x, y, ok := SplitRef(`x[y]`); ok {
		if x != `x` {
			t.Errorf(`Expected type name 'x', got '%s'`, x)
		}
		if y != `y` {
			t.Errorf(`Expected title 'y', got '%s'`, y)
		}
	} else {
		t.Errorf(`Expected succesful split of 'x[y]'`)
	}
}

func TestRefSplitNotOK(t *testing.T) {
	if _, _, ok := SplitRef(`[y]`); ok {
		t.Errorf(`Expected unsuccesful split of '[y]'`)
	}
	if _, _, ok := SplitRef(`[xyz]`); ok {
		t.Errorf(`Expected unsuccesful split of '[xyz]'`)
	}
	if _, _, ok := SplitRef(`x[]`); ok {
		t.Errorf(`Expected unsuccesful split of 'x[]'`)
	}
	if _, _, ok := SplitRef(`xyz[]`); ok {
		t.Errorf(`Expected unsuccesful split of 'xyz[]'`)
	}
	if _, _, ok := SplitRef(`x[y] `); ok {
		t.Errorf(`Expected unsuccesful split of 'x[y] '`)
	}
}
