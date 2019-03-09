package errors

import (
	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
)

type (
	Breaker struct {
		location issue.Location
	}

	StopIteration struct {
		Breaker
	}

	NextIteration struct {
		Breaker
		value px.Value
	}

	Return struct {
		Breaker
		value px.Value
	}

	InstantiationError interface {
		TypeName() string
		Error() string
	}

	GenericError string
)

func (e GenericError) Error() string {
	return string(e)
}

func (e *Breaker) Location() issue.Location {
	return e.location
}

func NewStopIteration(location issue.Location) *StopIteration {
	return &StopIteration{Breaker{location}}
}

func NewNextIteration(location issue.Location, value px.Value) *NextIteration {
	return &NextIteration{Breaker{location}, value}
}

func (e *NextIteration) Value() px.Value {
	return e.value
}

func NewReturn(location issue.Location, value px.Value) *Return {
	return &Return{Breaker{location}, value}
}

func (e *Return) Value() px.Value {
	return e.value
}
