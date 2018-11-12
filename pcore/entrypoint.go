package pcore

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sync"

	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/impl"
	"github.com/puppetlabs/go-evaluator/types"

	// Import ordering and subsequent initialisation currently
	// results in a segfault if functions is not imported (but
	// not used) at this point
	"context"
	_ "github.com/puppetlabs/go-evaluator/functions"
	"github.com/puppetlabs/go-issues/issue"
	"github.com/puppetlabs/go-parser/parser"
	"github.com/puppetlabs/go-parser/validator"
	"github.com/puppetlabs/go-evaluator/threadlocal"
)

type (
	pcoreImpl struct {
		lock              sync.RWMutex
		logger            eval.Logger
		systemLoader      eval.Loader
		environmentLoader eval.Loader
		moduleLoaders     map[string]eval.Loader
		settings          map[string]*setting
	}
)

var staticLock sync.Mutex
var puppet = &pcoreImpl{settings:make(map[string]*setting, 32)}
var topImplRegistry eval.ImplementationRegistry

func init() {
	eval.Puppet = puppet
	puppet.DefineSetting(`environment`, types.DefaultStringType(), types.WrapString(`production`))
	puppet.DefineSetting(`environmentpath`, types.DefaultStringType(), nil)
	puppet.DefineSetting(`module_path`, types.DefaultStringType(), nil)
	puppet.DefineSetting(`strict`, types.NewEnumType([]string{`off`, `warning`, `error`}, true), types.WrapString(`warning`))
	puppet.DefineSetting(`tasks`, types.DefaultBooleanType(), types.WrapBoolean(false))
	puppet.DefineSetting(`workflow`, types.DefaultBooleanType(), types.WrapBoolean(false))
}

func InitializePuppet() {
	// First call initializes the static loader. There can be only one since it receives
	// most of its contents from Go init() functions
	staticLock.Lock()
	defer staticLock.Unlock()

	if puppet.logger != nil {
		return
	}

	puppet.logger = eval.NewStdLogger()
	c := impl.NewContext(puppet.NewEvaluator(), eval.StaticLoader().(eval.DefiningLoader))
	c.ResolveResolvables()
	topImplRegistry = c.ImplementationRegistry()
}

func (p *pcoreImpl) Reset() {
	p.lock.Lock()
	p.systemLoader = nil
	p.environmentLoader = nil
	for _, s := range p.settings {
		s.reset()
	}
	p.lock.Unlock()
}

func (p *pcoreImpl) SystemLoader() eval.Loader {
	p.lock.Lock()
	p.ensureSystemLoader()
	p.lock.Unlock()
	return p.systemLoader
}

func (p *pcoreImpl) configuredStaticLoader() eval.Loader {
	return eval.StaticLoader()
}

// not exported, provides unprotected access to shared object
func (p *pcoreImpl) ensureSystemLoader() eval.Loader {
	if p.systemLoader == nil {
		p.systemLoader = eval.NewParentedLoader(p.configuredStaticLoader())
	}
	return p.systemLoader
}

func (p *pcoreImpl) EnvironmentLoader() eval.Loader {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.environmentLoader == nil {
		p.ensureSystemLoader()
		envLoader := p.systemLoader // TODO: Add proper environment loader
		s := p.settings[`module_path`]
		mds := make([]eval.ModuleLoader, 0)
		loadables := []eval.PathType{eval.PUPPET_FUNCTION_PATH, eval.PUPPET_DATA_TYPE_PATH, eval.PLAN_PATH, eval.TASK_PATH}
		if s.isSet() {
			modulesPath := s.get().String()
			fis, err := ioutil.ReadDir(modulesPath)
			if err == nil {
				for _, fi := range fis {
					if fi.IsDir() && eval.IsValidModuleName(fi.Name()) {
						ml := eval.NewFilebasedLoader(envLoader, filepath.Join(modulesPath, fi.Name()), fi.Name(), loadables...)
						mds = append(mds, ml)
					}
				}
			}
		}
		if len(mds) > 0 {
			p.environmentLoader = eval.NewDependencyLoader(mds)
		} else {
			p.environmentLoader = envLoader
		}
	}
	return p.environmentLoader
}

func (p *pcoreImpl) Loader(key string) eval.Loader {
	envLoader := p.EnvironmentLoader()
	if key == `` {
		return envLoader
	}
	if dp, ok := envLoader.(eval.DependencyLoader); ok {
		return dp.LoaderFor(key)
	}
	return nil
}

