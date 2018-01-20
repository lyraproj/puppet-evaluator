package loader

import (
	. "github.com/puppetlabs/go-evaluator/evaluator"
	. "github.com/puppetlabs/go-parser/parser"
	"strings"
	. "github.com/puppetlabs/go-parser/issue"
)

type Instantiator func(loader ContentProvidingLoader, tn TypedName, sources []string)

func InstantiatePuppetFunction(loader ContentProvidingLoader, tn TypedName, sources []string) {
	source := sources[0]
	content := string(loader.GetContent(source))
	ctx := CurrentContext()
	expr := ctx.ParseAndValidate(content, source, false)
	name := tn.Name()
	fd, ok := getDefinition(expr, FUNCTION, name).(*FunctionDefinition)
	if !ok {
		panic(ctx.Error(expr, EVAL_NO_DEFINITION, H{`source`: expr.File(), `type`: FUNCTION, `name`: name}))
	}
	if strings.ToLower(fd.Name()) != strings.ToLower(name) {
		panic(ctx.Error(expr, EVAL_WRONG_DEFINITION, H{`source`: expr.File(), `type`: FUNCTION, `expected`: name, `actual`: fd.Name()}))
	}
	e := ctx.Evaluator()
	e.AddDefinitions(expr)
	e.ResolveDefinitions(ctx)
}

func InstantiatePuppetType(loader ContentProvidingLoader, tn TypedName, sources []string) {
	content := string(loader.GetContent(sources[0]))
	ctx := CurrentContext()
	expr := ctx.ParseAndValidate(content, sources[0], false)
	name := tn.Name()
	def := getDefinition(expr, TYPE, name)
	var tdn string
	switch def.(type) {
	case *TypeAlias:
		tdn = def.(*TypeAlias).Name()
	case *TypeDefinition:
		tdn = def.(*TypeDefinition).Name()
	case *TypeMapping:
		tdn = def.(*TypeMapping).Type().Label()
	default:
		panic(ctx.Error(expr, EVAL_NO_DEFINITION, H{`source`: expr.File(), `type`: TYPE, `name`: name}))
	}
	if strings.ToLower(tdn) != strings.ToLower(name) {
		panic(ctx.Error(expr, EVAL_WRONG_DEFINITION, H{`source`: expr.File(), `type`: TYPE, `expected`: name, `actual`: tdn}))
	}
	e := ctx.Evaluator()
	e.AddDefinitions(expr)
	e.ResolveDefinitions(ctx)
}

func InstantiatePuppetTask(loader ContentProvidingLoader, name TypedName, sources []string) {
}

// Extract a single Definition and return it. Will fail and report an error unless the program contains
// only one Definition
func getDefinition(expr Expression, ns Namespace, name string) Definition {
	if p, ok := expr.(*Program); ok {
		if b, ok := p.Body().(*BlockExpression); ok {
			switch len(b.Statements()) {
			case 0:
			case 1:
				if d, ok := b.Statements()[0].(Definition); ok {
					return d
				}
			default:
				panic(Error2(expr, EVAL_NOT_ONLY_DEFINITION, H{`source`: expr.File(), `type`: ns, `name`: name}))
			}
		}
	}
	panic(Error2(expr, EVAL_NO_DEFINITION, H{`source`: expr.File(), `type`: ns, `name`: name}))
}