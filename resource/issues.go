package resource

import (
	"strings"

	"github.com/puppetlabs/go-issues/issue"
)

const (
	EVAL_APPLY_FUNCTION_SIZE_MISMATCH  = `EVAL_APPLY_FUNCTION_SIZE_MISMATCH`
	EVAL_APPLY_FUNCTION_INVALID_RETURN = `EVAL_APPLY_FUNCTION_INVALID_RETURN`
	EVAL_APPLY_FUNCTION_NIL_RETURN     = `EVAL_APPLY_FUNCTION_NIL_RETURN`
	EVAL_DECLARATION_MUST_HAVE_LAMBDA  = `EVAL_DECLARATION_MUST_HAVE_LAMBDA`
	EVAL_DUPLICATE_RESOURCE            = `EVAL_DUPLICATE_RESOURCE`
	EVAL_ILLEGAL_RESOURCE              = `EVAL_ILLEGAL_RESOURCE`
	EVAL_UNKNOWN_RESOURCE              = `EVAL_UNKNOWN_RESOURCE`
	EVAL_UNKNOWN_RESOURCE_TYPE         = `EVAL_UNKNOWN_RESOURCE_TYPE`
	EVAL_UNRESOLVED_GRAPH_NODE         = `EVAL_UNRESOLVED_GRAPH_NODE`
	EVAL_ILLEGAL_HANDLE_REPLACE        = `EVAL_ILLEGAL_HANDLE_REPLACE`
	EVAL_ILLEGAL_RESOURCE_REFERENCE    = `EVAL_ILLEGAL_RESOURCE_REFERENCE`
	EVAL_ILLEGAL_RESOURCE_OR_REFERENCE = `EVAL_ILLEGAL_RESOURCE_OR_REFERENCE`
)

func joinPath(path interface{}) string {
	return strings.Join(path.([]string), `/`)
}

func init() {
	issue.Hard(EVAL_APPLY_FUNCTION_SIZE_MISMATCH, `Slice returned by ApplyFunction has incorrect size. Expected %{expected}, actual %{actual}`)

	issue.Hard(EVAL_APPLY_FUNCTION_INVALID_RETURN, `Slice returned by ApplyFunction expects eval.PuppetObject. Got %<value>T`)

	issue.Hard(EVAL_APPLY_FUNCTION_NIL_RETURN, `Slice returned by ApplyFunction contains one or more nil values`)

	issue.Hard(EVAL_DECLARATION_MUST_HAVE_LAMBDA, `A %{declaration} declaration must be declared with a block`)

	issue.Hard(EVAL_DUPLICATE_RESOURCE, `Duplicate declaration: %{ref} is already declared %{previous_location}; cannot redeclare`)

	issue.Hard(EVAL_ILLEGAL_HANDLE_REPLACE, `Illegal replacement of Handle value. Expected value type '%{expected_type}', got '%{actual_type}'`)

	issue.Hard2(EVAL_ILLEGAL_RESOURCE, `%{value_type} cannot be used as a resource`, issue.HF{`value_type`: issue.A_anUc})

	issue.Hard(EVAL_ILLEGAL_RESOURCE_REFERENCE, `'%{str}' is not a valid resource reference. Must be in the form 'type_name[title]'`)

	issue.Hard(EVAL_UNKNOWN_RESOURCE, `Resource not found: %{type_name}['%{title}']`)

	issue.Hard(EVAL_UNKNOWN_RESOURCE_TYPE, `Resource type not found: %{res_type}`)

	issue.Hard(EVAL_UNRESOLVED_GRAPH_NODE, `Attempt to access value of not yet evaluated source '%{source}'`)

	issue.Hard2(EVAL_ILLEGAL_RESOURCE_OR_REFERENCE, `%{value_type} is not a valid resource or resource reference`,
		issue.HF{`value_type`: issue.A_anUc})
}
