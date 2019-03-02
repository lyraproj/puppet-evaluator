package types

import (
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/hash"
)

var annotationTypeDefault = &objectType{
	annotatable:         annotatable{annotations: emptyMap},
	hashKey:             eval.HashKey("\x00tAnnotation"),
	name:                `Annotation`,
	parameters:          hash.EmptyStringHash,
	attributes:          hash.EmptyStringHash,
	functions:           hash.EmptyStringHash,
	equalityIncludeType: true,
	equality:            nil}

func DefaultAnnotationType() eval.Type {
	return annotationTypeDefault
}

var typeAnnotations = NewHashType(NewTypeType(annotationTypeDefault), DefaultHashType(), nil)

type annotatable struct {
	annotations         *HashValue
	resolvedAnnotations *HashValue
}

func (a *annotatable) Annotations(c eval.Context) eval.OrderedMap {
	if a.resolvedAnnotations == nil {
		ah := a.annotations
		if ah.IsEmpty() {
			a.resolvedAnnotations = emptyMap
		} else {
			as := make([]*HashEntry, 0, ah.Len())
			ah.EachPair(func(k, v eval.Value) {
				at := k.(eval.ObjectType)
				as = append(as, WrapHashEntry(k, eval.New(c, at, v)))
			})
			a.resolvedAnnotations = WrapHash(as)
		}
	}
	return a.resolvedAnnotations
}

func (a *annotatable) initialize(initHash *HashValue) {
	a.annotations = hashArg(initHash, keyAnnotations)
}

func (a *annotatable) initHash() *hash.StringHash {
	h := hash.NewStringHash(5)
	if a.annotations.Len() > 0 {
		h.Put(keyAnnotations, a.annotations)
	}
	return h
}
