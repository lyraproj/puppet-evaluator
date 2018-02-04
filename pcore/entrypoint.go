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
		systemLoader      eval.Loader
		environmentLoader eval.Loader
		moduleLoaders     map[string]eval.Loader
		settings          map[string]eval.Setting
	}
)

func NewPcore(logger eval.Logger) eval.Pcore {
	loader := eval.NewParentedLoader(eval.StaticLoader())
	settings := make(map[string]eval.Setting, 32)
	p := &pcoreImpl{systemLoader: loader, settings: settings}

	impl.ResolveResolvables(loader, logger)

	p.DefineSetting(`environment`, types.DefaultStringType(), types.WrapString(`production`))
	p.DefineSetting(`environmentpath`, types.DefaultStringType(), nil)
	p.DefineSetting(`module_path`, types.DefaultStringType(), nil)
	p.DefineSetting(`tasks`, types.DefaultBooleanType(), types.WrapBoolean(false))
	eval.Puppet = p
	return p
}

func (p *pcoreImpl) Reset() {
	p.environmentLoader = nil
	for _, s := range p.settings {
		s.Reset()
	}
}

func (p *pcoreImpl) SystemLoader() eval.Loader {
	return p.systemLoader
}

func (p *pcoreImpl) EnvironmentLoader() eval.Loader {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.environmentLoader == nil {
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
	if dp, ok := envLoader.(eval.DependencyLoaer); ok {
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
