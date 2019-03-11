package puppet

import (
	"github.com/lyraproj/pcore/pcore"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/puppet-evaluator/evaluator"
	"github.com/lyraproj/puppet-evaluator/pdsl"

	// Ensure that all functions are loaded
	_ "github.com/lyraproj/puppet-evaluator/functions"
)

func Do(f func(ctx pdsl.EvaluationContext)) {
	pcore.Do(func(c px.Context) {
		f(evaluator.WithParent(c, evaluator.NewEvaluator))
	})
}
