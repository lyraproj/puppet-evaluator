package eval

import (
	"context"
	"github.com/puppetlabs/go-issues/issue"
	"github.com/puppetlabs/go-parser/parser"
)

// An Context holds all state during evaluation. Since it contains the stack, each
// thread of execution must use a context of its own. It's expected that multiple
// contexts share common parents for scope and loaders.
//
type Context interface {
	context.Context

	AddDefinitions(expression parser.Expression)

	// AddTypes Makes the given types known to the loader appointed by this context
	AddTypes(types ...PType)

	// Call calls a function known to the loader with arguments and an optional
	// block.
	Call(name string, args []PValue, block Lambda) PValue

	// Delete deletes the given key from the context variable map
	Delete(key string)

	// DefiningLoader returns a Loader that can receive new definitions
	DefiningLoader() DefiningLoader

	// Evaluate evaluates the given expression using the evaluator of the receiver.
	Evaluate(expr parser.Expression) PValue

	// EvaluateIn evaluates the given expression using the given scope.
	EvaluateIn(expr parser.Expression, scope Scope) PValue

	// Evaluator returns the evaluator of the receiver.
	Evaluator() Evaluator

	// Error creates a Reported with the given issue code, location, and arguments
	// Typical use is to panic with the returned value
	Error(location issue.Location, issueCode issue.Code, args issue.H) issue.Reported

	// Fail creates a Reported with the EVAL_FAILURE issue code, location from stack top,
	// and the given message
	// Typical use is to panic with the returned value
	Fail(message string) issue.Reported

	// Fork a new context from this context. The fork will have the same scope,
	// loaders, and logger as this context. The stack and the map of context variables will
	// be shallow copied
	Fork() Context

	// Get returns the context variable with the given key together with a bool to indicate
	// if the key was found
	Get(key string) (interface{}, bool)

	// ImplementationRegistry returns the registry that holds mappings between PType and reflect.Type
	ImplementationRegistry() ImplementationRegistry

	// Loader returns the loader of the receiver.
	Loader() Loader

	// Logger returns the logger of the receiver. This will be the same logger as the
	// logger of the evaluator.
	Logger() Logger

	// WithLoader creates a copy of the receiver where the loader is replaced with the
	// given loader.
	WithLoader(loader Loader) Context

	// WithScope creates a copy of the receiver where the scope is replaced with the
	// given scope.
	WithScope(scope Scope) Context

	// ParseAndValidate parses and evaluates the given content. It will panic with
	// an issue.Reported unless the parsing and evaluation was succesful.
	ParseAndValidate(filename, content string, singleExpression bool) parser.Expression

	// ParseType parses and evaluates the given PValue into a PType. It will panic with
	// an issue.Reported unless the parsing was succesfull and the result is evaluates
	// to a PType
	ParseType(str PValue) PType

	// ParseType2 parses and evaluates the given string into a PType. It will panic with
	// an issue.Reported unless the parsing was succesfull and the result is evaluates
	// to a PType
	ParseType2(typeString string) PType

	// Reflector returns a Reflector capable of converting to and from refleced values
	// and types
	Reflector() Reflector

	ResolveDefinitions()

	// Resolve types, constructions, or functions that has been recently added
	ResolveResolvables()

	// ResolveType evaluates the given Expression into a PType. It will panic with
	// an issue.Reported unless the evaluation was succesfull and the result
	// is evaluates to a PType
	ResolveType(expr parser.Expression) PType

	// Set adds or replaces the context variable for the given key with the given value
	Set(key string, value interface{})

	// StackPop pops the last pushed location from the stack
	StackPop()

	// StackPush pushes a location onto the stack. The location is typically the
	// currently evaluated expression.
	StackPush(location issue.Location)

	// StackTop returns the top of the stack
	StackTop() issue.Location

	// Scope returns the scope
	Scope() Scope

	// Stack returns the full stack. The returned value must not be modified.
	Stack() []issue.Location
}
