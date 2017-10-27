package errors

import (
	"fmt"
)

type (
	InstantiationError interface {
		TypeName() string
		Error() string
	}

	GenericError string

	ArgumentsError struct {
		typeName string
		error    string
	}

	IllegalArgument struct {
		typeName string
		error    string
		index    int
	}

	IllegalArgumentType struct {
		typeName string
		expected string
		actual   string
		index    int
	}

	IllegalArgumentCount struct {
		typeName string
		expected string
		actual   int
	}
)

func (e GenericError) Error() string {
	return string(e)
}

func (e *ArgumentsError) TypeName() string {
	return e.typeName
}

func (e *ArgumentsError) Error() string {
	return fmt.Sprintf("%s: %s", e.typeName, e.error)
}

func (e *IllegalArgument) TypeName() string {
	return e.typeName
}

func (e *IllegalArgument) Error() string {
	return fmt.Sprintf("%s argument %d: %s", e.typeName, e.index+1, e.error)
}

func (e *IllegalArgument) Index() int {
	return e.index
}

func (e *IllegalArgumentType) TypeName() string {
	return e.typeName
}

func (e *IllegalArgumentType) Index() int {
	return e.index
}

func (e *IllegalArgumentType) Expected() string {
	return e.expected
}

func (e *IllegalArgumentType) Actual() string {
	return e.actual
}

func (e *IllegalArgumentType) Error() string {
	return fmt.Sprintf("%s expected argument %d to be %s, got %s", e.typeName, e.index+1, e.expected, e.actual)
}

func (e *IllegalArgumentCount) TypeName() string {
	return e.typeName
}

func (e *IllegalArgumentCount) Error() string {
	return fmt.Sprintf("%s expecteds argument count to be %s, got %d", e.typeName, e.expected, e.actual)
}

func (e *IllegalArgumentCount) Expected() string {
	return e.expected
}

func (e *IllegalArgumentCount) Actual() int {
	return e.actual
}

// NewArgumentsError is a general error with the arguments such as min > max
func NewArgumentsError(name string, error string) InstantiationError {
	return &ArgumentsError{name, error}
}

func NewIllegalArgument(name string, index int, error string) InstantiationError {
	return &IllegalArgument{name, error, index}
}

func NewIllegalArgumentType(name string, index int, expected string, actual string) InstantiationError {
	return &IllegalArgumentType{name, expected, actual, index}
}

func NewIllegalArgumentCount(name string, expected string, actual int) InstantiationError {
	return &IllegalArgumentCount{name, expected, actual}
}
