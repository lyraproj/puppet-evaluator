package evaluator

import (
	"strings"

	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
	"github.com/lyraproj/puppet-evaluator/pdsl"
)

type (
	BasicScope struct {
		scopes  []map[string]px.Value
		mutable bool
	}

	parentedScope struct {
		BasicScope
		parent pdsl.Scope
	}
)

// NewScope creates a new Scope instance that in turn consists of a stack of ephemeral scopes. If
// the mutable flag is true, then all ephemeral scopes except the one that represents the global
// scope considered mutable.
func NewScope(mutable bool) pdsl.Scope {
	return &BasicScope{[]map[string]px.Value{make(map[string]px.Value, 8)}, mutable}
}

// NewParentedScope creates a scope that will override its parent. When a value isn't found in this
// scope, the search continues in the parent scope.
//
// All new or updated values will end up in this scope, i.e. no modifications are ever propagated to
// the parent scope.
func NewParentedScope(parent pdsl.Scope, mutable bool) pdsl.Scope {
	return &parentedScope{BasicScope{[]map[string]px.Value{make(map[string]px.Value, 8)}, mutable}, parent}
}

func NewScope2(h *types.Hash, mutable bool) pdsl.Scope {
	top := make(map[string]px.Value, h.Len())
	h.EachPair(func(k, v px.Value) { top[k.String()] = v })
	return &BasicScope{[]map[string]px.Value{top}, mutable}
}

// No key can ever start with '::' or a capital letter
var groupKey = `::R`

func (e *BasicScope) RxGet(index int) (value px.Value, found bool) {
	// Variable is in integer form. An attempt is made to find a Regexp result group
	// in this scope using the special key '::R'. No attempt is made to traverse parent
	// scopes.
	if r, ok := e.scopes[len(e.scopes)-1][groupKey]; ok {
		if gv, ok := r.(*types.Array); ok && index < gv.Len() {
			return gv.At(index), true
		}
	}
	return px.Undef, false
}

func (e *BasicScope) WithLocalScope(producer px.Producer) px.Value {
	epCount := len(e.scopes)
	defer func() {
		// Pop all ephemerals
		e.scopes = e.scopes[:epCount]
	}()
	e.scopes = append(e.scopes, make(map[string]px.Value, 8))
	result := producer()
	return result
}

func (e *BasicScope) Fork() pdsl.Scope {
	clone := &BasicScope{}
	clone.copyFrom(e)
	return clone
}

func (e *BasicScope) copyFrom(src *BasicScope) {
	e.mutable = src.mutable
	e.scopes = make([]map[string]px.Value, len(src.scopes))
	for i, s := range src.scopes {
		cm := make(map[string]px.Value, len(s))
		for k, v := range s {
			cm[k] = v
		}
		e.scopes[i] = cm
	}
}

func (e *BasicScope) Get(nv px.Value) (value px.Value, found bool) {
	return e.Get2(nv.String())
}

func (e *BasicScope) Get2(name string) (value px.Value, found bool) {
	if strings.HasPrefix(name, `::`) {
		if value, found = e.scopes[0][name[2:]]; found {
			return
		}
		return px.Undef, false
	}

	for idx := len(e.scopes) - 1; idx >= 0; idx-- {
		if value, found = e.scopes[idx][name]; found {
			return
		}
	}
	return px.Undef, false
}

func (e *BasicScope) RxSet(variables []string) {
	// Assign the regular expression groups to an array value using the special key
	// '::R'. This overwrites an previous assignment in this scope
	varStrings := make([]px.Value, len(variables))
	for idx, v := range variables {
		varStrings[idx] = types.WrapString(v)
	}
	e.scopes[len(e.scopes)-1][groupKey] = types.WrapValues(varStrings)
}

func (e *BasicScope) Set(name string, value px.Value) bool {
	scopeIdx := 0
	if strings.HasPrefix(name, `::`) {
		name = name[2:]
	} else {
		scopeIdx = len(e.scopes) - 1
	}
	current := e.scopes[scopeIdx]

	if e.mutable && scopeIdx > 0 {
		current[name] = value
		return true
	}

	if _, found := current[name]; !found {
		current[name] = value
		return true
	}
	return false
}

func (e *BasicScope) State(name string) pdsl.VariableState {
	if strings.HasPrefix(name, `::`) {
		// Shortcut to global scope
		_, ok := e.scopes[0][name[2:]]
		if ok {
			return pdsl.Global
		}
		return pdsl.NotFound
	}

	for idx := len(e.scopes) - 1; idx >= 0; idx-- {
		if _, ok := e.scopes[idx][name]; ok {
			if idx == 0 {
				return pdsl.Global
			}
			return pdsl.Local
		}
	}
	return pdsl.NotFound
}

func (e *parentedScope) Fork() pdsl.Scope {
	clone := &parentedScope{}
	clone.copyFrom(&e.BasicScope)
	clone.parent = e.parent.Fork()
	return clone
}

func (e *parentedScope) Get(nv px.Value) (value px.Value, found bool) {
	return e.Get2(nv.String())
}

func (e *parentedScope) Get2(name string) (value px.Value, found bool) {
	value, found = e.BasicScope.Get2(name)
	if !found {
		value, found = e.parent.Get2(name)
	}
	return
}

func (e *parentedScope) Set(name string, value px.Value) bool {
	scopeIdx := 0
	if strings.HasPrefix(name, `::`) {
		name = name[2:]
	} else {
		scopeIdx = len(e.scopes) - 1
	}
	if scopeIdx == 0 && e.parent.State(name) == pdsl.Global {
		// Attempt to override global declared in parent. Only $pnr can do
		// that and the override ends up here, not in the parent.
		if name == `pnr` {
			e.scopes[0][name] = value
			return true
		}
		return false
	}
	current := e.scopes[scopeIdx]

	if e.mutable && scopeIdx > 0 {
		current[name] = value
		return true
	}

	if _, found := current[name]; !found {
		current[name] = value
		return true
	}
	return false
}

func (e *parentedScope) State(name string) pdsl.VariableState {
	s := e.BasicScope.State(name)
	if s != 0 {
		return s
	}
	return e.parent.State(name)
}
