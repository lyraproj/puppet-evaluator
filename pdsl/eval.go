package pdsl

import (
	"github.com/lyraproj/issue/issue"
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

type ErrorObject interface {
	px.PuppetObject

	// Kind returns the error kind
	Kind() string

	// Message returns the error message
	Message() string

	// IssueCode returns the issue code
	IssueCode() string

	// PartialResult returns the optional partial result. It returns
	// pcore.UNDEF if no partial result exists
	PartialResult() px.Value

	// Details returns the optional details. It returns
	// an empty map when o details exist
	Details() px.OrderedMap
}

var ErrorFromReported func(c px.Context, err issue.Reported) ErrorObject

var NewError func(c px.Context, message, kind, issueCode string, partialResult px.Value, details px.OrderedMap) ErrorObject
