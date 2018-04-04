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
)

type (
	setting struct {
		name         string
		value        eval.PValue
		defaultValue eval.PValue
		valueType    eval.PType
	}

	pcoreImpl struct {
		lock              sync.RWMutex
		logger            eval.Logger
		systemLoader      eval.Loader
		environmentLoader eval.Loader
		moduleLoaders     map[string]eval.Loader
		settings          map[string]eval.Setting
	}
)

var staticLock sync.Mutex

func InitializePuppet() {
	// First call initializes the static loader. There can be only one since it receives
	// most of its contents from Go init() functions
	staticLock.Lock()
	defer staticLock.Unlock()

	if eval.Puppet != nil {
		return
	}

	logger := eval.NewStdLogger()
	impl.ResolveResolvables(eval.StaticLoader().(eval.DefiningLoader), logger)

	p := &pcoreImpl{settings: make(map[string]eval.Setting, 32), logger: logger}
	p.DefineSetting(`environment`, types.DefaultStringType(), types.WrapString(`production`))
	p.DefineSetting(`environmentpath`, types.DefaultStringType(), nil)
	p.DefineSetting(`module_path`, types.DefaultStringType(), nil)
	p.DefineSetting(`strict`, types.NewEnumType([]string{`off`, `warning`, `error`}, true), types.WrapString(`warning`))
	p.DefineSetting(`tasks`, types.DefaultBooleanType(), types.WrapBoolean(false))
	eval.Puppet = p
}

func (p *pcoreImpl) Reset() {
	p.lock.Lock()
	p.systemLoader = nil
	p.environmentLoader = nil
	for _, s := range p.settings {
		s.Reset()
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
	if p.settings[`tasks`].Get().(*types.BooleanValue).Bool() {
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

func (p *pcoreImpl) ResolveResolvables(loader eval.DefiningLoader) {
	impl.ResolveResolvables(loader, p.logger)
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
		if s.IsSet() {
			modulesPath := s.Get().String()
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
		s.Set(dflt)
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
		if v.IsSet() {
			return v.Get()
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

func (p *pcoreImpl) NewEvaluator() eval.Evaluator {
	return p.NewEvaluatorWithLogger(p.logger)
}

func (p *pcoreImpl) NewEvaluatorWithLogger(logger eval.Logger) eval.Evaluator {
	loader := eval.NewParentedLoader(p.EnvironmentLoader())
	if eval.GetSetting(`tasks`, types.Boolean_FALSE).(*types.BooleanValue).Bool() {
		// Just script evaluator. No resource expressions
		return impl.NewEvaluator(loader, logger)
	}
	return resource.NewEvaluator(loader, logger)
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
		v.Set(value)
		return
	}
	panic(fmt.Sprintf(`Attempt to assign unknown setting '%s'`, key))
}

func (s *setting) Name() string {
	return s.name
}

func (s *setting) Get() eval.PValue {
	return s.value
}

func (s *setting) Reset() {
	s.value = s.defaultValue
}

func (s *setting) Set(value eval.PValue) {
	if !eval.IsInstance(s.valueType, value) {
		panic(eval.DescribeMismatch(fmt.Sprintf(`Setting '%s'`, s.name), s.valueType, eval.DetailedValueType(value)))
	}
	s.value = value
}

func (s *setting) IsSet() bool {
	return s.value != nil // As opposed to UNDEF which is a proper value
}

func (s *setting) Type() eval.PType {
	return s.valueType
}
