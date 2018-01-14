package pcore

import (
	"fmt"
	. "github.com/puppetlabs/go-evaluator/eval"
	. "github.com/puppetlabs/go-evaluator/evaluator"
	. "github.com/puppetlabs/go-evaluator/types"
	_ "github.com/puppetlabs/go-evaluator/loader"
	_ "github.com/puppetlabs/go-evaluator/functions"
	"io/ioutil"
	"path/filepath"
	"sync"
)

type (
	setting struct {
		name         string
		value        PValue
		defaultValue PValue
		valueType    PType
	}

	pcoreImpl struct {
		lock              sync.Mutex
		systemLoader      Loader
		environmentLoader Loader
		moduleLoaders     map[string]Loader
		settings          map[string]Setting
	}
)

var Puppet Pcore

func NewPcore(logger Logger) Pcore {
	loader := NewParentedLoader(StaticLoader())
	settings := make(map[string]Setting, 32)
	p := &pcoreImpl{systemLoader: loader, settings: settings}

	ResolveGoFunctions(loader, logger)

	p.DefineSetting(`environment`, DefaultStringType(), WrapString(`production`))
	p.DefineSetting(`environmentpath`, DefaultStringType(), nil)
	p.DefineSetting(`module_path`, DefaultStringType(), nil)
	Puppet = p
	return p
}

func (p *pcoreImpl) Reset() {
	p.environmentLoader = nil
	for _, s := range p.settings {
		s.Reset()
	}
}

func (p *pcoreImpl) SystemLoader() Loader {
	return p.systemLoader
}

func (p *pcoreImpl) EnvironmentLoader() Loader {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.environmentLoader == nil {
		envLoader := p.systemLoader // TODO: Add proper environment loader
		s := p.settings[`module_path`]
		mds := make([]ModuleLoader, 0)
		loadables := []PathType{PUPPET_FUNCTION_PATH, PUPPET_DATA_TYPE_PATH, PLAN_PATH, TASK_PATH}
		if s.IsSet() {
			modulesPath := s.Get().String()
			fis, err := ioutil.ReadDir(modulesPath)
			if err == nil {
				for _, fi := range fis {
					if fi.IsDir() && IsValidModuleName(fi.Name()) {
						ml := NewFilebasedLoader(envLoader, filepath.Join(modulesPath, fi.Name()), fi.Name(), loadables...)
						mds = append(mds, ml)
					}
				}
			}
		}
		if len(mds) > 0 {
			p.environmentLoader = NewDependencyLoader(mds)
		} else {
			p.environmentLoader = envLoader
		}
	}
	return p.environmentLoader
}

func (p *pcoreImpl) Loader(key string) Loader {
	envLoader := p.EnvironmentLoader()
	if key == `` {
		return envLoader
	}
	if dp, ok := envLoader.(DependencyLoaer); ok {
		return dp.LoaderFor(key)
	}
	return nil
}

func (p *pcoreImpl) DefineSetting(key string, valueType PType, dflt PValue) {
	p.lock.Lock()
	defer p.lock.Unlock()

	s := &setting{name: key, valueType: valueType, defaultValue: dflt}
	if dflt != nil {
		s.Set(dflt)
	}
	p.settings[key] = s
}

func (p *pcoreImpl) Get(key string, defaultProducer Producer) PValue {
	p.lock.Lock()
	defer p.lock.Unlock()

	if v, ok := p.settings[key]; ok {
		if v.IsSet() {
			return v.Get()
		}
		if defaultProducer == nil {
			return UNDEF
		}
		return defaultProducer()
	}
	panic(fmt.Sprintf(`Attempt to access unknown setting '%s'`, key))
}

func (p *pcoreImpl) Set(key string, value PValue) {
	p.lock.Lock()
	defer p.lock.Unlock()

	if v, ok := p.settings[key]; ok {
		v.Set(value)
		return
	}
	panic(fmt.Sprintf(`Attempt to assign unknown setting '%s'`, key))
}

func (s *setting) Name() string {
	return s.name
}

func (s *setting) Get() PValue {
	return s.value
}

func (s *setting) Reset() {
	s.value = s.defaultValue
}

func (s *setting) Set(value PValue) {
	if !IsInstance(s.valueType, value) {
		panic(DescribeMismatch(fmt.Sprintf(`Setting '%s'`, s.name), s.valueType, DetailedValueType(value)))
	}
	s.value = value
}

func (s *setting) IsSet() bool {
	return s.value != nil // As opposed to UNDEF which is a proper value
}

func (s *setting) Type() PType {
	return s.valueType
}
