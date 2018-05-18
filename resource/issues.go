package resource

import (
	"github.com/puppetlabs/go-issues/issue"
	"strings"
)

const (
	EVAL_DUPLICATE_RESOURCE            = `EVAL_DUPLICATE_RESOURCE`
	EVAL_ILLEGAL_RESOURCE              = `EVAL_ILLEGAL_RESOURCE`
	EVAL_UNKNOWN_RESOURCE              = `EVAL_UNKNOWN_RESOURCE`
	EVAL_UNKNOWN_RESOURCE_TYPE         = `EVAL_UNKNOWN_RESOURCE_TYPE`
	EVAL_UNRESOLVED_GRAPH_NODE         = `EVAL_UNRESOLVED_GRAPH_NODE`
	EVAL_ILLEGAL_HANDLE_REPLACE        = `EVAL_ILLEGAL_HANDLE_REPLACE`
	EVAL_ILLEGAL_RESOURCE_REFERENCE    = `EVAL_ILLEGAL_RESOURCE_REFERENCE`
	EVAL_ILLEGAL_RESOURCE_OR_REFERENCE = `EVAL_ILLEGAL_RESOURCE_OR_REFERENCE`
	EVAL_YAML_ILLEGAL_TYPE     = `EVAL_YAML_ELEMENT_MUST_BE_HASH`
	EVAL_YAML_RESOURCE_TYPE_MUST_BE_NAME = `EVAL_YAML_RESOURCE_TYPE_MUST_BE_NAME`
	EVAL_YAML_UNRECOGNIZED_TOP_CONSTRUCT = `EVAL_YAML_UNRECOGNIZED_TOP_CONSTRUCT`
)

func joinPath(path interface{}) string {
	return strings.Join(path.([]string), `/`)
}

func init() {
	issue.Hard(EVAL_DUPLICATE_RESOURCE, `Duplicate declaration: %{ref} is already declared %{previous_location}; cannot redeclare`)

	issue.Hard(EVAL_ILLEGAL_HANDLE_REPLACE, `Illegal replacement of Handle value. Expected value type '%{expected_type}', got '%{actual_type}'`)

	issue.Hard2(EVAL_ILLEGAL_RESOURCE, `%{value_type} cannot be used as a resource`, issue.HF{`value_type`: issue.A_anUc})

	issue.Hard(EVAL_ILLEGAL_RESOURCE_REFERENCE, `'%{str}' is not a valid resource reference. Must be in the form 'type_name[title]'`)

	issue.Hard(EVAL_UNKNOWN_RESOURCE, `Resource not found: %{type_name}['%{title}']`)

	issue.Hard(EVAL_UNKNOWN_RESOURCE_TYPE, `Resource type not found: %{res_type}`)

	issue.Hard(EVAL_UNRESOLVED_GRAPH_NODE, `Attempt to access value of not yet evaluated source '%{source}'`)

	issue.Hard2(EVAL_ILLEGAL_RESOURCE_OR_REFERENCE, `%{value_type} is not a valid resource or resource reference`,
		issue.HF{`value_type`: issue.A_anUc})

	issue.Hard2(EVAL_YAML_ILLEGAL_TYPE, `the value of key '%{key}' must be %{expected}. Got %{actual}. Path %{path}`,
		issue.HF{`path`: joinPath, `expected`: issue.A_an, `actual`: issue.A_an})

	issue.Hard2(EVAL_YAML_RESOURCE_TYPE_MUST_BE_NAME, `'%{key}' is not a valid resource name. Path %{path}`, issue.HF{`path`: joinPath})

	issue.Hard2(EVAL_YAML_UNRECOGNIZED_TOP_CONSTRUCT, `unrecognized key '%{key}'. Path %{path}`, issue.HF{`path`: joinPath})
}
