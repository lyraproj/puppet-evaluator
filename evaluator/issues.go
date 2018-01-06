package evaluator

import . "github.com/puppetlabs/go-parser/issue"

const (
	EVAL_ARGUMENTS_ERROR               = `EVAL_ARGUMENTS_ERROR`
	EVAL_FAILURE                       = `EVAL_FAILURE`
	EVAL_ILLEGAL_ARGUMENT              = `EVAL_ILLEGAL_ARGUMENT`
	EVAL_ILLEGAL_ARGUMENT_COUNT        = `EVAL_ILLEGAL_ARGUMENT_COUNT`
	EVAL_ILLEGAL_ARGUMENT_TYPE         = `EVAL_ILLEGAL_ARGUMENT_TYPE`
	EVAL_ILLEGAL_ASSIGNMENT            = `EVAL_ILLEGAL_ASSIGNMENT`
	EVAL_ILLEGAL_BREAK                 = `EVAL_ILLEGAL_BREAK`
	EVAL_ILLEGAL_NEXT                  = `EVAL_ILLEGAL_NEXT`
	EVAL_ILLEGAL_RETURN                = `EVAL_ILLEGAL_RETURN`
	EVAL_ILLEGAL_MULTI_ASSIGNMENT_SIZE = `EVAL_ILLEGAL_MULTI_ASSIGNMENT_SIZE`
	EVAL_ILLEGAL_REASSIGNMENT          = `EVAL_ILLEGAL_REASSIGNMENT`
	EVAL_LHS_MUST_BE_QREF              = `EVAL_LHS_MUST_BE_QREF`
	EVAL_MATCH_NOT_REGEXP              = `EVAL_MATCH_NOT_REGEXP`
	EVAL_MATCH_NOT_SEMVER_RANGE        = `EVAL_MATCH_NOT_SEMVER_RANGE`
	EVAL_MATCH_NOT_STRING              = `EVAL_MATCH_NOT_STRING`
	EVAL_MISSING_MULTI_ASSIGNMENT_KEY  = `EVAL_MISSING_MULTI_ASSIGNMENT_KEY`
	EVAL_NOT_NUMERIC                   = `EVAL_NOT_NUMERIC`
	EVAL_NOT_PARAMETERIZED_TYPE        = `EVAL_NOT_PARAMETERIZED_TYPE`
	EVAL_NOT_SEMVER                    = `EVAL_NOT_SEMVER`
	EVAL_OPERATOR_NOT_APPLICABLE       = `EVAL_OPERATOR_NOT_APPLICABLE`
	EVAL_OPERATOR_NOT_APPLICABLE_WHEN  = `EVAL_OPERATOR_NOT_APPLICABLE_WHEN`
	EVAL_UNHANDLED_EXPRESSION          = `EVAL_UNHANDLED_EXPRESSION`
	EVAL_UNKNOWN_FUNCTION              = `EVAL_UNKNOWN_FUNCTION`
	EVAL_UNKNOWN_VARIABLE              = `EVAL_UNKNOWN_VARIABLE`
)

func init() {
	HardIssue(EVAL_ARGUMENTS_ERROR, `Error when evaluating %s: %s`)
	HardIssue(EVAL_FAILURE, `Failure: %s`)
	HardIssue(EVAL_ILLEGAL_ARGUMENT, `Error when evaluating %s, argument %d: %s`)
	HardIssue(EVAL_ILLEGAL_ARGUMENT_COUNT, `Error when evaluating %s: Expected %s arguments, got %d`)
	HardIssue(EVAL_ILLEGAL_ARGUMENT_TYPE, `Error when evaluating %s: Expected argument %d to be %s, got %s`)
	HardIssue(EVAL_ILLEGAL_ASSIGNMENT, `Illegal attempt to assign to %s. Not an assignable reference`)
	HardIssue(EVAL_ILLEGAL_BREAK, `break() from context where this is illegal`)
	HardIssue(EVAL_ILLEGAL_NEXT, `next() from context where this is illegal`)
	HardIssue(EVAL_ILLEGAL_RETURN, `return() from context where this is illegal`)
	HardIssue(EVAL_ILLEGAL_MULTI_ASSIGNMENT_SIZE, `Mismatched number of assignable entries and values, expected %d, got %d`)
	HardIssue(EVAL_ILLEGAL_REASSIGNMENT, `Cannot reassign variable '$%s'`)
	HardIssue(EVAL_LHS_MUST_BE_QREF, `Expression to the left of [] expression must be a Type name`)
	HardIssue(EVAL_MATCH_NOT_REGEXP, `Can not convert right match operand to a regular expression. Caused by '%s'`)
	HardIssue(EVAL_MATCH_NOT_SEMVER_RANGE, `Can not convert right match operand to a semantic version range. Caused by '%s'`)
	HardIssue(EVAL_MATCH_NOT_STRING, `"Left match operand must result in a String value. Got %s`)
	HardIssue(EVAL_MISSING_MULTI_ASSIGNMENT_KEY, `No value for required key '%v' in assignment to variables from hash`)
	HardIssue(EVAL_NOT_NUMERIC, `The value '%s' cannot be converted to Numeric`)
	HardIssue(EVAL_NOT_PARAMETERIZED_TYPE, `%s is not a parameterized type`)
	HardIssue(EVAL_NOT_SEMVER, `The value cannot be converted to semantic version. Caused by '%s'`)
	HardIssue(EVAL_OPERATOR_NOT_APPLICABLE, `Operator '%s' is not applicable to %s`)
	HardIssue(EVAL_OPERATOR_NOT_APPLICABLE_WHEN, `Operator '%s' is not applicable to %s when right side is %s`)
	HardIssue(EVAL_UNHANDLED_EXPRESSION, `Evaluator cannot handle an expression of type %T`)
	HardIssue(EVAL_UNKNOWN_FUNCTION, `Unknown function: '%v'`)
	HardIssue(EVAL_UNKNOWN_VARIABLE, `Unknown variable: '$%v'`)
}
