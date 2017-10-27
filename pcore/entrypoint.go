package pcore

import (
	. "github.com/puppetlabs/go-evaluator/eval"
	. "github.com/puppetlabs/go-evaluator/evaluator"
	_ "github.com/puppetlabs/go-evaluator/functions"
)

type (
	pcoreImpl struct {
		loader Loader
	}
)

func NewPcore(logger Logger) Pcore {
	loader := NewParentedLoader(StaticLoader())
	p := &pcoreImpl{loader: loader}

	ResolveGoFunctions(loader, logger)
	return p
}

func (p *pcoreImpl) Loader() Loader {
	return p.loader
}
