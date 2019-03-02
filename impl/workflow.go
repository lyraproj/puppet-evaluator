package impl

import (
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-parser/parser"
)

var NewPuppetActivity func(c eval.EvaluationContext, expr *parser.ActivityExpression) eval.Resolvable

func init() {
	NewPuppetActivity = func(c eval.EvaluationContext, expr *parser.ActivityExpression) eval.Resolvable {
		panic("no workflow support in this runtime")
	}
}
