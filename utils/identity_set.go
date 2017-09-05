package utils

import (
  . "reflect"
)

type IdentitySet map[uintptr]bool

// Associate the given value to the set.
// Returns true if the value was added, false if it was already present.
func (m IdentitySet) Add(key interface{}) bool {
  p := identity(key)
  if m[p] {
    return false
  }
  m[p] = true
  return true
}

// Returns the value associated with the key identity.
func (m IdentitySet) Include(key interface{}) bool {
  _, ok := m[identity(key)]
  return ok
}

func identity(obj interface{}) uintptr {
  v := ValueOf(obj)
  t := v.Type()
  if t.Kind() == Ptr {
    t = t.Elem()
  }

  // All empty structs may share the same pointer in Go, so for empty structs, the
  // identity is their type
  ts := t.Size()
  if ts == 0 {
    return ValueOf(t).Pointer()
  }
  return v.Pointer()
}
