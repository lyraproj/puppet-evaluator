package impl

import (
	"strings"

	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/types"
)

type (
	BasicScope struct {
		scopes  []map[string]eval.Value
		mutable bool
	}

	parentedScope struct {
		BasicScope
		parent eval.Scope
	}
)

// NewScope creates a new Scope instance that in turn consists of a stack of ephemeral scopes. If
// the mutable flag is true, then all ephemeral scopes except the one that represents the global
// scope considered mutable.
func NewScope(mutable bool) eval.Scope {
	return &BasicScope{[]map[string]eval.Value{make(map[string]eval.Value, 8)}, mutable}
}

// NewParentedScope creates a scope that will override its parent. When a value isn't found in this
// scope, the search continues in the parent scope.
//
// All new or updated values will end up in this scope, i.e. no modifications are ever propagated to
// the parent scope.
func NewParentedScope(parent eval.Scope, mutable bool) eval.Scope {
	return &parentedScope{BasicScope{[]map[string]eval.Value{make(map[string]eval.Value, 8)}, mutable}, parent}
}

func NewScope2(h *types.HashValue, mutable bool) eval.Scope {
	top := make(map[string]eval.Value, h.Len())
	h.EachPair(func(k, v eval.Value) { top[k.String()] = v })
	return &BasicScope{[]map[string]eval.Value{top}, mutable}
}

// No key can ever start with '::' or a capital letter
var groupKey = `::R`

func (e *BasicScope) RxGet(index int) (value eval.Value, found bool) {
	// Variable is in integer form. An attempt is made to find a Regexp result group
	// in this scope using the special key '::R'. No attempt is made to traverse parent
	// scopes.
	if r, ok := e.scopes[len(e.scopes)-1][groupKey]; ok {
		if gv, ok := r.(*types.ArrayValue); ok && index < gv.Len() {
			return gv.At(index), true
		}
	}
	return eval.Undef, false
}

func (e *BasicScope) WithLocalScope(producer eval.Producer) eval.Value {
	epCount := len(e.scopes)
	defer func() {
		// Pop all ephemerals
		e.scopes = e.scopes[:epCount]
	}()
	e.scopes = append(e.scopes, make(map[string]eval.Value, 8))
	result := producer()
	return result
}

func (e *BasicScope) Fork() eval.Scope {
	clone := &BasicScope{}
	clone.copyFrom(e)
	return clone
}

func (e *BasicScope) copyFrom(src *BasicScope) {
	e.mutable = src.mutable
	e.scopes = make([]map[string]eval.Value, len(src.scopes))
	for i, s := range src.scopes {
		cm := make(map[string]eval.Value, len(s))
		for k, v := range s {
			cm[k] = v
		}
		e.scopes[i] = cm
	}
}

func (e *BasicScope) Get(name string) (value eval.Value, found bool) {
	if strings.HasPrefix(name, `::`) {
		if value, found = e.scopes[0][name[2:]]; found {
			return
		}
		return eval.Undef, false
	}

	for idx := len(e.scopes) - 1; idx >= 0; idx-- {
		if value, found = e.scopes[idx][name]; found {
			return
		}
	}
	return eval.Undef, false
}

func (e *BasicScope) RxSet(variables []string) {
	// Assign the regular expression groups to an array value using the special key
	// '::R'. This overwrites an previous assignment in this scope
	varStrings := make([]eval.Value, len(variables))
	for idx, v := range variables {
		varStrings[idx] = types.WrapString(v)
	}
	e.scopes[len(e.scopes)-1][groupKey] = types.WrapValues(varStrings)
}

func (e *BasicScope) Set(name string, value eval.Value) bool {
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

func (e *BasicScope) State(name string) eval.VariableState {
	if strings.HasPrefix(name, `::`) {
		// Shortcut to global scope
		_, ok := e.scopes[0][name[2:]]
		if ok {
			return eval.Global
		}
		return eval.NotFound
	}

	for idx := len(e.scopes) - 1; idx >= 0; idx-- {
		if _, ok := e.scopes[idx][name]; ok {
			if idx == 0 {
				return eval.Global
			}
			return eval.Local
		}
	}
	return eval.NotFound
}

func (e *parentedScope) Fork() eval.Scope {
	clone := &parentedScope{}
	clone.copyFrom(&e.BasicScope)
	clone.parent = e.parent.Fork()
	return clone
}

func (e *parentedScope) Get(name string) (value eval.Value, found bool) {
	value, found = e.BasicScope.Get(name)
	if !found {
		value, found = e.parent.Get(name)
	}
	return
}

func (e *parentedScope) Set(name string, value eval.Value) bool {
	scopeIdx := 0
	if strings.HasPrefix(name, `::`) {
		name = name[2:]
	} else {
		scopeIdx = len(e.scopes) - 1
	}
	if scopeIdx == 0 && e.parent.State(name) == eval.Global {
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

func (e *parentedScope) State(name string) eval.VariableState {
	s := e.BasicScope.State(name)
	if s != 0 {
		return s
	}
	return e.parent.State(name)
}
