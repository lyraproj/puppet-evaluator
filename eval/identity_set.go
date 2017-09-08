package eval

type IdentitySet map[interface{}]bool

// Associate the given value to the set.
// Returns true if the value was added, false if it was already present.
func (m IdentitySet) Add(key interface{}) bool {
	if m[key] {
		return false
	}
	m[key] = true
	return true
}

// Returns the value associated with the key identity.
func (m IdentitySet) Include(key interface{}) bool {
	_, ok := m[key]
	return ok
}
