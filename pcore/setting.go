package pcore

import (
	"fmt"
	"github.com/puppetlabs/go-evaluator/eval"
)

type (
	setting struct {
		name         string
		value        eval.PValue
		defaultValue eval.PValue
		valueType    eval.PType
	}
)

func (s *setting) get() eval.PValue {
	return s.value
}

func (s *setting) reset() {
	s.value = s.defaultValue
}

func (s *setting) set(value eval.PValue) {
	if !eval.IsInstance(nil, s.valueType, value) {
		panic(eval.DescribeMismatch(fmt.Sprintf(`Setting '%s'`, s.name), s.valueType, eval.DetailedValueType(value)))
	}
	s.value = value
}

func (s *setting) isSet() bool {
	return s.value != nil // As opposed to UNDEF which is a proper value
}
