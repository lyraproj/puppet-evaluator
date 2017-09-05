package types

import . "github.com/puppetlabs/go-evaluator/utils"

const (
  NO_SELF_RECURSION = rcState(0)
  SELF_RECURSION_IN_THIS = rcState(1)
  SELF_RECURSION_IN_THAT = rcState(2)
  SELF_RECURSION_IN_BOTH = SELF_RECURSION_IN_THIS | SELF_RECURSION_IN_THAT
)

type (
  rcState int

  rcGuard struct {
    state rcState
    thatMap IdentitySet
    thisMap IdentitySet
    recursiveThatMap IdentitySet
    recursiveThisMap IdentitySet
  }

  guardedFunc func(state rcState) interface{}
)

// Add the given argument as 'that' and call block with the resulting state. Restore state after call.
func (g *rcGuard) withThat(instance interface{}, block guardedFunc) (result interface{}) {
  if g.getThatMap().Add(instance) {
    result = block(g.state)
  } else {
    g.getRecursiveThatMap().Add(instance)
    if (g.state & SELF_RECURSION_IN_THAT) == 0 {
      g.state |= SELF_RECURSION_IN_THAT
      result = block(g.state)
      g.state &^= SELF_RECURSION_IN_THAT
    } else {
      result = block(g.state)
    }
  }
  return
}

// Add the given argument as 'this' and call block with the resulting state. Restore state after call.
func (g *rcGuard) withThis(instance interface{}, block guardedFunc) (result interface{}) {
  if g.getThisMap().Add(instance) {
    result = block(g.state)
  } else {
    g.getRecursiveThisMap().Add(instance)
    if (g.state & SELF_RECURSION_IN_THIS) == 0 {
      g.state |= SELF_RECURSION_IN_THIS
      result = block(g.state)
      g.state &^= SELF_RECURSION_IN_THIS
    } else {
      result = block(g.state)
    }
  }
  return
}

// Checks if rc was detected for the given argument in the 'that' context
func (g* rcGuard) recursiveThat(instance interface{}) bool {
  return g.recursiveThatMap != nil && g.recursiveThatMap.Include(instance)
}

// Checks if rc was detected for the given argument in the 'this' context
func (g* rcGuard) recursiveThis(instance interface{}) bool {
  return g.recursiveThisMap != nil && g.recursiveThisMap.Include(instance)
}

func (g* rcGuard) getThatMap() IdentitySet {
  if g.thatMap == nil {
    g.thatMap = make(IdentitySet, 8)
  }
  return g.thatMap
}

func (g* rcGuard) getThisMap() IdentitySet {
  if g.recursiveThatMap == nil {
    g.recursiveThatMap = make(IdentitySet, 8)
  }
  return g.recursiveThatMap
}

func (g* rcGuard) getRecursiveThatMap() IdentitySet {
  if g.thisMap == nil {
    g.thisMap = make(IdentitySet, 8)
  }
  return g.thisMap
}

func (g* rcGuard) getRecursiveThisMap() IdentitySet {
  if g.recursiveThisMap == nil {
    g.recursiveThisMap = make(IdentitySet, 8)
  }
  return g.recursiveThisMap
}