func (p *pcoreImpl) DefineSetting(key string, valueType eval.Type, dflt eval.Value) {
	s := &setting{name: key, valueType: valueType, defaultValue: dflt}
	if dflt != nil {
		s.set(dflt)
	}
	p.lock.Lock()
	p.settings[key] = s
	p.lock.Unlock()
}

func (p *pcoreImpl) Get(key string, defaultProducer eval.Producer) eval.Value {
	p.lock.RLock()
	v, ok := p.settings[key]
	p.lock.RUnlock()

	if ok {
		if v.isSet() {
			return v.get()
		}
		if defaultProducer == nil {
			return eval.UNDEF
		}
		return defaultProducer()
	}
	panic(fmt.Sprintf(`Attempt to access unknown setting '%s'`, key))
}

func (p *pcoreImpl) Logger() eval.Logger {
	return p.logger
}

func (p *pcoreImpl) RootContext() eval.Context {
	InitializePuppet()
	c := impl.WithParent(context.Background(), p.NewEvaluator(), eval.NewParentedLoader(p.EnvironmentLoader()), topImplRegistry)
	threadlocal.Init()
	threadlocal.Set(eval.PuppetContextKey, c)
	return c
}

func (p *pcoreImpl) Do(actor func(eval.Context)) {
	p.DoWithParent(p.RootContext(), actor)
}

func (p *pcoreImpl) DoWithParent(parentCtx context.Context, actor func(eval.Context)) {
	InitializePuppet()
	var ctx eval.Context
	if ec, ok := parentCtx.(eval.Context); ok {
		ctx = ec.Fork()
	} else {
		ctx = impl.WithParent(parentCtx, p.NewEvaluator(), eval.NewParentedLoader(p.EnvironmentLoader()), topImplRegistry)
	}
	threadlocal.Init()
	threadlocal.Set(eval.PuppetContextKey, ctx)
	actor(ctx)
}

func DoWithContext(testCtx eval.Context, actor func(eval.Context)) {
	threadlocal.Init()
	threadlocal.Set(eval.PuppetContextKey, testCtx)
	actor(testCtx)
}

func (p *pcoreImpl) Try(actor func(eval.Context) error) (err error) {
	return p.TryWithParent(p.RootContext(), actor)
}

func (p *pcoreImpl) TryWithParent(parentCtx context.Context, actor func(eval.Context) error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if ri, ok := r.(issue.Reported); ok {
				err = ri
			} else {
				panic(r)
			}
		}
	}()
	p.DoWithParent(parentCtx, func(c eval.Context) {
		err = actor(c)
	})
	return
}

func (p *pcoreImpl) NewEvaluator() eval.Evaluator {
	return p.NewEvaluatorWithLogger(p.logger)
}

func (p *pcoreImpl) NewEvaluatorWithLogger(logger eval.Logger) eval.Evaluator {
	return impl.NewEvaluator(logger)
}

func (p *pcoreImpl) NewParser() validator.ParserValidator {
	lo := make([]parser.Option, 0)
	var v validator.Validator
	wf := eval.GetSetting(`workflow`, types.Boolean_FALSE).(*types.BooleanValue).Bool()
	if eval.GetSetting(`tasks`, types.Boolean_FALSE).(*types.BooleanValue).Bool() {
		// Keyword 'plan' enabled. No resource expressions allowed in validation
		lo = append(lo, parser.PARSER_TASKS_ENABLED)
		if wf {
			lo = append(lo, parser.PARSER_WORKFLOW_ENABLED)
			v = validator.NewWorkflowChecker()
		} else {
			v = validator.NewTasksChecker()
		}
	} else {
		if wf {
			lo = append(lo, parser.PARSER_WORKFLOW_ENABLED)
			v = validator.NewWorkflowChecker()
		} else {
			v = validator.NewChecker(validator.Strict(p.Get(`strict`, nil).String()))
		}
	}
	return validator.NewParserValidator(parser.CreateParser(lo...), v)
}

func (p *pcoreImpl) Set(key string, value eval.Value) {
	p.lock.RLock()
	v, ok := p.settings[key]
	p.lock.RUnlock()

	if ok {
		v.set(value)
		return
	}
	panic(fmt.Sprintf(`Attempt to assign unknown setting '%s'`, key))
}
