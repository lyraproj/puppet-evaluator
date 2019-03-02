package hash

import (
	"fmt"

	"github.com/lyraproj/puppet-evaluator/eval"
)

// Mutable and order preserving hash with string keys and arbitrary values. Used, among other things, by the
// Object type to store parameters, attributes, and functions

type (
	stringEntry struct {
		key   string
		value interface{}
	}

	StringHash struct {
		entries []*stringEntry
		index   map[string]int
		frozen  bool
	}

	frozenError struct {
		key string
	}
)

var EmptyStringHash = &StringHash{[]*stringEntry{}, map[string]int{}, true}

func (f *frozenError) Error() string {
	return fmt.Sprintf("attempt to add, modify, or delete key '%s' in a frozen StringHash", f.key)
}

// NewStringHash returns an empty *StringHash initialized with given capacity
func NewStringHash(capacity int) *StringHash {
	return &StringHash{make([]*stringEntry, 0, capacity), make(map[string]int, capacity), false}
}

// Copy returns a shallow copy of this hash, i.e. each key and value is not cloned
func (h *StringHash) Copy() *StringHash {
	entries := make([]*stringEntry, len(h.entries))
	for i, e := range h.entries {
		entries[i] = &stringEntry{e.key, e.value}
	}
	index := make(map[string]int, len(h.index))
	for k, v := range h.index {
		index[k] = v
	}
	return &StringHash{entries, index, false}
}

// EachKey calls the given consumer function once for each key in this hash
func (h *StringHash) EachKey(consumer func(key string)) {
	for _, e := range h.entries {
		consumer(e.key)
	}
}

// AllPair calls the given function once for each key/value pair in this hash. Return
// true if all invocations returned true. False otherwise.
// The method returns true if the hash i empty.
func (h *StringHash) AllPair(f func(key string, value interface{}) bool) bool {
	for _, e := range h.entries {
		if !f(e.key, e.value) {
			return false
		}
	}
	return true
}

// AnyPair calls the given function once for each key/value pair in this hash. Return
// true when an invocation returns true. False otherwise.
// The method returns false if the hash i empty.
func (h *StringHash) AnyPair(f func(key string, value interface{}) bool) bool {
	for _, e := range h.entries {
		if f(e.key, e.value) {
			return true
		}
	}
	return false
}

// EachPair calls the given consumer function once for each key/value pair in this hash
func (h *StringHash) EachPair(consumer func(key string, value interface{})) {
	for _, e := range h.entries {
		consumer(e.key, e.value)
	}
}

// EachValue calls the given consumer function once for each value in this hash
func (h *StringHash) EachValue(consumer func(value interface{})) {
	for _, e := range h.entries {
		consumer(e.value)
	}
}

// Equals compares two hashes for equality. Hashes are considered equal if the have
// the same size and contains the same key/value associations irrespective of order
func (h *StringHash) Equals(other interface{}, g eval.Guard) bool {
	oh, ok := other.(*StringHash)
	if !ok || len(h.entries) != len(oh.entries) {
		return false
	}

	for _, e := range h.entries {
		oi, ok := oh.index[e.key]
		if !(ok && eval.GuardedEquals(e.value, oh.entries[oi].value, g)) {
			return false
		}
	}
	return true
}

// Freeze prevents further changes to the hash
func (h *StringHash) Freeze() {
	h.frozen = true
}

// Get returns a value from the hash or the given default if no value was found
func (h *StringHash) Get(key string, dflt interface{}) interface{} {
	if p, ok := h.index[key]; ok {
		return h.entries[p].value
	}
	return dflt
}

// Get2 returns a value from the hash or the value returned by given default function if no value was found
func (h *StringHash) Get2(key string, dflt func() interface{}) interface{} {
	if p, ok := h.index[key]; ok {
		return h.entries[p].value
	}
	return dflt()
}

// Get3 returns a value from the hash or nil together with a boolean to indicate if the key was present or not
func (h *StringHash) Get3(key string) (interface{}, bool) {
	if p, ok := h.index[key]; ok {
		return h.entries[p].value, true
	}
	return nil, false
}

// Delete the entry for the given key from the hash. Returns the old value or nil if not found
func (h *StringHash) Delete(key string) (oldValue interface{}) {
	if h.frozen {
		panic(frozenError{key})
	}
	index := h.index
	oldValue = nil
	if p, ok := index[key]; ok {
		oldValue = h.entries[p].value
		delete(h.index, key)
		for k, v := range index {
			if v > p {
				index[k] = p - 1
			}
		}
		ne := make([]*stringEntry, len(h.entries)-1)
		for i, e := range h.entries {
			if i < p {
				ne[i] = e
			} else if i > p {
				ne[i-1] = e
			}
		}
		h.entries = ne
	}
	return
}

// Includes returns true if the hash contains the given key
func (h *StringHash) Includes(key string) bool {
	_, ok := h.index[key]
	return ok
}

// IsEmpty returns true if the hash has no entries
func (h *StringHash) IsEmpty() bool {
	return len(h.entries) == 0
}

// Keys returns the keys of the hash in the order that they were first entered
func (h *StringHash) Keys() []string {
	keys := make([]string, len(h.entries))
	for i, e := range h.entries {
		keys[i] = e.key
	}
	return keys
}

// Merge this hash with the other hash giving the other precedence. A new hash is returned
func (h *StringHash) Merge(other *StringHash) (merged *StringHash) {
	merged = h.Copy()
	merged.PutAll(other)
	return
}

// Put adds a new key/value association to the hash or replace the value of an existing association
func (h *StringHash) Put(key string, value interface{}) (oldValue interface{}) {
	if h.frozen {
		panic(frozenError{key})
	}
	if p, ok := h.index[key]; ok {
		e := h.entries[p]
		oldValue = e.value
		e.value = value
	} else {
		oldValue = nil
		h.index[key] = len(h.entries)
		h.entries = append(h.entries, &stringEntry{key, value})
	}
	return
}

// PutAll merges this hash with the other hash giving the other precedence. A new hash is returned
func (h *StringHash) PutAll(other *StringHash) {
	for _, e := range other.entries {
		h.Put(e.key, e.value)
	}
}

// Len returns the number of entries in the hash
func (h *StringHash) Len() int {
	return len(h.entries)
}

// Values returns the values of the hash in the order that their respective keys were first entered
func (h *StringHash) Values() []interface{} {
	values := make([]interface{}, len(h.entries))
	for i, e := range h.entries {
		values[i] = e.value
	}
	return values
}
