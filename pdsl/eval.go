package pdsl

import (
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/puppet-parser/parser"
)

// An Evaluator is responsible for evaluating an Abstract Syntax Tree, typically produced by
// the parser. An implementation must be re-entrant.
type Evaluator interface {
	EvaluationContext

	// Eval should be considered internal. The only reason it is public is to allow
	// the evaluator to be extended. This is subject to change. Don't use
	Eval(expression parser.Expression) px.Value
}

type ParserExtension interface {
	Evaluate(e Evaluator) px.Value
}
