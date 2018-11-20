package eval

import (
	"github.com/puppetlabs/go-issues/issue"
)

const (
	EVAL_ARGUMENTS_ERROR                           = `EVAL_ARGUMENTS_ERROR`
	EVAL_ATTEMPT_TO_REDEFINE                       = `EVAL_ATTEMPT_TO_REDEFINE`
	EVAL_ATTEMPT_TO_SET_UNSETTABLE                 = `EVAL_ATTEMPT_TO_SET_UNSETTABLE`
	EVAL_ATTEMPT_TO_SET_WRONG_KIND                 = `EVAL_ATTEMPT_TO_SET_WRONG_KIND`
	EVAL_ATTRIBUTE_HAS_NO_VALUE                    = `EVAL_ATTRIBUTE_HAS_NO_VALUE`
	EVAL_ATTRIBUTE_NOT_FOUND                       = `EVAL_ATTRIBUTE_NOT_FOUND`
	EVAL_BAD_JSON_PATH                             = `EVAL_BAD_JSON_PATH`
	EVAL_BAD_TYPE_STRING                           = `EVAL_BAD_TYPE_STRING`
	EVAL_BOTH_CONSTANT_AND_ATTRIBUTE               = `EVAL_BOTH_CONSTANT_AND_ATTRIBUTE`
	EVAL_CONSTANT_REQUIRES_VALUE                   = `EVAL_CONSTANT_REQUIRES_VALUE`
	EVAL_CONSTANT_WITH_FINAL                       = `EVAL_CONSTANT_WITH_FINAL`
	EVAL_CTOR_NOT_FOUND                            = `EVAL_CTOR_NOT_FOUND`
	EVAL_DUPLICATE_KEY                             = `EVAL_DUPLICATE_KEY`
	EVAL_EMPTY_TYPE_PARAMETER_LIST                 = `EVAL_EMPTY_TYPE_PARAMETER_LIST`
	EVAL_EQUALITY_ATTRIBUTE_NOT_FOUND              = `EVAL_EQUALITY_ATTRIBUTE_NOT_FOUND`
	EVAL_EQUALITY_NOT_ATTRIBUTE                    = `EVAL_EQUALITY_NOT_ATTRIBUTE`
	EVAL_EQUALITY_ON_CONSTANT                      = `EVAL_EQUALITY_ON_CONSTANT`
	EVAL_EQUALITY_REDEFINED                        = `EVAL_EQUALITY_REDEFINED`
	EVAL_FAILURE                                   = `EVAL_FAILURE`
	EVAL_FILE_NOT_FOUND                            = `EVAL_FILE_NOT_FOUND`
	EVAL_FILE_READ_DENIED                          = `EVAL_FILE_READ_DENIED`
	EVAL_GO_FUNCTION_ERROR                         = `EVAL_GO_FUNCTION_ERROR`
	EVAL_GO_RUNTIME_TYPE_WITHOUT_GO_TYPE           = `EVAL_GO_RUNTIME_TYPE_WITHOUT_GO_TYPE`
	EVAL_ILLEGAL_ARGUMENT                          = `EVAL_ILLEGAL_ARGUMENT`
	EVAL_ILLEGAL_ARGUMENT_COUNT                    = `EVAL_ILLEGAL_ARGUMENT_COUNT`
	EVAL_ILLEGAL_ARGUMENT_TYPE                     = `EVAL_ILLEGAL_ARGUMENT_TYPE`
	EVAL_ILLEGAL_ASSIGNMENT                        = `EVAL_ILLEGAL_ASSIGNMENT`
	EVAL_ILLEGAL_BREAK                             = `EVAL_ILLEGAL_BREAK`
	EVAL_ILLEGAL_KIND_VALUE_COMBINATION            = `EVAL_ILLEGAL_KIND_VALUE_COMBINATION`
	EVAL_ILLEGAL_NEXT                              = `EVAL_ILLEGAL_NEXT`
	EVAL_ILLEGAL_OBJECT_INHERITANCE                = `EVAL_ILLEGAL_OBJECT_INHERITANCE`
	EVAL_ILLEGAL_RETURN                            = `EVAL_ILLEGAL_RETURN`
	EVAL_ILLEGAL_MULTI_ASSIGNMENT_SIZE             = `EVAL_ILLEGAL_MULTI_ASSIGNMENT_SIZE`
	EVAL_ILLEGAL_WHEN_STATIC_EXPRESSION            = `EVAL_ILLEGAL_WHEN_STATIC_EXPRESSION`
	EVAL_IMPL_ALREDY_REGISTERED                    = `EVAL_IMPL_ALREDY_REGISTERED`
	EVAL_ILLEGAL_REASSIGNMENT                      = `EVAL_ILLEGAL_REASSIGNMENT`
	EVAL_INSTANCE_DOES_NOT_RESPOND                 = `EVAL_INSTANCE_DOES_NOT_RESPOND`
	EVAL_INVALID_CHARACTERS_IN_NAME                = `EVAL_INVALID_CHARACTERS_IN_NAME`
	EVAL_INVALID_REGEXP                            = `EVAL_INVALID_REGEXP`
	EVAL_INVALID_SOURCE_FOR_GET                    = `EVAL_INVALID_SOURCE_FOR_GET`
	EVAL_INVALID_SOURCE_FOR_SET                    = `EVAL_INVALID_SOURCE_FOR_SET`
	EVAL_INVALID_STRING_FORMAT_SPEC                = `EVAL_INVALID_STRING_FORMAT_SPEC`
	EVAL_INVALID_STRING_FORMAT_DELIMITER           = `EVAL_INVALID_STRING_FORMAT_DELIMITER`
	EVAL_INVALID_STRING_FORMAT_REPEATED_FLAG       = `EVAL_INVALID_STRING_FORMAT_REPEATED_FLAG`
	EVAL_INVALID_TIMEZONE                          = `EVAL_INVALID_TIMEZONE`
	EVAL_INVALID_TYPEDNAME_MAPKEY                  = `EVAL_INVALID_TYPEDNAME_MAPKEY`
	EVAL_INVALID_URI                               = `EVAL_INVALID_URI`
	EVAL_INVALID_VERSION                           = `EVAL_INVALID_VERSION`
	EVAL_INVALID_VERSION_RANGE                     = `EVAL_INVALID_VERSION_RANGE`
	EVAL_IS_DIRECTORY                              = `EVAL_IS_DIRECTORY`
	EVAL_MATCH_NOT_REGEXP                          = `EVAL_MATCH_NOT_REGEXP`
	EVAL_MATCH_NOT_STRING                          = `EVAL_MATCH_NOT_STRING`
	EVAL_MEMBER_NAME_CONFLICT                      = `EVAL_MEMBER_NAME_CONFLICT`
	EVAL_MISSING_MULTI_ASSIGNMENT_KEY              = `EVAL_MISSING_MULTI_ASSIGNMENT_KEY`
	EVAL_MISSING_REGEXP_IN_TYPE                    = `EVAL_MISSING_REGEXP_IN_TYPE`
	EVAL_MISSING_REQUIRED_ATTRIBUTE                = `EVAL_MISSING_REQUIRED_ATTRIBUTE`
	EVAL_MISSING_TYPE_PARAMETER                    = `EVAL_MISSING_TYPE_PARAMETER`
	EVAL_NO_ATTRIBUTE_READER                       = `EVAL_NO_ATTRIBUTE_READER`
	EVAL_NO_CURRENT_CONTEXT                        = `EVAL_NO_CURRENT_CONTEXT`
	EVAL_NO_DEFINITION                             = `EVAL_NO_DEFINITION`
	EVAL_NOT_COLLECTION_AT                         = `EVAL_NOT_COLLECTION_AT`
	EVAL_NOT_EXPECTED_TYPESET                      = `EVAL_NOT_EXPECTED_TYPESET`
	EVAL_NOT_INTEGER                               = `EVAL_NOT_INTEGER`
	EVAL_NOT_ONLY_DEFINITION                       = `EVAL_NOT_ONLY_DEFINITION`
	EVAL_NOT_NUMERIC                               = `EVAL_NOT_NUMERIC`
	EVAL_NOT_PARAMETERIZED_TYPE                    = `EVAL_NOT_PARAMETERIZED_TYPE`
	EVAL_NOT_SEMVER                                = `EVAL_NOT_SEMVER`
	EVAL_NOT_SUPPORTED_BY_GO_TIME_LAYOUT           = `EVAL_NOT_SUPPORTED_BY_GO_TIME_LAYOUT`
	EVAL_OBJECT_INHERITS_SELF                      = `EVAL_OBJECT_INHERITS_SELF`
	EVAL_OPERATOR_NOT_APPLICABLE                   = `EVAL_OPERATOR_NOT_APPLICABLE`
	EVAL_OPERATOR_NOT_APPLICABLE_WHEN              = `EVAL_OPERATOR_NOT_APPLICABLE_WHEN`
	EVAL_OVERRIDE_MEMBER_MISMATCH                  = `EVAL_OVERRIDE_MEMBER_MISMATCH`
	EVAL_OVERRIDE_TYPE_MISMATCH                    = `EVAL_OVERRIDE_TYPE_MISMATCH`
	EVAL_OVERRIDDEN_NOT_FOUND                      = `EVAL_OVERRIDDEN_NOT_FOUND`
	EVAL_OVERRIDE_OF_FINAL                         = `EVAL_OVERRIDE_OF_FINAL`
	EVAL_OVERRIDE_IS_MISSING                       = `EVAL_OVERRIDE_IS_MISSING`
	EVAL_PARSE_ERROR                               = `EVAL_PARSE_ERROR`
	EVAL_SERIALIZATION_ATTRIBUTE_NOT_FOUND         = `EVAL_SERIALIZATION_ATTRIBUTE_NOT_FOUND`
	EVAL_SERIALIZATION_NOT_ATTRIBUTE               = `EVAL_SERIALIZATION_NOT_ATTRIBUTE`
	EVAL_SERIALIZATION_BAD_KIND                    = `EVAL_SERIALIZATION_BAD_KIND`
	EVAL_SERIALIZATION_DEFAULT_CONVERTED_TO_STRING = `EVAL_SERIALIZATION_DEFAULT_CONVERTED_TO_STRING`
	EVAL_SERIALIZATION_ENDLESS_RECURSION           = `EVAL_SERIALIZATION_ENDLESS_RECURSION`
	EVAL_SERIALIZATION_REQUIRED_AFTER_OPTIONAL     = `EVAL_SERIALIZATION_REQUIRED_AFTER_OPTIONAL`
	EVAL_SERIALIZATION_UNKNOWN_CONVERTED_TO_STRING = `EVAL_SERIALIZATION_UNKNOWN_CONVERTED_TO_STRING`
	EVAL_TASK_BAD_JSON                             = `EVAL_TASK_BAD_JSON`
	EVAL_TASK_INITIALIZER_NOT_FOUND                = `EVAL_TASK_INITIALIZER_NOT_FOUND`
	EVAL_TASK_NO_EXECUTABLE_FOUND                  = `EVAL_TASK_NO_EXECUTABLE_FOUND`
	EVAL_TASK_NOT_JSON_OBJECT                      = `EVAL_TASK_NOT_JSON_OBJECT`
	EVAL_TASK_TOO_MANY_FILES                       = `EVAL_TASK_TOO_MANY_FILES`
	EVAL_TIMESPAN_BAD_FSPEC                        = `EVAL_TIMESPAN_BAD_FSPEC`
	EVAL_TIMESPAN_CANNOT_BE_PARSED                 = `EVAL_TIMESPAN_CANNOT_BE_PARSED`
	EVAL_TIMESPAN_FSPEC_NOT_HIGHER                 = `EVAL_TIMESPAN_FSPEC_NOT_HIGHER`
	EVAL_TIMESTAMP_CANNOT_BE_PARSED                = `EVAL_TIMESTAMP_CANNOT_BE_PARSED`
	EVAL_TIMESTAMP_TZ_AMBIGUITY                    = `EVAL_TIMESTAMP_TZ_AMBIGUITY`
	EVAL_TYPE_MISMATCH                             = `EVAL_TYPE_MISMATCH`
	EVAL_TYPESET_ALIAS_COLLIDES                    = `EVAL_TYPESET_ALIAS_COLLIDES`
	EVAL_TYPESET_MISSING_NAME_AUTHORITY            = `EVAL_TYPESET_MISSING_NAME_AUTHORITY`
	EVAL_TYPESET_REFERENCE_BAD_TYPE                = `EVAL_TYPESET_REFERENCE_BAD_TYPE`
	EVAL_TYPESET_REFERENCE_DUPLICATE               = `EVAL_TYPESET_REFERENCE_DUPLICATE`
	EVAL_TYPESET_REFERENCE_MISMATCH                = `EVAL_TYPESET_REFERENCE_MISMATCH`
	EVAL_TYPESET_REFERENCE_OVERLAP                 = `EVAL_TYPESET_REFERENCE_OVERLAP`
	EVAL_TYPESET_REFERENCE_UNRESOLVED              = `EVAL_TYPESET_REFERENCE_UNRESOLVED`
	EVAL_UNABLE_TO_DESERIALIZE_TYPE                = `EVAL_UNABLE_TO_DESERIALIZE_TYPE`
	EVAL_UNABLE_TO_DESERIALIZE_VALUE               = `EVAL_UNABLE_TO_DESERIALIZE_VALUE`
	EVAL_UNABLE_TO_READ_FILE                       = `EVAL_UNABLE_TO_READ_FILE`
	EVAL_UNHANDLED_PCORE_VERSION                   = `EVAL_UNHANDLED_PCORE_VERSION`
	EVAL_UNHANDLED_EXPRESSION                      = `EVAL_UNHANDLED_EXPRESSION`
	EVAL_UNKNOWN_FUNCTION                          = `EVAL_UNKNOWN_FUNCTION`
	EVAL_UNKNOWN_PLAN                              = `EVAL_UNKNOWN_PLAN`
	EVAL_UNKNOWN_TASK                              = `EVAL_UNKNOWN_TASK`
	EVAL_UNKNOWN_VARIABLE                          = `EVAL_UNKNOWN_VARIABLE`
	EVAL_UNREFLECTABLE_RETURN                      = `EVAL_UNREFLECTABLE_RETURN`
	EVAL_UNREFLECTABLE_VALUE                       = `EVAL_UNREFLECTABLE_VALUE`
	EVAL_UNRESOLVED_TYPE                           = `EVAL_UNRESOLVED_TYPE`
	EVAL_UNRESOLVED_TYPE_OF                        = `EVAL_UNRESOLVED_TYPE_OF`
	EVAL_UNSUPPORTED_STRING_FORMAT                 = `EVAL_UNSUPPORTED_STRING_FORMAT`
	EVAL_WRONG_DEFINITION                          = `EVAL_WRONG_DEFINITION`
)

