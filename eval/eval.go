package eval

import (
	"github.com/puppetlabs/go-issues/issue"
	"github.com/puppetlabs/go-parser/parser"
)

// An Evaluator is responsible for evaluating an Abstract Syntax Tree, typically produced by
// the parser. An implementation must be re-entrant.
type Evaluator interface {
	// CallFunction evaluates the given function call
	CallFunction(name string, args []PValue, call parser.CallExpression, c Context) PValue

	Evaluate(c Context, expression parser.Expression) (PValue, issue.Reported)

	// Eval should be considered internal. The only reason it is public is to allow
	// the evaluator to be extended. This is subject to change. Don't use
	Eval(expression parser.Expression, c Context) PValue

	Logger() Logger
}

type ParserExtension interface {
	Evaluate(e Evaluator, c Context) PValue
}

// Go calls the given function in a new go routine. The CurrentContext is forked and becomes
// the CurrentContext for that routine.
func Go(f func()) {
	CurrentContext().Go(f)
}

// Error creates a Reported with the given issue code, location from stack top, and arguments
// Typical use is to panic with the returned value
var Error func(issueCode issue.Code, args issue.H) issue.Reported

// Error2 creates a Reported with the given issue code, location from stack top, and arguments
// Typical use is to panic with the returned value
var Error2 func(location issue.Location, issueCode issue.Code, args issue.H) issue.Reported

// Warning creates a Reported with the given issue code, location from stack top, and arguments
// and logs it on the currently active logger
var Warning func(issueCode issue.Code, args issue.H) issue.Reported
