package impl

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-parser/parser"
)

var NewPuppetActivity func(expr *parser.ActivityExpression) eval.Function

func init() {
	NewPuppetActivity = func(expr *parser.ActivityExpression) eval.Function {
		panic("no workflow support in this runtime")
	}
}
