package js2ast

import (
	"github.com/puppetlabs/go-issues/issue"
)

const (
	EVAL_JS_ATTRIBUTE_NAME_MUST_BE_STRING = `EVAL_JS_ATTRIBUTE_NAME_MUST_BE_STRING`
	EVAL_JS_ILLEGAL_TYPE = `EVAL_JS_ILLEGAL_TYPE`
	EVAL_JS_RESOURCE_ARGCOUNT = `EVAL_JS_RESOURCE_ARGCOUNT`
	EVAL_JS_UNHANDLED_EXPRESSION = `EVAL_JS_UNHANDLED_EXPRESSION`
	EVAL_JS_UNHANDLED_STATEMENT = `EVAL_JS_UNHANDLED_STATEMENT`
	EVAL_JS_INVALID_NUMBER       = `EVAL_JS_INVALID_NUMBER`
)

func init() {
	issue.Hard2(EVAL_JS_ATTRIBUTE_NAME_MUST_BE_STRING, `names of attributes must be strings, got %{actual}`,
		issue.HF{`actual`: issue.Label})

	issue.Hard2(EVAL_JS_ILLEGAL_TYPE, `the value must be %{expected}. Got %{actual}`,
		issue.HF{`expected`: issue.A_an, `actual`: issue.A_an})

	issue.Hard(EVAL_JS_RESOURCE_ARGCOUNT, `a call to resource expects one Hash argument`)

	issue.Hard(EVAL_JS_UNHANDLED_EXPRESSION, `the JavaScript transformer cannot convert an expression of type %<expr>T`)

	issue.Hard(EVAL_JS_UNHANDLED_STATEMENT, `the JavaScript transformer cannot convert a statment of type %<stmt>T`)

	issue.Hard(EVAL_JS_INVALID_NUMBER, `the string '%{src}' cannot be converted to a number`)
}
