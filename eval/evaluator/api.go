package evaluator

import (
	. "github.com/puppetlabs/go-evaluator/eval/utils"
	. "github.com/puppetlabs/go-evaluator/eval/values/api"
	"github.com/puppetlabs/go-parser/issue"
	. "github.com/puppetlabs/go-parser/parser"
)

const (
	ALERT   = LogLevel(`alert`)
	CRIT    = LogLevel(`crit`)
	DEBUG   = LogLevel(`debug`)
	EMERG   = LogLevel(`emerg`)
	ERR     = LogLevel(`err`)
	INFO    = LogLevel(`info`)
	NOTICE  = LogLevel(`notice`)
	WARNING = LogLevel(`warning`)
)

var LOG_LEVELS = []LogLevel{ALERT, CRIT, DEBUG, EMERG, ERR, INFO, NOTICE, WARNING}

type (
	ValueProducer func(scope Scope) PValue

	Scope interface {
		WithLocalScope(producer ValueProducer) PValue

		Get(name string) (value PValue, found bool)

		Set(name string, value PValue) bool

		RxSet(variables []string)

		RxGet(index int) (value PValue, found bool)
	}

	Evaluator interface {
		AddDefinitions(expression Expression)

		ResolveDefinitions()

		Evaluate(expression Expression, scope Scope, loader Loader) (PValue, *issue.ReportedIssue)

		Eval(expression Expression, c EvalContext) PValue

		Logger() Logger
	}

	TypedName interface {
		Equals(tn TypedName) bool

		IsQualified() bool

		MapKey() string

		String() string

		Namespace() Namespace
	}

	Loader interface {
		Load(name TypedName) (interface{}, bool)

		NameAuthority() URI
	}

	DefiningLoader interface {
		Loader

		ResolveGoFunctions(c EvalContext)

		SetEntry(name TypedName, value interface{})
	}

	EvalContext interface {
		Evaluate(expr Expression) PValue

		EvaluateIn(expr Expression, scope Scope) PValue

		Evaluator() Evaluator

		WithScope(scope Scope) EvalContext

		ParseType(str PValue) PType

		Loader() Loader

		Logger() Logger

		Scope() Scope
	}

	CallStackEntry interface {
	}

	Lambda interface {
		PValue

		CallStackEntry

		Call(c EvalContext, block Lambda, args ...PValue) PValue

		Signature() Signature
	}

	Function interface {
		PValue

		Call(c EvalContext, block Lambda, args ...PValue) PValue

		Dispatchers() []Lambda

		Name() string
	}
)
