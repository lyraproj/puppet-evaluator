package pdsl

import "github.com/lyraproj/issue/issue"

const (
	IllegalArgument             = `EVAL_ILLEGAL_ARGUMENT`
	IllegalArgumentCount        = `EVAL_ILLEGAL_ARGUMENT_COUNT`
	IllegalArgumentType         = `EVAL_ILLEGAL_ARGUMENT_TYPE`
	IllegalAssignment           = `EVAL_ILLEGAL_ASSIGNMENT`
	IllegalBreak                = `EVAL_ILLEGAL_BREAK`
	IllegalNext                 = `EVAL_ILLEGAL_NEXT`
	IllegalReturn               = `EVAL_ILLEGAL_RETURN`
	IllegalMultiAssignmentSize  = `EVAL_ILLEGAL_MULTI_ASSIGNMENT_SIZE`
	IllegalWhenStaticExpression = `EVAL_ILLEGAL_WHEN_STATIC_EXPRESSION`
	IllegalReassignment         = `EVAL_ILLEGAL_REASSIGNMENT`
	MissingMultiAssignmentKey   = `EVAL_MISSING_MULTI_ASSIGNMENT_KEY`
	MissingRegexpInType         = `EVAL_MISSING_REGEXP_IN_TYPE`
	NotCollectionAt             = `EVAL_NOT_COLLECTION_AT`
	NotOnlyDefinition           = `EVAL_NOT_ONLY_DEFINITION`
	NotNumeric                  = `EVAL_NOT_NUMERIC`
	OperatorNotApplicable       = `EVAL_OPERATOR_NOT_APPLICABLE`
	OperatorNotApplicableWhen   = `EVAL_OPERATOR_NOT_APPLICABLE_WHEN`
	TaskBadJson                 = `EVAL_TASK_BAD_JSON`
	TaskInitializerNotFound     = `EVAL_TASK_INITIALIZER_NOT_FOUND`
	TaskNoExecutableFound       = `EVAL_TASK_NO_EXECUTABLE_FOUND`
	TaskNotJsonObject           = `EVAL_TASK_NOT_JSON_OBJECT`
	TaskTooManyFiles            = `EVAL_TASK_TOO_MANY_FILES`
	UnhandledExpression         = `EVAL_UNHANDLED_EXPRESSION`
	UnknownPlan                 = `EVAL_UNKNOWN_PLAN`
	UnknownTask                 = `EVAL_UNKNOWN_TASK`
	UnknownVariable             = `EVAL_UNKNOWN_VARIABLE`
)

func init() {
	issue.Hard2(IllegalArgument,
		`Error when evaluating %{expression}, argument %{number}:  %{message}`, issue.HF{`expression`: issue.AnOrA})

	issue.Hard2(IllegalArgumentCount,
		`Error when evaluating %{expression}: Expected %{expected} arguments, got %{actual}`,
		issue.HF{`expression`: issue.AnOrA})

	issue.Hard2(IllegalArgumentType,
		`Error when evaluating %{expression}: Expected argument %{number} to be %{expected}, got %{actual}`,
		issue.HF{`expression`: issue.AnOrA})

	issue.Hard2(IllegalAssignment, `Illegal attempt to assign to %{value}. Not an assignable reference`,
		issue.HF{`value`: issue.AnOrA})

	issue.Hard(IllegalBreak, `break() from context where this is illegal`)

	issue.Hard2(IllegalWhenStaticExpression, `%{expression} is illegal within a type declaration`, issue.HF{`expression`: issue.UcAnOrA})

	issue.Hard(IllegalNext, `next() from context where this is illegal`)

	issue.Hard(IllegalReturn, `return() from context where this is illegal`)

	issue.Hard(IllegalMultiAssignmentSize, `Mismatched number of assignable entries and values, expected %{expected}, got %{actual}`)

	issue.Hard(IllegalReassignment, `Cannot reassign variable '$%{var}'`)

	issue.Hard(MissingMultiAssignmentKey, `No value for required key '%{name}' in assignment to variables from hash`)

	issue.Hard(MissingRegexpInType, `Given Regexp Type has no regular expression`)

	issue.Hard(NotCollectionAt, `The given data does not contain a Collection at %{walked_path}, got '%{klass}'`)

	issue.Hard(NotNumeric, `The value '%{value}' cannot be converted to Numeric`)

	issue.Hard(NotOnlyDefinition, `The code loaded from %{source} must contain only the %{type} '%{name}`)

	issue.Hard2(OperatorNotApplicable, `Operator '%{operator}' is not applicable to %{left}`,
		issue.HF{`left`: issue.AnOrA})

	issue.Hard2(OperatorNotApplicableWhen,
		`Operator '%{operator}' is not applicable to %{left} when right side is %{right}`,
		issue.HF{`left`: issue.AnOrA, `right`: issue.AnOrA})

	issue.Hard(TaskBadJson, `Unable to parse task metadata from '%{path}': %{detail}`)

	issue.Hard(TaskInitializerNotFound, `Unable to load the initializer for the Task data`)

	issue.Hard(TaskNoExecutableFound, `No source besides task metadata was found in directory %{directory} for task %{name}`)

	issue.Hard(TaskNotJsonObject, `The content of '%{path}' does not represent a JSON Object`)

	issue.Hard(TaskTooManyFiles, `Only one file can exists besides the .json file for task %{name} in directory %{directory}`)

	issue.Hard(UnhandledExpression, `Evaluator cannot handle an expression of type %<expression>T`)

	issue.Hard(UnknownPlan, `Unknown plan: '%{name}'`)

	issue.Hard(UnknownTask, `Task not found: '%{name}'`)

	issue.Hard(UnknownVariable, `Unknown variable: '$%{name}'`)
}
