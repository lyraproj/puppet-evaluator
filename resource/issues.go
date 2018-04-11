package resource

import (
	"github.com/puppetlabs/go-issues/issue"
)

const (
	EVAL_DUPLICATE_RESOURCE = `EVAL_DUPLICATE_RESOURCE`
	EVAL_ILLEGAL_RESOURCE = `EVAL_ILLEGAL_RESOURCE`
	EVAL_UNKNOWN_RESOURCE = `EVAL_UNKNOWN_RESOURCE`
	EVAL_UNKNOWN_RESOURCE_TYPE = `EVAL_UNKNOWN_RESOURCE_TYPE`
	EVAL_UNRESOLVED_GRAPH_NODE = `EVAL_UNRESOLVED_GRAPH_NODE`
	EVAL_ILLEGAL_HANDLE_REPLACE = `EVAL_ILLEGAL_HANDLE_REPLACE`
	EVAL_ILLEGAL_RESOURCE_REFERENCE = `EVAL_ILLEGAL_RESOURCE_REFERENCE`
	EVAL_ILLEGAL_RESOURCE_OR_REFERENCE = `EVAL_ILLEGAL_RESOURCE_OR_REFERENCE`
)

func init() {
	issue.Hard(EVAL_DUPLICATE_RESOURCE, `Duplicate declaration: %{ref} is already declared %{previous_location}; cannot redeclare`)

	issue.Hard(EVAL_ILLEGAL_HANDLE_REPLACE, `Illegal replacement of Handle value. Expected value type '%{expected_type}', got '%{actual_type}'`)

	issue.Hard2(EVAL_ILLEGAL_RESOURCE, `%{value_type} cannot be used as a resource`, issue.HF{ `value_type`: issue.A_anUc})

	issue.Hard(EVAL_ILLEGAL_RESOURCE_REFERENCE, `'%{str}' is not a valid resource reference. Must be in the form 'type_name[title]'`)

	issue.Hard(EVAL_UNKNOWN_RESOURCE, `Resource not found: %{type_name}['%{title}']`)

	issue.Hard(EVAL_UNKNOWN_RESOURCE_TYPE, `Resource type not found: %{res_type}`)

	issue.Hard(EVAL_UNRESOLVED_GRAPH_NODE, `Attempt to access value of not yet evaluated source '%{source}'`)

	issue.Hard2(EVAL_ILLEGAL_RESOURCE_OR_REFERENCE, `%{value_type} is not a valid resource or resource reference`,
		issue.HF{ `value_type`: issue.A_anUc})
}