func init() {
	issue.Hard2(EVAL_ARGUMENTS_ERROR, `Error when evaluating %{expression}: %{message}`, issue.HF{`expression`: issue.A_an})

	issue.Hard(EVAL_ATTEMPT_TO_REDEFINE, `attempt to redefine %{name}`)

	issue.Hard(EVAL_ATTEMPT_TO_SET_UNSETTABLE, `attempt to set a value of kind %{kind} in an unsettable reflect.Value`)

	issue.Hard(EVAL_ATTEMPT_TO_SET_WRONG_KIND, `attempt to assign a value of kind %{expected} to a reflect.Value of kind %{actual}`)

	issue.Hard(EVAL_ATTRIBUTE_HAS_NO_VALUE, `%{label} has no value`)

	issue.Hard2(EVAL_ATTRIBUTE_NOT_FOUND, `%{type} has no attribute named %{name}`, issue.HF{`type`: issue.A_anUc})

	issue.Hard(EVAL_BAD_JSON_PATH, `unable to resolve JSON path '${path}'`)

	issue.Hard(EVAL_BAD_TYPE_STRING, `%{label} type string '%{string}' cannot be parsed into a data type: %{detail}`)

	issue.Hard(EVAL_BOTH_CONSTANT_AND_ATTRIBUTE, `attribute %{label}[%{key}] is defined as both a constant and an attribute`)

	issue.Hard(EVAL_CONSTANT_REQUIRES_VALUE, `%{label} of kind 'constant' requires a value`)

	issue.Hard(EVAL_CTOR_NOT_FOUND, `Unable to load the constructor for data type '%{type}'`)

	// TRANSLATOR 'final => false' is puppet syntax and should not be translated
	issue.Hard(EVAL_CONSTANT_WITH_FINAL, `%{label} of kind 'constant' cannot be combined with final => false`)

	issue.Hard(EVAL_DUPLICATE_KEY, `The key '%{key}' is declared more than once`)

	issue.Hard(EVAL_EMPTY_TYPE_PARAMETER_LIST, `The %{label}-Type cannot be parameterized using an empty parameter list`)

	issue.Hard(EVAL_EQUALITY_ATTRIBUTE_NOT_FOUND, `%{label} equality is referencing non existent attribute '%{attribute}'`)

	issue.Hard(EVAL_EQUALITY_NOT_ATTRIBUTE, `{label} equality is referencing %{attribute}. Only attribute references are allowed`)

	issue.Hard(EVAL_EQUALITY_ON_CONSTANT, `%{label} equality is referencing constant %{attribute}.`)

	issue.Hard(EVAL_EQUALITY_REDEFINED, `%{label} equality is referencing %{attribute} which is included in equality of %{including_parent}`)

	issue.Hard(EVAL_FAILURE, `%{message}`)

	issue.Hard(EVAL_FILE_NOT_FOUND, `File '%{path}' does not exist`)

	issue.Hard(EVAL_FILE_READ_DENIED, `Insufficient permissions to read '%{path}'`)

	issue.Hard(EVAL_GO_FUNCTION_ERROR, `Go function %{name} returned error '%{error}'`)

	issue.Hard(EVAL_GO_RUNTIME_TYPE_WITHOUT_GO_TYPE, `Attempt to create a Runtime['go', '%{name}'] without providing a Go type`)

	issue.Hard2(EVAL_ILLEGAL_ARGUMENT,
		`Error when evaluating %{expression}, argument %{number}:  %{message}`, issue.HF{`expression`: issue.A_an})

	issue.Hard2(EVAL_ILLEGAL_ARGUMENT_COUNT,
		`Error when evaluating %{expression}: Expected %{expected} arguments, got %{actual}`,
		issue.HF{`expression`: issue.A_an})

	issue.Hard2(EVAL_ILLEGAL_ARGUMENT_TYPE,
		`Error when evaluating %{expression}: Expected argument %{number} to be %{expected}, got %{actual}`,
		issue.HF{`expression`: issue.A_an})

	issue.Hard2(EVAL_ILLEGAL_ASSIGNMENT, `Illegal attempt to assign to %{value}. Not an assignable reference`,
		issue.HF{`value`: issue.A_an})

	issue.Hard(EVAL_ILLEGAL_BREAK, `break() from context where this is illegal`)

	issue.Hard2(EVAL_ILLEGAL_WHEN_STATIC_EXPRESSION, `%{expression} is illegal within a type declaration`, issue.HF{`expression`: issue.A_anUc})

	issue.Hard(EVAL_ILLEGAL_KIND_VALUE_COMBINATION, `%{label} of kind '%{kind}' cannot be combined with an attribute value`)

	issue.Hard(EVAL_ILLEGAL_NEXT, `next() from context where this is illegal`)

	issue.Hard(EVAL_ILLEGAL_OBJECT_INHERITANCE, `An Object can only inherit another Object or alias thereof. The %{label} inherits from a %{type}.`)

	issue.Hard(EVAL_ILLEGAL_RETURN, `return() from context where this is illegal`)

	issue.Hard(EVAL_ILLEGAL_MULTI_ASSIGNMENT_SIZE, `Mismatched number of assignable entries and values, expected %{expected}, got %{actual}`)

	issue.Hard(EVAL_ILLEGAL_REASSIGNMENT, `Cannot reassign variable '$%{var}'`)

	issue.Hard(EVAL_IMPL_ALREDY_REGISTERED, `The type %{type} is already present in the implementation registry`)

	issue.Hard(EVAL_IS_DIRECTORY, `The path '%{path}' is a directory`)

	issue.Hard(EVAL_INSTANCE_DOES_NOT_RESPOND, `An instance of %<instance>T does not respond to %{message}`)

	issue.Hard(EVAL_INVALID_CHARACTERS_IN_NAME, `Name '%{name} contains invalid characters. Must start with letter and only contain letters, digits, and underscore'`)

	issue.Hard(EVAL_INVALID_REGEXP, `Cannot compile regular expression '${pattern}': %{detail}`)

	issue.Hard2(EVAL_INVALID_SOURCE_FOR_GET, `Cannot create a reflect.Value from %{type}`, issue.HF{`type`: issue.A_an})

	issue.Hard2(EVAL_INVALID_SOURCE_FOR_SET, `Cannot set a reflect.Value from %{type}`, issue.HF{`type`: issue.A_an})

	issue.Hard(EVAL_INVALID_STRING_FORMAT_SPEC, `The string format '%{format}' is not a valid format on the form '%%<flags><width>.<prec><format>'`)

	issue.Hard(EVAL_INVALID_STRING_FORMAT_DELIMITER, `Only one of the delimiters [ { ( < | can be given in the string format flags, got '%<delimiter>c'`)

	issue.Hard(EVAL_INVALID_STRING_FORMAT_REPEATED_FLAG, `The same flag can only be used once in a string format, got '%{format}'`)

	issue.Hard(EVAL_INVALID_TIMEZONE, `Unable to load timezone '%{zone}': %{detail}`)

	issue.Hard(EVAL_INVALID_TYPEDNAME_MAPKEY, `The key '%{mapkey}' does not represent a valid TypedName`)

	issue.Hard(EVAL_INVALID_VERSION, `Cannot parse a semantic version from string '%{str}': '%{detail}'`)

	issue.Hard(EVAL_INVALID_VERSION_RANGE, `Cannot parse a semantic version range from string '%{str}': '%{detail}'`)

	issue.Hard(EVAL_INVALID_URI, `Cannot parse an URI from string '%{str}': '%{detail}'`)

	issue.Hard(EVAL_MATCH_NOT_REGEXP, `Can not convert right match operand to a regular expression. Caused by '%{detail}'`)

	issue.Hard2(EVAL_MATCH_NOT_STRING, `"Left match operand must result in a String value. Got %{left}`, issue.HF{`left`: issue.A_an})

	issue.Hard(EVAL_MEMBER_NAME_CONFLICT, `%{label} conflicts with attribute with the same name`)

	issue.Hard(EVAL_MISSING_MULTI_ASSIGNMENT_KEY, `No value for required key '%{name}' in assignment to variables from hash`)

	issue.Hard(EVAL_MISSING_REGEXP_IN_TYPE, `Given Regexp Type has no regular expression`)

	issue.Hard(EVAL_MISSING_REQUIRED_ATTRIBUTE, `%{label} requires a value but none was provided`)

	issue.Hard(EVAL_MISSING_TYPE_PARAMETER, `'%{name}' is not a known type parameter for %{label}-Type`)

	issue.Hard(EVAL_OBJECT_INHERITS_SELF, `The Object type '%{label}' inherits from itself`)

	issue.Hard(EVAL_NO_ATTRIBUTE_READER, `No attribute reader is implemented for %{label}`)

	issue.Hard(EVAL_NO_CURRENT_CONTEXT, `There is no current evaluation context`)

	issue.Hard(EVAL_NO_DEFINITION, `The code loaded from %{source} does not define the %{type} '%{name}`)

	issue.Hard(EVAL_NOT_COLLECTION_AT, `The given data does not contain a Collection at %{walked_path}, got '%{klass}'`)

	issue.Hard(EVAL_NOT_INTEGER, `The value '%{value}' cannot be converted to an Integer`)

	issue.Hard(EVAL_NOT_EXPECTED_TYPESET, `The code loaded from %{source} does not define the TypeSet %{name}'`)

	issue.Hard(EVAL_NOT_NUMERIC, `The value '%{value}' cannot be converted to Numeric`)

	issue.Hard(EVAL_NOT_ONLY_DEFINITION, `The code loaded from %{source} must contain only the %{type} '%{name}`)

	issue.Hard2(EVAL_NOT_PARAMETERIZED_TYPE, `%{type} is not a parameterized type`,
		issue.HF{`type`: issue.A_anUc})

	issue.Hard(EVAL_NOT_SEMVER, `The value cannot be converted to semantic version. Caused by '%{detail}'`)

	issue.Hard(EVAL_NOT_SUPPORTED_BY_GO_TIME_LAYOUT, `The format specifier '%{format_specifier}' "%{description}" can not be converted to a Go Time Layout`)

	issue.Hard2(EVAL_OPERATOR_NOT_APPLICABLE, `Operator '%{operator}' is not applicable to %{left}`,
		issue.HF{`left`: issue.A_an})

	issue.Hard2(EVAL_OPERATOR_NOT_APPLICABLE_WHEN,
		`Operator '%{operator}' is not applicable to %{left} when right side is %{right}`,
		issue.HF{`left`: issue.A_an, `right`: issue.A_an})

	issue.Hard(EVAL_OVERRIDE_MEMBER_MISMATCH, `%{member} attempts to override %{label}`)

	issue.Hard(EVAL_OVERRIDDEN_NOT_FOUND, `expected %{label} to override an inherited %{feature_type}, but no such %{feature_type} was found`)

	// TRANSLATOR 'override => true' is a puppet syntax and should not be translated
	issue.Hard(EVAL_OVERRIDE_IS_MISSING, `%{member} attempts to override %{label} without having override => true`)

	issue.Hard(EVAL_OVERRIDE_OF_FINAL, `%{member} attempts to override final %{label}`)

	issue.Hard(EVAL_PARSE_ERROR, `Unable to parse %{language}. Detail: %{detail}`)

	issue.Hard(EVAL_SERIALIZATION_ATTRIBUTE_NOT_FOUND, `%{label} serialization is referencing non existent attribute '%{attribute}'`)

	issue.Hard(EVAL_SERIALIZATION_NOT_ATTRIBUTE, `{label} serialization is referencing %{attribute}. Only attribute references are allowed`)

	issue.Hard(EVAL_SERIALIZATION_BAD_KIND, `%{label} equality is referencing {kind} %{attribute}.`)

	issue.Hard(EVAL_SERIALIZATION_DEFAULT_CONVERTED_TO_STRING, `%{path} contains the special value default. It will be converted to the String 'default'`)

	issue.Hard(EVAL_SERIALIZATION_ENDLESS_RECURSION, `Endless recursion detected when attempting to serialize value of class %{type_name}'`)

	issue.Hard2(EVAL_SERIALIZATION_UNKNOWN_CONVERTED_TO_STRING, `%{path} contains %{klass} value. It will be converted to the String '%{value}'`, issue.HF{`klass`: issue.A_an})

	issue.Hard(EVAL_SERIALIZATION_REQUIRED_AFTER_OPTIONAL, `%{label} serialization is referencing required %{required} after optional %{optional}. Optional attributes must be last`)

	issue.Hard(EVAL_TASK_BAD_JSON, `Unable to parse task metadata from '%{path}': %{detail}`)

	issue.Hard(EVAL_TASK_INITIALIZER_NOT_FOUND, `Unable to load the initializer for the Task data`)

	issue.Hard(EVAL_TASK_NO_EXECUTABLE_FOUND, `No source besides task metadata was found in directory %{directory} for task %{name}`)

	issue.Hard(EVAL_TASK_NOT_JSON_OBJECT, `The content of '%{path}' does not represent a JSON Object`)

	issue.Hard(EVAL_TASK_TOO_MANY_FILES, `Only one file can exists besides the .json file for task %{name} in directory %{directory}`)

	issue.Hard(EVAL_TIMESPAN_BAD_FSPEC, `Bad format specifier '%{expression}' in '%{format}', at position %{position}`)

	issue.Hard(EVAL_TIMESPAN_CANNOT_BE_PARSED, `Unable to parse Timespan '%{str}' using any of the formats %{formats}`)

	issue.Hard(EVAL_TIMESPAN_FSPEC_NOT_HIGHER, `Format specifiers %L and %N denotes fractions and must be used together with a specifier of higher magnitude`)

	issue.Hard(EVAL_TIMESTAMP_CANNOT_BE_PARSED, `Unable to parse Timestamp '%{str}' using any of the formats %{formats}`)

	issue.Hard(EVAL_TIMESTAMP_TZ_AMBIGUITY, `Parsed timezone '%{parsed}' conflicts with provided timezone argument %{given}`)

	issue.Hard(EVAL_TYPE_MISMATCH, `Type mismatch: %{detail}`)

	issue.Hard(EVAL_TYPESET_ALIAS_COLLIDES, `TypeSet '%{name}' references a TypeSet using alias '%{ref_alias}'. The alias collides with the name of a declared type`)

	issue.Hard(EVAL_TYPESET_MISSING_NAME_AUTHORITY, `No 'name_authority' is declared in TypeSet '%{name}' and it cannot be inferred`)

	issue.Hard(EVAL_TYPESET_REFERENCE_BAD_TYPE, `TypeSet '%{name}' reference to TypeSet named %{ref_name} resoles to a %{type_name}`)

	issue.Hard(EVAL_TYPESET_REFERENCE_DUPLICATE, `TypeSet '%{name}' references a TypeSet using alias '%{ref_alias}' more than once`)

	issue.Hard(EVAL_TYPESET_REFERENCE_MISMATCH, `TypeSet '%{name}' reference to TypeSet named %{ref_name} resolves to an incompatible version. Expected %{version_range}, got %{version`)

	issue.Hard(EVAL_TYPESET_REFERENCE_OVERLAP, `TypeSet '%{name}' references TypeSet '%{ref_na}/%{ref_name}' more than once using overlapping version ranges`)

	issue.Hard(EVAL_TYPESET_REFERENCE_UNRESOLVED, `TypeSet '%{name}' reference to TypeSet '%{ref_name}' cannot be resolved`)

	issue.Hard(EVAL_UNABLE_TO_DESERIALIZE_TYPE, `Unable to deserialize a data type from hash %{hash}`)

	issue.Hard2(EVAL_UNABLE_TO_DESERIALIZE_VALUE, `Unable to deserialize an instance of %{type} from %{arg_type}`, issue.HF{`arg_type`: issue.A_an})

	issue.Hard(EVAL_UNABLE_TO_READ_FILE, `Unable to read file '%{path}': %{detail}`)

	issue.Hard(EVAL_UNHANDLED_EXPRESSION, `Evaluator cannot handle an expression of type %<expression>T`)

	issue.Hard(EVAL_UNHANDLED_PCORE_VERSION, `The pcore version for TypeSet '%{name}' is not understood by this runtime. Expected range %{expected_range}, got %{pcore_version}`)

	issue.Hard(EVAL_UNKNOWN_FUNCTION, `Unknown function: '%{name}'`)

	issue.Hard(EVAL_UNKNOWN_PLAN, `Unknown plan: '%{name}'`)

	issue.Hard(EVAL_UNKNOWN_TASK, `Task not found: '%{name}'`)

	issue.Hard(EVAL_UNKNOWN_VARIABLE, `Unknown variable: '$%{name}'`)

	issue.Hard(EVAL_UNREFLECTABLE_RETURN, `Unable to reflect return type of method %{type}.%{method}`)

	issue.Hard(EVAL_UNREFLECTABLE_VALUE, `Unable to create a reflect.Value from value of type '%{type}'`)

	issue.Hard(EVAL_UNRESOLVED_TYPE, `Reference to unresolved type '%{typeString}'`)

	issue.Hard(EVAL_UNRESOLVED_TYPE_OF, `Unable to resolve attribute '%{navigation}' of type '%{type}'`)

	issue.Hard(EVAL_UNSUPPORTED_STRING_FORMAT, `Illegal format '%<format>c' specified for value of %{type} type - expected one of the characters '%{supported_formats}'`)

	issue.Hard(EVAL_WRONG_DEFINITION, `The code loaded from %{source} produced %{type} with the wrong name, expected %{expected}, actual %{actual}`)
}
