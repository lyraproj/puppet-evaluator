package impl

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-parser/parser"
)

var NewPuppetActivity func(c eval.Context, expr *parser.ActivityExpression) eval.Resolvable

func init() {
	NewPuppetActivity = func(c eval.Context, expr *parser.ActivityExpression) eval.Resolvable {
		panic("no workflow support in this runtime")
	}
}
