package impl

type IdentitySet map[interface{}]bool

// Add the given value to the set.
// Returns true if the value was added, false if it was already present.
func (m IdentitySet) Add(key interface{}) bool {
	if m[key] {
		return false
	}
	m[key] = true
	return true
}

// Include returns true if the given key is present
func (m IdentitySet) Include(key interface{}) bool {
	_, ok := m[key]
	return ok
}
