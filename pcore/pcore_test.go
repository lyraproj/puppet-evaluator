package pcore

import (
	"github.com/puppetlabs/go-evaluator/evaluator"
	"testing"
)

func TestPcore(t *testing.T) {
	p := NewPcore(evaluator.NewStdLogger())
	l, _ := evaluator.Load(p.SystemLoader(), evaluator.NewTypedName(evaluator.TYPE, `ObjectTypeExtensionType`))
	x, ok := l.(evaluator.PType)
	if !(ok && x.Name() == `ObjectTypeExtensionType`) {
		t.Errorf(`failed to load %s`, `ObjectTypeExtensionType`)
	}
}
