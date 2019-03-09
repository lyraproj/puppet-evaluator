package evaluator

import (
	"github.com/lyraproj/puppet-evaluator/pdsl"
	"github.com/lyraproj/puppet-parser/parser"
)

var NewPuppetActivity func(c pdsl.EvaluationContext, expr *parser.ActivityExpression) Resolvable

func init() {
	NewPuppetActivity = func(c pdsl.EvaluationContext, expr *parser.ActivityExpression) Resolvable {
		panic("no workflow support in this runtime")
	}
}
