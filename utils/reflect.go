package utils

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"reflect"
)

const tagName = "puppet"

func FieldName(c eval.Context, f *reflect.StructField) string {
	if tagHash, ok := TagHash(c, f); ok {
		if nv, ok := tagHash.Get4(`name`); ok {
			return nv.String()
		}
	}
	return CamelToSnakeCase(f.Name)
}

func TagHash(c eval.Context, f *reflect.StructField) (eval.KeyedValue, bool) {
	if tag := f.Tag.Get(tagName); tag != `` {
		tagExpr := c.ParseAndValidate(``, `{` + tag + `}`, true)
		if tagHash, ok := c.Evaluate(tagExpr).(eval.KeyedValue); ok {
			return tagHash, true
		}
	}
	return nil, false
}