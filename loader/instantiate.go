package loader

import (
	. "github.com/puppetlabs/go-evaluator/evaluator"
)

type Instantiator func(loader ContentProvidingLoader, name TypedName, sources []string)

func InstantiatePuppetFunction(loader ContentProvidingLoader, name TypedName, sources []string) {
	content := string(loader.GetContent(sources[0]))
	ctx := CurrentContext()
	expr := ctx.ParseAndValidate(content, sources[0], false)
	e := ctx.Evaluator()
	e.AddDefinitions(expr)
	e.ResolveDefinitions()
}

func InstantiatePuppetType(loader ContentProvidingLoader, name TypedName, sources []string) {
}

func InstantiatePuppetTask(loader ContentProvidingLoader, name TypedName, sources []string) {
}
