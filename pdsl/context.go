package pdsl

import (
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/puppet-parser/parser"
)

const PuppetContextKey = `puppet.context`

type EvaluationContext interface {
	px.Context

	AddDefinitions(expression parser.Expression)

	// DoStatic ensures that the receiver is in static mode during the evaluation of the given doer
	DoStatic(doer px.Doer)

	// DoWithScope assigns the given scope to the receiver and calls the doer. The original scope is
	// restored before this call returns.
	DoWithScope(scope Scope, doer px.Doer)

	// EvaluatorConstructor returns the evaluator constructor
	GetEvaluator() Evaluator

	// ParseAndValidate parses and evaluates the given content. It will panic with
	// an issue.Reported unless the parsing and evaluation was successful.
	ParseAndValidate(filename, content string, singleExpression bool) parser.Expression

	// ResolveDefinitions resolves all definitions of a parser.Program
	ResolveDefinitions() []interface{}

	// ResolveType evaluates the given Expression into a Type. It will panic with
	// an issue.Reported unless the evaluation was successful and the result
	// is evaluates to a Type
	ResolveType(expr parser.Expression) px.Type

	// Static returns true during evaluation of type expressions. It is used to prevent
	// dynamic expressions within such expressions
	Static() bool
}

// TopEvaluate resolves all pending definitions prior to evaluating. The evaluated expression is not
// allowed ot contain return, next, or break.
var TopEvaluate func(c EvaluationContext, expr parser.Expression) px.Value

// Evaluate the given expression. Allow return, break, etc.
func Evaluate(c EvaluationContext, expr parser.Expression) px.Value {
	return c.GetEvaluator().Eval(expr)
}
