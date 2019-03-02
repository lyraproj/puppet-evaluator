package types

import (
	"github.com/lyraproj/puppet-evaluator/eval"
	"reflect"
)

type taggedType struct {
	typ         reflect.Type
	puppetTags  map[string]string
	annotations eval.OrderedMap
}

func init() {
	eval.NewTaggedType = func(typ reflect.Type, puppetTags map[string]string) eval.AnnotatedType {
		return &taggedType{typ, puppetTags, emptyMap}
	}

	eval.NewAnnotatedType = func(typ reflect.Type, puppetTags map[string]string, annotations eval.OrderedMap) eval.AnnotatedType {
		return &taggedType{typ, puppetTags, annotations}
	}
}

func (tg *taggedType) Annotations() eval.OrderedMap {
	return tg.annotations
}

func (tg *taggedType) Type() reflect.Type {
	return tg.typ
}

func (tg *taggedType) Tags() map[string]eval.OrderedMap {
	fs := Fields(tg.typ)
	nf := len(fs)
	tags := make(map[string]eval.OrderedMap, 7)
	if nf > 0 {
		for i, f := range fs {
			if i == 0 && f.Anonymous {
				// Parent
				continue
			}
			if f.PkgPath != `` {
				// Unexported
				continue
			}
			if ft, ok := TagHash(&f); ok {
				tags[f.Name] = ft
			}
		}
	}
	if tg.puppetTags != nil && len(tg.puppetTags) > 0 {
		for k, v := range tg.puppetTags {
			if h, ok := ParseTagHash(v); ok {
				tags[k] = h
			}
		}
	}
	return tags
}
