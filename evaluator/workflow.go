package evaluator

import (
	"github.com/lyraproj/puppet-evaluator/pdsl"
	"github.com/lyraproj/puppet-parser/parser"
)

var NewPuppetStep func(c pdsl.EvaluationContext, expr *parser.StepExpression) Resolvable

func init() {
	NewPuppetStep = func(c pdsl.EvaluationContext, expr *parser.StepExpression) Resolvable {
		panic("no workflow support in this runtime")
	}
}
