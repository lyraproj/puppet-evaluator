package pcore

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"testing"
)

func TestPcore(t *testing.T) {
	p := NewPcore(eval.NewStdLogger())
	l, _ := eval.Load(p.SystemLoader(), eval.NewTypedName(eval.TYPE, `ObjectTypeExtensionType`))
	x, ok := l.(eval.PType)
	if !(ok && x.Name() == `ObjectTypeExtensionType`) {
		t.Errorf(`failed to load %s`, `ObjectTypeExtensionType`)
	}
}
