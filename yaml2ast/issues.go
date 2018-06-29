package yaml2ast

import (
	"strings"

	"github.com/puppetlabs/go-issues/issue"
)

const (
	EVAL_YAML_DUPLICATE_KEY              = `EVAL_YAML_DUPLICATE_KEY`
	EVAL_YAML_ILLEGAL_TYPE               = `EVAL_YAML_ELEMENT_MUST_BE_HASH`
	EVAL_ILLEGAL_VARIABLE_NAME           = `EVAL_ILLEGAL_VARIABLE_NAME`
	EVAL_YAML_RESOURCE_TYPE_MUST_BE_NAME = `EVAL_YAML_RESOURCE_TYPE_MUST_BE_NAME`
	EVAL_YAML_UNRECOGNIZED_TOP_CONSTRUCT = `EVAL_YAML_UNRECOGNIZED_TOP_CONSTRUCT`
)

func joinPath(path interface{}) string {
	return strings.Join(path.([]string), `/`)
}

func init() {
	issue.Hard(EVAL_YAML_DUPLICATE_KEY, `the key '%{key}' is defined more than once. Path %{path}`)

	issue.Hard2(EVAL_YAML_ILLEGAL_TYPE, `the value must be %{expected}. Got %{actual}. Path %{path}`,
		issue.HF{`path`: joinPath, `expected`: issue.A_an, `actual`: issue.A_an})

	issue.Hard(EVAL_ILLEGAL_VARIABLE_NAME, `'%{name}' is not a legal variable name`)

	issue.Hard2(EVAL_YAML_RESOURCE_TYPE_MUST_BE_NAME, `'%{key}' is not a valid resource name. Path %{path}`, issue.HF{`path`: joinPath})

	issue.Hard2(EVAL_YAML_UNRECOGNIZED_TOP_CONSTRUCT, `unrecognized key '%{key}'. Path %{path}`, issue.HF{`path`: joinPath})
}
