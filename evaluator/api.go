package evaluator

import (
	. "github.com/puppetlabs/go-parser/issue"
	. "github.com/puppetlabs/go-parser/parser"
)

type (
	Evaluator interface {
		AddDefinitions(expression Expression)

		ResolveDefinitions(c EvalContext)

		Evaluate(expression Expression, scope Scope, loader Loader) (PValue, *ReportedIssue)

		Eval(expression Expression, c EvalContext) PValue

		Logger() Logger
	}

	EvalContext interface {
		Call(name string, args []PValue, block Lambda) PValue

		Evaluate(expr Expression) PValue

		EvaluateIn(expr Expression, scope Scope) PValue

		Evaluator() Evaluator

		// Creates a ReportedIssue with the given issue code, location, and arguments
		// Typical use is to panic with the returned value
		Error(location Location, issueCode IssueCode, args H) *ReportedIssue

		// Creates a ReportedIssue with the EVAL_FAILURE issue code, location from stack top,
		// and the given message
		// Typical use is to panic with the returned value
		Fail(message string) *ReportedIssue

		Loader() Loader

		Logger() Logger

		WithScope(scope Scope) EvalContext

		ParseAndValidate(filename, content string, singleExpression bool) Expression

		ParseResolve(typeString string) PType

		ParseType(str PValue) PType

		Resolve(expr Expression) PValue

		ResolveType(expr Expression) PType

		StackPop()

		StackPush(location Location)

		StackTop() Location

		Scope() Scope

		Stack() []Location
	}
)

var CurrentContext func() EvalContext

// Creates a ReportedIssue with the given issue code, location from stack top, and arguments
// Typical use is to panic with the returned value
var Error func(issueCode IssueCode, args H) *ReportedIssue

// Creates a ReportedIssue with the given issue code, location from stack top, and arguments
// Typical use is to panic with the returned value
var Error2 func(location Location, issueCode IssueCode, args H) *ReportedIssue
