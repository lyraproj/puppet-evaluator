package pdsl

import "github.com/lyraproj/pcore/px"

type VariableState int

const NotFound = VariableState(0)
const Global = VariableState(1)
const Local = VariableState(2)

type (
	// A Scope is the container for Puppet variables. It is constituted of
	// a stack of ephemeral scopes where each ephemeral scope represents a local
	// variable container that overrides the previous entry.
	//
	// The first ephemeral scope on the stack is the container of global variables.
	//
	// A Scope is not re-entrant and two important rules must be honored when working
	// with scopes.
	//
	// 1. New go-routines must use the method NewParentedScope to create a new modifiable
	// scope that shares a read-only parent scope.
	//
	// 2. A scope must be considered immutable once it is used as a parent scope.
	Scope interface {
		// WithLocalScope pushes an ephemeral scope and calls the producer. The
		// ephemeral scope is guaranteed to be popped before this method returns.
		WithLocalScope(producer px.Producer) px.Value

		// Fork Creates a copy of this scope. If the scope is parented, the copy will
		// contain a fork of the parent
		Fork() Scope

		// Get returns a named variable from this scope together with a boolean indicating
		// if the variable was found or not
		Get(name px.Value) (value px.Value, found bool)

		// Get returns a named variable from this scope together with a boolean indicating
		// if the variable was found or not
		Get2(name string) (value px.Value, found bool)

		// Set assigns a named variable to this scope provided that the name didn't
		// already exist. It returns a boolean indicating success.
		Set(name string, value px.Value) bool

		// RxSet assigns the result of a regular expression match to this scope. It
		// will be available until the current ephemeral scope it is popped or a new
		// call to RxSet replaces it.
		RxSet(variables []string)

		// RxGet returns a numeric variable that has been assigned by RxSet together
		// with a boolean indicating success.
		RxGet(index int) (value px.Value, found bool)

		// State returns NotFound, Global, or Local
		State(name string) VariableState
	}
)
