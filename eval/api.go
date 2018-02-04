package eval

import (
	"github.com/puppetlabs/go-parser/issue"
	"github.com/puppetlabs/go-parser/parser"
)

type (
	Evaluator interface {
		AddDefinitions(expression parser.Expression)

		ResolveDefinitions(c EvalContext)

		Evaluate(expression parser.Expression, scope Scope, loader Loader) (PValue, *issue.Reported)

		Eval(expression parser.Expression, c EvalContext) PValue

		Logger() Logger
	}

	EvalContext interface {
		Call(name string, args []PValue, block Lambda) PValue

		Evaluate(expr parser.Expression) PValue

		EvaluateIn(expr parser.Expression, scope Scope) PValue

		Evaluator() Evaluator

		// Error creates a Reported with the given issue code, location, and arguments
		// Typical use is to panic with the returned value
		Error(location issue.Location, issueCode issue.Code, args issue.H) *issue.Reported

		// Fail creates a Reported with the EVAL_FAILURE issue code, location from stack top,
		// and the given message
		// Typical use is to panic with the returned value
		Fail(message string) *issue.Reported

		Loader() Loader

		Logger() Logger

		WithScope(scope Scope) EvalContext

		ParseAndValidate(filename, content string, singleExpression bool) parser.Expression

		ParseResolve(typeString string) PType

		ParseType(str PValue) PType

		Resolve(expr parser.Expression) PValue

		ResolveType(expr parser.Expression) PType

		StackPop()

		StackPush(location issue.Location)

		StackTop() issue.Location

		Scope() Scope

		Stack() []issue.Location
	}
)

var CurrentContext func() EvalContext

// Error creates a Reported with the given issue code, location from stack top, and arguments
// Typical use is to panic with the returned value
var Error func(issueCode issue.Code, args issue.H) *issue.Reported

// Error2 creates a Reported with the given issue code, location from stack top, and arguments
// Typical use is to panic with the returned value
var Error2 func(location issue.Location, issueCode issue.Code, args issue.H) *issue.Reported

// Warning creates a Reported with the given issue code, location from stack top, and arguments
// and logs it on the currently active logger
var Warning func(issueCode issue.Code, args issue.H) *issue.Reported
