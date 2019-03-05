package evaluator

import (
	"github.com/lyraproj/pcore/eval"
	"github.com/lyraproj/puppet-evaluator/pdsl"
	"github.com/lyraproj/puppet-parser/parser"
)

var NewPuppetActivity func(c pdsl.EvaluationContext, expr *parser.ActivityExpression) eval.Resolvable

func init() {
	NewPuppetActivity = func(c pdsl.EvaluationContext, expr *parser.ActivityExpression) eval.Resolvable {
		panic("no workflow support in this runtime")
	}
}
