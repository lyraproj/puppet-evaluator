package pcore

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"testing"
)

func TestPcore(t *testing.T) {
	p := NewPcore(eval.NewStdLogger())
	l, _ := eval.Load(p.SystemLoader(), eval.NewTypedName(eval.TYPE, `Pcore::ObjectTypeExtensionType`))
	x, ok := l.(eval.PType)
	if !(ok && x.Name() == `Pcore::ObjectTypeExtensionType`) {
		t.Errorf(`failed to load %s`, `Pcore::ObjectTypeExtensionType`)
	}
}
