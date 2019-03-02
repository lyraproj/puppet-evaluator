package pcore

import (
	"github.com/lyraproj/puppet-evaluator/eval"
	"testing"
)

func TestPcore(t *testing.T) {
	eval.Puppet.Do(func(ctx eval.Context) {
		l, _ := eval.Load(ctx, eval.NewTypedName(eval.NsType, `Pcore::ObjectTypeExtensionType`))
		x, ok := l.(eval.Type)
		if !(ok && x.Name() == `Pcore::ObjectTypeExtensionType`) {
			t.Errorf(`failed to load %s`, `Pcore::ObjectTypeExtensionType`)
		}
	})
}
