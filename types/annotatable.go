package types

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/hash"
)

var annotationType_DEFAULT = &objectType{
	annotatable: annotatable{annotations: _EMPTY_MAP},
	hashKey:     eval.HashKey("\x00tAnnotation"),
	name:        `Annotation`,
	parameters:  hash.EMPTY_STRINGHASH,
	attributes:  hash.EMPTY_STRINGHASH,
	functions:   hash.EMPTY_STRINGHASH,
	equality:    nil}

func DefaultAnnotationType() eval.Type {
	return annotationType_DEFAULT
}

var TYPE_ANNOTATIONS = NewHashType(annotationType_DEFAULT, DefaultHashType(), nil)

type annotatable struct {
	annotations *HashValue
}

func (a *annotatable) initialize(initHash *HashValue) {
	a.annotations = hashArg(initHash, KEY_ANNOTATIONS)
}

func (a *annotatable) initHash() *hash.StringHash {
	h := hash.NewStringHash(5)
	if a.annotations.Len() > 0 {
		h.Put(KEY_ANNOTATIONS, a.annotations)
	}
	return h
}
