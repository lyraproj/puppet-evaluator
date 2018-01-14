package loader

import (
	. "github.com/puppetlabs/go-evaluator/evaluator"
	. "github.com/puppetlabs/go-parser/parser"
	"strings"
)

type Instantiator func(loader ContentProvidingLoader, tn TypedName, sources []string)

func InstantiatePuppetFunction(loader ContentProvidingLoader, tn TypedName, sources []string) {
	source := sources[0]
	content := string(loader.GetContent(source))
	ctx := CurrentContext()
	expr := ctx.ParseAndValidate(content, source, false)
	name := tn.Name()
	fd, ok := getDefinition(ctx, expr, FUNCTION, name).(*FunctionDefinition)
	if !ok {
		panic(ctx.Error(expr, EVAL_NO_DEFINITION, expr.File(), FUNCTION, name))
	}
	if strings.ToLower(fd.Name()) != strings.ToLower(name) {
		panic(ctx.Error(expr, EVAL_WRONG_DEFINITION, expr.File(), FUNCTION, name, fd.Name()))
	}
	e := ctx.Evaluator()
	e.AddDefinitions(expr)
	e.ResolveDefinitions()
}

func InstantiatePuppetType(loader ContentProvidingLoader, tn TypedName, sources []string) {
	content := string(loader.GetContent(sources[0]))
	ctx := CurrentContext()
	expr := ctx.ParseAndValidate(content, sources[0], false)
	name := tn.Name()
	def := getDefinition(ctx, expr, TYPE, name)
	var tdn string
	switch def.(type) {
	case *TypeAlias:
		tdn = def.(*TypeAlias).Name()
	case *TypeDefinition:
		tdn = def.(*TypeDefinition).Name()
	default:
		panic(ctx.Error(expr, EVAL_NO_DEFINITION, expr.File(), TYPE, name))
	}
	if strings.ToLower(tdn) != strings.ToLower(name) {
		panic(ctx.Error(expr, EVAL_WRONG_DEFINITION, expr.File(), TYPE, name, tdn))
	}
	e := ctx.Evaluator()
	e.AddDefinitions(expr)
	e.ResolveDefinitions()
}

func InstantiatePuppetTask(loader ContentProvidingLoader, name TypedName, sources []string) {
}

// Extract a single Definition and return it. Will fail and report an error unless the program contains
// only one Definition
func getDefinition(ctx EvalContext, expr Expression, ns Namespace, name string) Definition {
	if p, ok := expr.(*Program); ok {
		if b, ok := p.Body().(*BlockExpression); ok {
			switch len(b.Statements()) {
			case 0:
			case 1:
				if d, ok := b.Statements()[0].(Definition); ok {
					return d
				}
			default:
				panic(ctx.Error(expr, EVAL_NOT_ONLY_DEFINITION, expr.File(), ns, name))
			}
		}
	}
	panic(ctx.Error(expr, EVAL_NO_DEFINITION, expr.File(), ns, name))
}