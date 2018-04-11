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
	_ "github.com/puppetlabs/go-evaluator/functions"
	"github.com/puppetlabs/go-evaluator/resource"
	"github.com/puppetlabs/go-parser/parser"
	"github.com/puppetlabs/go-parser/validator"
	"context"
	"github.com/puppetlabs/go-parser/issue"
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
var puppet = &pcoreImpl{}

func init() {
	eval.Puppet = puppet
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
	puppet.settings = make(map[string]*setting, 32)
	puppet.DefineSetting(`environment`, types.DefaultStringType(), types.WrapString(`production`))
	puppet.DefineSetting(`environmentpath`, types.DefaultStringType(), nil)
	puppet.DefineSetting(`module_path`, types.DefaultStringType(), nil)
	puppet.DefineSetting(`strict`, types.NewEnumType([]string{`off`, `warning`, `error`}, true), types.WrapString(`warning`))
	puppet.DefineSetting(`tasks`, types.DefaultBooleanType(), types.WrapBoolean(false))

	c := impl.NewContext(puppet.NewEvaluator(), eval.StaticLoader().(eval.DefiningLoader), nil)
	c.ResolveResolvables()
	resource.InitBuiltinResources()
	c.WithLoader(eval.StaticResourceLoader()).ResolveResolvables()
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
	if p.settings[`tasks`].get().(*types.BooleanValue).Bool() {
		return eval.StaticLoader()
	}
	return eval.StaticResourceLoader()
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

func (p *pcoreImpl) DefineSetting(key string, valueType eval.PType, dflt eval.PValue) {
	s := &setting{name: key, valueType: valueType, defaultValue: dflt}
	if dflt != nil {
		s.set(dflt)
	}
	p.lock.Lock()
	p.settings[key] = s
	p.lock.Unlock()
}

func (p *pcoreImpl) Get(key string, defaultProducer eval.Producer) eval.PValue {
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

func (p *pcoreImpl) Do(actor func(eval.Context)) (err error) {
	return p.DoWithParent(context.Background(), actor)
}

func (p *pcoreImpl) DoWithParent(parentCtx context.Context, actor func(eval.Context)) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if ri, ok := r.(*issue.Reported); ok {
				err = ri
			} else {
				panic(r)
			}
		}
	}()
	InitializePuppet()
	actor(p.newContext(parentCtx))
	return
}

func (p *pcoreImpl) Produce(producer func(eval.Context) (eval.PValue, error)) (value eval.PValue, err error) {
	return p.ProduceWithParent(context.Background(), producer)
}

func (p *pcoreImpl) ProduceWithParent(parentCtx context.Context, producer func(eval.Context) (eval.PValue, error)) (value eval.PValue, err error) {
	defer func() {
		if r := recover(); r != nil {
			if ri, ok := r.(*issue.Reported); ok {
				err = ri
			} else {
				panic(r)
			}
		}
	}()
	InitializePuppet()
	value, err = producer(p.newContext(parentCtx))
	return
}

func (p *pcoreImpl) newContext(parent context.Context) eval.Context {
	return impl.WithParent(parent, p.NewEvaluator(), eval.NewParentedLoader(p.EnvironmentLoader()), nil)
}

func (p *pcoreImpl) NewEvaluator() eval.Evaluator {
	return p.NewEvaluatorWithLogger(p.logger)
}

func (p *pcoreImpl) NewEvaluatorWithLogger(logger eval.Logger) eval.Evaluator {
	if eval.GetSetting(`tasks`, types.Boolean_FALSE).(*types.BooleanValue).Bool() {
		// Just script evaluator. No resource expressions
		return impl.NewEvaluator(logger)
	}
	return resource.NewEvaluator(logger)
}

func (p *pcoreImpl) NewParser() validator.ParserValidator {
	lo := make([]parser.Option, 0)
	var v validator.Validator
	if eval.GetSetting(`tasks`, types.Boolean_FALSE).(*types.BooleanValue).Bool() {
		// Keyword 'plan' enabled. No resource expressions allowed in validation
		lo = append(lo, parser.PARSER_TASKS_ENABLED)
		v = validator.NewTasksChecker()
	} else {
		v = validator.NewChecker(validator.Strict(p.Get(`strict`, nil).String()))
	}
	return validator.NewParserValidator(parser.CreateParser(lo...), v)
}

func (p *pcoreImpl) Set(key string, value eval.PValue) {
	p.lock.RLock()
	v, ok := p.settings[key]
	p.lock.RUnlock()

	if ok {
		v.set(value)
		return
	}
	panic(fmt.Sprintf(`Attempt to assign unknown setting '%s'`, key))
}
