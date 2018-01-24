package eval

import (
	"strings"

	. "github.com/puppetlabs/go-evaluator/evaluator"
	. "github.com/puppetlabs/go-evaluator/types"
)

type (
	BasicScope struct {
		scopes []map[string]PValue
	}
)

func NewScope() Scope {
	return &BasicScope{[]map[string]PValue{make(map[string]PValue, 8)}}
}

func NewScope2(h *HashValue) Scope {
	top := make(map[string]PValue, h.Len())
	for _, he := range h.EntriesSlice() {
		top[he.Key().String()] = he.Value()
	}
	return &BasicScope{[]map[string]PValue{top}}
}

// No key can ever start with '::' or a capital letter
var groupKey = `::R`

func (e *BasicScope) RxGet(index int) (value PValue, found bool) {
	// Variable is in integer form. An attempt is made to find a Regexp result group
	// in this scope using the special key '::R'. No attempt is made to traverse parent
	// scopes.
	if r, ok := e.scopes[len(e.scopes)-1][groupKey]; ok {
		if gv, ok := r.(*ArrayValue); ok && index < gv.Len() {
			return gv.At(index), true
		}
	}
	return UNDEF, false
}

func (e *BasicScope) WithLocalScope(producer ValueProducer) PValue {
	local := make([]map[string]PValue, len(e.scopes)+1)
	copy(local, e.scopes)
	return producer(&BasicScope{append(local, make(map[string]PValue, 8))})
}

func (e *BasicScope) Get(name string) (value PValue, found bool) {
	if strings.HasPrefix(name, `::`) {
		if value, found = e.scopes[0][name[2:]]; found {
			return
		}
		return UNDEF, false
	}

	for idx := len(e.scopes) - 1; idx >= 0; idx-- {
		if value, found = e.scopes[idx][name]; found {
			return
		}
	}
	return UNDEF, false
}

func (e *BasicScope) RxSet(variables []string) {
	// Assign the regular expression groups to an array value using the special key
	// '::R'. This overwrites an previous assignment in this scope
	varStrings := make([]PValue, len(variables))
	for idx, v := range variables {
		varStrings[idx] = WrapString(v)
	}
	e.scopes[len(e.scopes)-1][groupKey] = WrapArray(varStrings)
}

func (e *BasicScope) Set(name string, value PValue) bool {
	var current map[string]PValue
	if strings.HasPrefix(name, `::`) {
		name = name[2:]
		current = e.scopes[0]
	} else {
		current = e.scopes[len(e.scopes)-1]
	}
	if _, found := current[name]; !found {
		current[name] = value
		return true
	}
	return false
}
