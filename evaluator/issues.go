package evaluator

import (
	. "github.com/puppetlabs/go-parser/issue"
	. "github.com/puppetlabs/go-parser/parser"
)

const (
	EVAL_ARGUMENTS_ERROR                = `EVAL_ARGUMENTS_ERROR`
	EVAL_ATTRIBUTE_HAS_NO_VALUE         = `EVAL_ATTRIBUTE_HAS_NO_VALUE`
	EVAL_BOTH_CONSTANT_AND_ATTRIBUTE    = `EVAL_BOTH_CONSTANT_AND_ATTRIBUTE`
	EVAL_CONSTANT_REQUIRES_VALUE        = `EVAL_CONSTANT_REQUIRES_VALUE`
	EVAL_CONSTANT_WITH_FINAL            = `EVAL_CONSTANT_WITH_FINAL`
	EVAL_EQUALITY_ATTRIBUTE_NOT_FOUND   = `EVAL_EQUALITY_ATTRIBUTE_NOT_FOUND`
	EVAL_EQUALITY_NOT_ATTRIBUTE         = `EVAL_EQUALITY_NOT_ATTRIBUTE`
	EVAL_EQUALITY_ON_CONSTANT           = `EVAL_EQUALITY_ON_CONSTANT`
	EVAL_EQUALITY_REDEFINED             = `EVAL_EQUALITY_REDEFINED`
	EVAL_FAILURE                        = `EVAL_FAILURE`
	EVAL_GO_RUNTIME_TYPE_WITHOUT_GO_TYPE = `EVAL_GO_RUNTIME_TYPE_WITHOUT_GO_TYPE`
	EVAL_ILLEGAL_ARGUMENT               = `EVAL_ILLEGAL_ARGUMENT`
	EVAL_ILLEGAL_ARGUMENT_COUNT         = `EVAL_ILLEGAL_ARGUMENT_COUNT`
	EVAL_ILLEGAL_ARGUMENT_TYPE          = `EVAL_ILLEGAL_ARGUMENT_TYPE`
	EVAL_ILLEGAL_ASSIGNMENT             = `EVAL_ILLEGAL_ASSIGNMENT`
	EVAL_ILLEGAL_BREAK                  = `EVAL_ILLEGAL_BREAK`
	EVAL_ILLEGAL_KIND_VALUE_COMBINATION = `EVAL_ILLEGAL_KIND_VALUE_COMBINATION`
	EVAL_ILLEGAL_NEXT                   = `EVAL_ILLEGAL_NEXT`
	EVAL_ILLEGAL_OBJECT_INHERITANCE     = `EVAL_ILLEGAL_OBJECT_INHERITANCE`
	EVAL_ILLEGAL_RETURN                 = `EVAL_ILLEGAL_RETURN`
	EVAL_ILLEGAL_MULTI_ASSIGNMENT_SIZE  = `EVAL_ILLEGAL_MULTI_ASSIGNMENT_SIZE`
	EVAL_ILLEGAL_REASSIGNMENT           = `EVAL_ILLEGAL_REASSIGNMENT`
	EVAL_INVALID_REGEXP                 = `EVAL_INVALID_REGEXP`
	EVAL_LHS_MUST_BE_QREF               = `EVAL_LHS_MUST_BE_QREF`
	EVAL_MATCH_NOT_REGEXP               = `EVAL_MATCH_NOT_REGEXP`
	EVAL_MATCH_NOT_SEMVER_RANGE         = `EVAL_MATCH_NOT_SEMVER_RANGE`
	EVAL_MATCH_NOT_STRING               = `EVAL_MATCH_NOT_STRING`
	EVAL_MISSING_MULTI_ASSIGNMENT_KEY   = `EVAL_MISSING_MULTI_ASSIGNMENT_KEY`
	EVAL_NO_DEFINITION                  = `EVAL_NO_DEFINITION`
	EVAL_NOT_EXPECTED_TYPESET           = `EVAL_NOT_EXPECTED_TYPESET`
	EVAL_NOT_ONLY_DEFINITION            = `EVAL_NOT_ONLY_DEFINITION`
	EVAL_NOT_NUMERIC                    = `EVAL_NOT_NUMERIC`
	EVAL_NOT_PARAMETERIZED_TYPE         = `EVAL_NOT_PARAMETERIZED_TYPE`
	EVAL_NOT_SEMVER                     = `EVAL_NOT_SEMVER`
	EVAL_OBJECT_INHERITS_SELF           = `EVAL_OBJECT_INHERITS_SELF`
	EVAL_OPERATOR_NOT_APPLICABLE        = `EVAL_OPERATOR_NOT_APPLICABLE`
	EVAL_OPERATOR_NOT_APPLICABLE_WHEN   = `EVAL_OPERATOR_NOT_APPLICABLE_WHEN`
	EVAL_OVERRIDE_MEMBER_MISMATCH       = `EVAL_OVERRIDE_MEMBER_MISMATCH`
	EVAL_OVERRIDE_TYPE_MISMATCH         = `EVAL_OVERRIDE_TYPE_MISMATCH`
	EVAL_OVERRIDDEN_NOT_FOUND           = `EVAL_OVERRIDDEN_NOT_FOUND`
	EVAL_OVERRIDE_OF_FINAL              = `EVAL_OVERRIDE_OF_FINAL`
	EVAL_OVERRIDE_IS_MISSING            = `EVAL_OVERRIDE_IS_MISSING`
	EVAL_TYPE_MISMATCH                  = `EVAL_TYPE_MISMATCH`
	EVAL_UNABLE_TO_READ_FILE            = `EVAL_UNABLE_TO_READ_FILE`
	EVAL_UNHANDLED_EXPRESSION           = `EVAL_UNHANDLED_EXPRESSION`
	EVAL_UNKNOWN_FUNCTION               = `EVAL_UNKNOWN_FUNCTION`
	EVAL_UNKNOWN_VARIABLE               = `EVAL_UNKNOWN_VARIABLE`
	EVAL_WRONG_DEFINITION               = `EVAL_WRONG_DEFINITION`
)

