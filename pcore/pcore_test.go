package pcore

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"testing"
)

func TestPcore(t *testing.T) {
	InitializePuppet()
	l, _ := eval.Load(eval.Puppet.SystemLoader(), eval.NewTypedName(eval.TYPE, `Pcore::ObjectTypeExtensionType`))
	x, ok := l.(eval.PType)
	if !(ok && x.Name() == `Pcore::ObjectTypeExtensionType`) {
		t.Errorf(`failed to load %s`, `Pcore::ObjectTypeExtensionType`)
	}
}
