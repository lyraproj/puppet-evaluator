package types

import (
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/hash"
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

var TYPE_ANNOTATIONS = NewHashType(NewTypeType(annotationType_DEFAULT), DefaultHashType(), nil)

type annotatable struct {
	annotations         *HashValue
	resolvedAnnotations *HashValue
}

func (a *annotatable) Annotations() eval.OrderedMap {
	return a.resolvedAnnotations
}

func (a *annotatable) Resolve(c eval.Context) {
	ah := a.annotations
	if ah.IsEmpty() {
		a.resolvedAnnotations = _EMPTY_MAP
	} else {
		as := make([]*HashEntry, 0, ah.Len())
		ah.EachPair(func(k, v eval.Value) {
			at := k.(eval.ObjectType)
			as = append(as, WrapHashEntry(k, eval.New(c, at, v)))
		})
		a.resolvedAnnotations = WrapHash(as)
	}
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