func init() {
	HardIssue2(EVAL_ARGUMENTS_ERROR, `Error when evaluating %{expression}: %{message}`, HF{`expression`: A_an})

	HardIssue(EVAL_BOTH_CONSTANT_AND_ATTRIBUTE, `attribute %{label}[%{key}] is defined as both a constant and an attribute`)

	HardIssue(EVAL_ATTRIBUTE_HAS_NO_VALUE, `%{label} has no value`)

	HardIssue(EVAL_CONSTANT_REQUIRES_VALUE, `%{label} of kind 'constant' requires a value`)

	// TRANSLATOR 'final => false' is puppet syntax and should not be translated
	HardIssue(EVAL_CONSTANT_WITH_FINAL, `%{label} of kind 'constant' cannot be combined with final => false`)

	HardIssue(EVAL_EQUALITY_ATTRIBUTE_NOT_FOUND, `%{label} equality is referencing non existent attribute '%{attribute}'`)

	HardIssue(EVAL_EQUALITY_NOT_ATTRIBUTE, `{label} equality is referencing %{attribute}. Only attribute references are allowed`)

	HardIssue(EVAL_EQUALITY_ON_CONSTANT, `%{label} equality is referencing constant %{attribute}.`)

	HardIssue(EVAL_EQUALITY_REDEFINED, `%{label} equality is referencing %{attribute} which is included in equality of %{including_parent}`)

	HardIssue(EVAL_FAILURE, `%{message}`)

	HardIssue(EVAL_GO_RUNTIME_TYPE_WITHOUT_GO_TYPE, `Attempt to create a Runtime['go', '%{name}'] without providing a Go type`)

	HardIssue2(EVAL_ILLEGAL_ARGUMENT,
		`Error when evaluating %{expression}, argument %{number}:  %{message}`, HF{`expression`: A_an})

	HardIssue2(EVAL_ILLEGAL_ARGUMENT_COUNT,
		`Error when evaluating %{expression}: Expected %{expected} arguments, got %{actual}`,
		HF{`expression`: A_an})

	HardIssue2(EVAL_ILLEGAL_ARGUMENT_TYPE,
		`Error when evaluating %{expression}: Expected argument %{number} to be %{expected}, got %{actual}`,
		HF{`expression`: A_an})

	HardIssue2(EVAL_ILLEGAL_ASSIGNMENT, `Illegal attempt to assign to %{value}. Not an assignable reference`,
		HF{`value`: A_an})

	HardIssue(EVAL_ILLEGAL_BREAK, `break() from context where this is illegal`)

	HardIssue(EVAL_ILLEGAL_KIND_VALUE_COMBINATION, `%{label} of kind '%{kind}' cannot be combined with an attribute value`)

	HardIssue(EVAL_ILLEGAL_NEXT, `next() from context where this is illegal`)

	HardIssue(EVAL_ILLEGAL_OBJECT_INHERITANCE, `An Object can only inherit another Object or alias thereof. The %{label} inherits from a %{type}.`)

	HardIssue(EVAL_ILLEGAL_RETURN, `return() from context where this is illegal`)

	HardIssue(EVAL_ILLEGAL_MULTI_ASSIGNMENT_SIZE, `Mismatched number of assignable entries and values, expected %{expected}, got %{actual}`)

	HardIssue(EVAL_ILLEGAL_REASSIGNMENT, `Cannot reassign variable '$%{var}'`)

	HardIssue(EVAL_INVALID_REGEXP, `Cannot compile regular expression '${pattern}': %{detail}`)

	HardIssue(EVAL_LHS_MUST_BE_QREF, `Expression to the left of [] expression must be a Type name`)

	HardIssue(EVAL_MATCH_NOT_REGEXP, `Can not convert right match operand to a regular expression. Caused by '%{detail}'`)

	HardIssue(EVAL_MATCH_NOT_SEMVER_RANGE, `Can not convert right match operand to a semantic version range. Caused by '%s'`)

	HardIssue2(EVAL_MATCH_NOT_STRING, `"Left match operand must result in a String value. Got %{left}`, HF{`left`: A_an})

	HardIssue(EVAL_MISSING_MULTI_ASSIGNMENT_KEY, `No value for required key '%{name}' in assignment to variables from hash`)

	HardIssue(EVAL_OBJECT_INHERITS_SELF, `The Object type '%{label}' inherits from itself`)

	HardIssue(EVAL_NO_DEFINITION, `The code loaded from %{source} does not define the %{type} '%{name}`)

	HardIssue(EVAL_NOT_EXPECTED_TYPESET, `The code loaded from %{source} does not define the TypeSet %{name}'`)

	HardIssue(EVAL_NOT_NUMERIC, `The value '%{value}' cannot be converted to Numeric`)

	HardIssue(EVAL_NOT_ONLY_DEFINITION, `The code loaded from %{source} must contain only the %{type} '%{name}`)

	HardIssue2(EVAL_NOT_PARAMETERIZED_TYPE, `%{type} is not a parameterized type`,
		HF{`type`: A_anUc})

	HardIssue(EVAL_NOT_SEMVER, `The value cannot be converted to semantic version. Caused by '%{detail}'`)

	HardIssue2(EVAL_OPERATOR_NOT_APPLICABLE, `Operator '%{operator}' is not applicable to %{left}`,
		HF{`left`: A_an})

	HardIssue2(EVAL_OPERATOR_NOT_APPLICABLE_WHEN,
		`Operator '%{operator}' is not applicable to %{left} when right side is %{right}`,
		HF{`left`: A_an, `right`: A_an})

	HardIssue(EVAL_OVERRIDE_MEMBER_MISMATCH, `%{member} attempts to override %{label}`)

	HardIssue(EVAL_OVERRIDDEN_NOT_FOUND, `expected %{label} to override an inherited %{feature_type}, but no such %{feature_type} was found`)

	// TRANSLATOR 'override => true' is a puppet syntax and should not be translated
	HardIssue(EVAL_OVERRIDE_IS_MISSING, `%{member} attempts to override %{label} without having override => true`)

	HardIssue(EVAL_OVERRIDE_OF_FINAL, `%{member} attempts to override final %{label}`)

	HardIssue(EVAL_TYPE_MISMATCH, `Type mismatch: %{detail}`)

	HardIssue(EVAL_UNABLE_TO_READ_FILE, `Unable to read file '%{path}': %{detail}`)

	HardIssue(EVAL_UNHANDLED_EXPRESSION, `Evaluator cannot handle an expression of type %<expression>T`)

	HardIssue(EVAL_UNKNOWN_FUNCTION, `Unknown function: '%{name}'`)

	HardIssue(EVAL_UNKNOWN_VARIABLE, `Unknown variable: '$%{name}'`)

	HardIssue(EVAL_WRONG_DEFINITION, `The code loaded from %{source} produced %{type} with the wrong name, expected %{expected}, actual %{actual}`)
}
