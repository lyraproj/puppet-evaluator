package impl

import (
	"github.com/puppetlabs/go-parser/parser"
)

var NewPuppetActivity func(expr *parser.ActivityExpression) interface{}

func init() {
	NewPuppetActivity = func(expr *parser.ActivityExpression) interface{} {
		panic("no workflow support in this runtime")
	}
}
