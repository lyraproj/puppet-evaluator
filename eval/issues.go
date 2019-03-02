package eval

import "github.com/lyraproj/issue/issue"

const (
	ArgumentsError                        = `EVAL_ARGUMENTS_ERROR`
	AttemptToRedefine                     = `EVAL_ATTEMPT_TO_REDEFINE`
	AttemptToSetUnsettable                = `EVAL_ATTEMPT_TO_SET_UNSETTABLE`
	AttemptToSetWrongKind                 = `EVAL_ATTEMPT_TO_SET_WRONG_KIND`
	AttributeHasNoValue                   = `EVAL_ATTRIBUTE_HAS_NO_VALUE`
	AttributeNotFound                     = `EVAL_ATTRIBUTE_NOT_FOUND`
	BadTypeString                         = `EVAL_BAD_TYPE_STRING`
	BothConstantAndAttribute              = `EVAL_BOTH_CONSTANT_AND_ATTRIBUTE`
	ConstantRequiresValue                 = `EVAL_CONSTANT_REQUIRES_VALUE`
	ConstantWithFinal                     = `EVAL_CONSTANT_WITH_FINAL`
	CtorNotFound                          = `EVAL_CTOR_NOT_FOUND`
	DuplicateKey                          = `EVAL_DUPLICATE_KEY`
	EmptyTypeParameterList                = `EVAL_EMPTY_TYPE_PARAMETER_LIST`
	EqualityAttributeNotFound             = `EVAL_EQUALITY_ATTRIBUTE_NOT_FOUND`
	EqualityNotAttribute                  = `EVAL_EQUALITY_NOT_ATTRIBUTE`
	EqualityOnConstant                    = `EVAL_EQUALITY_ON_CONSTANT`
	EqualityRedefined                     = `EVAL_EQUALITY_REDEFINED`
	Failure                               = `EVAL_FAILURE`
	FileNotFound                          = `EVAL_FILE_NOT_FOUND`
	FileReadDenied                        = `EVAL_FILE_READ_DENIED`
	GoFunctionError                       = `EVAL_GO_FUNCTION_ERROR`
	GoRuntimeTypeWithoutGoType            = `EVAL_GO_RUNTIME_TYPE_WITHOUT_GO_TYPE`
	IllegalArgument                       = `EVAL_ILLEGAL_ARGUMENT`
	IllegalArgumentCount                  = `EVAL_ILLEGAL_ARGUMENT_COUNT`
	IllegalArgumentType                   = `EVAL_ILLEGAL_ARGUMENT_TYPE`
	IllegalAssignment                     = `EVAL_ILLEGAL_ASSIGNMENT`
	IllegalBreak                          = `EVAL_ILLEGAL_BREAK`
	IllegalKindValueCombination           = `EVAL_ILLEGAL_KIND_VALUE_COMBINATION`
	IllegalNext                           = `EVAL_ILLEGAL_NEXT`
	IllegalObjectInheritance              = `EVAL_ILLEGAL_OBJECT_INHERITANCE`
	IllegalReturn                         = `EVAL_ILLEGAL_RETURN`
	IllegalMultiAssignmentSize            = `EVAL_ILLEGAL_MULTI_ASSIGNMENT_SIZE`
	IllegalWhenStaticExpression           = `EVAL_ILLEGAL_WHEN_STATIC_EXPRESSION`
	ImplAlreadyRegistered                 = `EVAL_IMPL_ALREADY_REGISTERED`
	IllegalReassignment                   = `EVAL_ILLEGAL_REASSIGNMENT`
	InstanceDoesNotRespond                = `EVAL_INSTANCE_DOES_NOT_RESPOND`
	ImpossibleOptional                    = `EVAL_IMPOSSIBLE_OPTIONAL`
	InvalidCharactersInName               = `EVAL_INVALID_CHARACTERS_IN_NAME`
	InvalidRegexp                         = `EVAL_INVALID_REGEXP`
	InvalidSourceForGet                   = `EVAL_INVALID_SOURCE_FOR_GET`
	InvalidSourceForSet                   = `EVAL_INVALID_SOURCE_FOR_SET`
	InvalidStringFormatSpec               = `EVAL_INVALID_STRING_FORMAT_SPEC`
	InvalidStringFormatDelimiter          = `EVAL_INVALID_STRING_FORMAT_DELIMITER`
	InvalidStringFormatRepeatedFlag       = `EVAL_INVALID_STRING_FORMAT_REPEATED_FLAG`
	InvalidTimezone                       = `EVAL_INVALID_TIMEZONE`
	InvalidTypedNameMapKey                = `EVAL_INVALID_TYPED_NAME_MAP_KEY`
	InvalidUri                            = `EVAL_INVALID_URI`
	InvalidVersion                        = `EVAL_INVALID_VERSION`
	InvalidVersionRange                   = `EVAL_INVALID_VERSION_RANGE`
	IsDirectory                           = `EVAL_IS_DIRECTORY`
	MatchNotRegexp                        = `EVAL_MATCH_NOT_REGEXP`
	MatchNotString                        = `EVAL_MATCH_NOT_STRING`
	MemberNameConflict                    = `EVAL_MEMBER_NAME_CONFLICT`
	MissingMultiAssignmentKey             = `EVAL_MISSING_MULTI_ASSIGNMENT_KEY`
	MissingRegexpInType                   = `EVAL_MISSING_REGEXP_IN_TYPE`
	MissingRequiredAttribute              = `EVAL_MISSING_REQUIRED_ATTRIBUTE`
	MissingTypeParameter                  = `EVAL_MISSING_TYPE_PARAMETER`
	NoAttributeReader                     = `EVAL_NO_ATTRIBUTE_READER`
	NoCurrentContext                      = `EVAL_NO_CURRENT_CONTEXT`
	NoDefinition                          = `EVAL_NO_DEFINITION`
	NotCollectionAt                       = `EVAL_NOT_COLLECTION_AT`
	NotExpectedTypeset                    = `EVAL_NOT_EXPECTED_TYPESET`
	NotInteger                            = `EVAL_NOT_INTEGER`
	NotOnlyDefinition                     = `EVAL_NOT_ONLY_DEFINITION`
	NotNumeric                            = `EVAL_NOT_NUMERIC`
	NotParameterizedType                  = `EVAL_NOT_PARAMETERIZED_TYPE`
	NotSemver                             = `EVAL_NOT_SEMVER`
	NotSupportedByGoTimeLayout            = `EVAL_NOT_SUPPORTED_BY_GO_TIME_LAYOUT`
	ObjectInheritsSelf                    = `EVAL_OBJECT_INHERITS_SELF`
	OperatorNotApplicable                 = `EVAL_OPERATOR_NOT_APPLICABLE`
	OperatorNotApplicableWhen             = `EVAL_OPERATOR_NOT_APPLICABLE_WHEN`
	OverrideMemberMismatch                = `EVAL_OVERRIDE_MEMBER_MISMATCH`
	OverrideTypeMismatch                  = `EVAL_OVERRIDE_TYPE_MISMATCH`
	OverriddenNotFound                    = `EVAL_OVERRIDDEN_NOT_FOUND`
	OverrideOfFinal                       = `EVAL_OVERRIDE_OF_FINAL`
	OverrideIsMissing                     = `EVAL_OVERRIDE_IS_MISSING`
	ParseError                            = `EVAL_PARSE_ERROR`
	SerializationAttributeNotFound        = `EVAL_SERIALIZATION_ATTRIBUTE_NOT_FOUND`
	SerializationNotAttribute             = `EVAL_SERIALIZATION_NOT_ATTRIBUTE`
	SerializationBadKind                  = `EVAL_SERIALIZATION_BAD_KIND`
	SerializationDefaultConvertedToString = `EVAL_SERIALIZATION_DEFAULT_CONVERTED_TO_STRING`
	SerializationRequiredAfterOptional    = `EVAL_SERIALIZATION_REQUIRED_AFTER_OPTIONAL`
	SerializationUnknownConvertedToString = `EVAL_SERIALIZATION_UNKNOWN_CONVERTED_TO_STRING`
	TaskBadJson                           = `EVAL_TASK_BAD_JSON`
	TaskInitializerNotFound               = `EVAL_TASK_INITIALIZER_NOT_FOUND`
	TaskNoExecutableFound                 = `EVAL_TASK_NO_EXECUTABLE_FOUND`
	TaskNotJsonObject                     = `EVAL_TASK_NOT_JSON_OBJECT`
	TaskTooManyFiles                      = `EVAL_TASK_TOO_MANY_FILES`
	TimespanBadFormatSpec                 = `EVAL_TIMESPAN_BAD_FORMAT_SPEC`
	CannotBeParsed                        = `EVAL_TIMESPAN_CANNOT_BE_PARSED`
	TimespanFormatSpecNotHigher           = `EVAL_TIMESPAN_FORMAT_SPEC_NOT_HIGHER`
	TimestampCannotBeParsed               = `EVAL_TIMESTAMP_CANNOT_BE_PARSED`
	TimestampTzAmbiguity                  = `EVAL_TIMESTAMP_TZ_AMBIGUITY`
	TypeMismatch                          = `EVAL_TYPE_MISMATCH`
	TypesetAliasCollides                  = `EVAL_TYPESET_ALIAS_COLLIDES`
	TypesetMissingNameAuthority           = `EVAL_TYPESET_MISSING_NAME_AUTHORITY`
	TypesetReferenceBadType               = `EVAL_TYPESET_REFERENCE_BAD_TYPE`
	TypesetReferenceDuplicate             = `EVAL_TYPESET_REFERENCE_DUPLICATE`
	TypesetReferenceMismatch              = `EVAL_TYPESET_REFERENCE_MISMATCH`
	TypesetReferenceOverlap               = `EVAL_TYPESET_REFERENCE_OVERLAP`
	TypesetReferenceUnresolved            = `EVAL_TYPESET_REFERENCE_UNRESOLVED`
	UnableToDeserializeType               = `EVAL_UNABLE_TO_DESERIALIZE_TYPE`
	UnableToDeserializeValue              = `EVAL_UNABLE_TO_DESERIALIZE_VALUE`
	UnableToReadFile                      = `EVAL_UNABLE_TO_READ_FILE`
	UnhandledPcoreVersion                 = `EVAL_UNHANDLED_PCORE_VERSION`
	UnhandledExpression                   = `EVAL_UNHANDLED_EXPRESSION`
	UnknownFunction                       = `EVAL_UNKNOWN_FUNCTION`
	UnknownVariable                       = `EVAL_UNKNOWN_VARIABLE`
	UnreflectableType                     = `EVAL_UNREFLECTABLE_TYPE`
	UnreflectableValue                    = `EVAL_UNREFLECTABLE_VALUE`
	UnresolvedType                        = `EVAL_UNRESOLVED_TYPE`
	UnresolvedTypeOf                      = `EVAL_UNRESOLVED_TYPE_OF`
	UnsupportedStringFormat               = `EVAL_UNSUPPORTED_STRING_FORMAT`
	WrongDefinition                       = `EVAL_WRONG_DEFINITION`
)

func init() {

	issue.Hard2(ArgumentsError, `Error when evaluating %{expression}: %{message}`, issue.HF{`expression`: issue.AnOrA})

	issue.Hard(AttemptToRedefine, `attempt to redefine %{name}`)

	issue.Hard(AttemptToSetUnsettable, `attempt to set a value of kind %{kind} in an unsettable reflect.Value`)

	issue.Hard(AttemptToSetWrongKind, `attempt to assign a value of kind %{expected} to a reflect.Value of kind %{actual}`)

	issue.Hard(AttributeHasNoValue, `%{label} has no value`)

	issue.Hard2(AttributeNotFound, `%{type} has no attribute named %{name}`, issue.HF{`type`: issue.UcAnOrA})

	issue.Hard(BadTypeString, `%{label} type string '%{string}' cannot be parsed into a data type: %{detail}`)

	issue.Hard(BothConstantAndAttribute, `attribute %{label}[%{key}] is defined as both a constant and an attribute`)

	issue.Hard(ConstantRequiresValue, `%{label} of kind 'constant' requires a value`)

	issue.Hard(CtorNotFound, `Unable to load the constructor for data type '%{type}'`)

	// TRANSLATOR 'final => false' is puppet syntax and should not be translated
	issue.Hard(ConstantWithFinal, `%{label} of kind 'constant' cannot be combined with final => false`)

	issue.Hard(DuplicateKey, `The key '%{key}' is declared more than once`)

	issue.Hard(EmptyTypeParameterList, `The %{label}-Type cannot be parameterized using an empty parameter list`)

	issue.Hard(EqualityAttributeNotFound, `%{label} equality is referencing non existent attribute '%{attribute}'`)

	issue.Hard(EqualityNotAttribute, `{label} equality is referencing %{attribute}. Only attribute references are allowed`)

	issue.Hard(EqualityOnConstant, `%{label} equality is referencing constant %{attribute}.`)

	issue.Hard(EqualityRedefined, `%{label} equality is referencing %{attribute} which is included in equality of %{including_parent}`)

	issue.Hard(Failure, `%{message}`)

	issue.Hard(FileNotFound, `File '%{path}' does not exist`)

	issue.Hard(FileReadDenied, `Insufficient permissions to read '%{path}'`)

	issue.Hard(GoFunctionError, `Go function %{name} returned error '%{error}'`)

	issue.Hard(GoRuntimeTypeWithoutGoType, `Attempt to create a Runtime['go', '%{name}'] without providing a Go type`)

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

	issue.Hard(IllegalKindValueCombination, `%{label} of kind '%{kind}' cannot be combined with an attribute value`)

	issue.Hard(IllegalNext, `next() from context where this is illegal`)

	issue.Hard(IllegalObjectInheritance, `An Object can only inherit another Object or alias thereof. The %{label} inherits from a %{type}.`)

	issue.Hard(IllegalReturn, `return() from context where this is illegal`)

	issue.Hard(IllegalMultiAssignmentSize, `Mismatched number of assignable entries and values, expected %{expected}, got %{actual}`)

	issue.Hard(IllegalReassignment, `Cannot reassign variable '$%{var}'`)

	issue.Hard(ImplAlreadyRegistered, `The type %{type} is already present in the implementation registry`)

	issue.Hard(IsDirectory, `The path '%{path}' is a directory`)

	issue.Hard(ImpossibleOptional, `The field %{name} cannot have the type %{type}. Optional attributes must be pointers`)

	issue.Hard(InstanceDoesNotRespond, `An instance of %{type} does not respond to %{message}`)

	issue.Hard(InvalidCharactersInName, `Name '%{name} contains invalid characters. Must start with letter and only contain letters, digits, and underscore'`)

	issue.Hard(InvalidRegexp, `Cannot compile regular expression '%{pattern}': %{detail}`)

	issue.Hard2(InvalidSourceForGet, `Cannot create a reflect.Value from %{type}`, issue.HF{`type`: issue.AnOrA})

	issue.Hard2(InvalidSourceForSet, `Cannot set a reflect.Value from %{type}`, issue.HF{`type`: issue.AnOrA})

	issue.Hard(InvalidStringFormatSpec, `The string format '%{format}' is not a valid format on the form '%%<flags><width>.<precision><format>'`)

	issue.Hard(InvalidStringFormatDelimiter, `Only one of the delimiters [ { ( < | can be given in the string format flags, got '%<delimiter>c'`)

	issue.Hard(InvalidStringFormatRepeatedFlag, `The same flag can only be used once in a string format, got '%{format}'`)

	issue.Hard(InvalidTimezone, `Unable to load timezone '%{zone}': %{detail}`)

	issue.Hard(InvalidTypedNameMapKey, `The key '%{mapKey}' does not represent a valid TypedName`)

	issue.Hard(InvalidVersion, `Cannot parse a semantic version from string '%{str}': '%{detail}'`)

	issue.Hard(InvalidVersionRange, `Cannot parse a semantic version range from string '%{str}': '%{detail}'`)

	issue.Hard(InvalidUri, `Cannot parse an URI from string '%{str}': '%{detail}'`)

	issue.Hard(MatchNotRegexp, `Can not convert right match operand to a regular expression. Caused by '%{detail}'`)

	issue.Hard2(MatchNotString, `"Left match operand must result in a String value. Got %{left}`, issue.HF{`left`: issue.AnOrA})

	issue.Hard(MemberNameConflict, `%{label} conflicts with attribute with the same name`)

	issue.Hard(MissingMultiAssignmentKey, `No value for required key '%{name}' in assignment to variables from hash`)

	issue.Hard(MissingRegexpInType, `Given Regexp Type has no regular expression`)

	issue.Hard(MissingRequiredAttribute, `%{label} requires a value but none was provided`)

	issue.Hard(MissingTypeParameter, `'%{name}' is not a known type parameter for %{label}-Type`)

	issue.Hard(ObjectInheritsSelf, `The Object type '%{label}' inherits from itself`)

	issue.Hard(NoAttributeReader, `No attribute reader is implemented for %{label}`)

	issue.Hard(NoCurrentContext, `There is no current evaluation context`)

	issue.Hard(NoDefinition, `The code loaded from %{source} does not define the %{type} '%{name}`)

	issue.Hard(NotCollectionAt, `The given data does not contain a Collection at %{walked_path}, got '%{klass}'`)

	issue.Hard(NotInteger, `The value '%{value}' cannot be converted to an Integer`)

	issue.Hard(NotExpectedTypeset, `The code loaded from %{source} does not define the TypeSet %{name}'`)

	issue.Hard(NotNumeric, `The value '%{value}' cannot be converted to Numeric`)

	issue.Hard(NotOnlyDefinition, `The code loaded from %{source} must contain only the %{type} '%{name}`)

	issue.Hard2(NotParameterizedType, `%{type} is not a parameterized type`,
		issue.HF{`type`: issue.UcAnOrA})

	issue.Hard(NotSemver, `The value cannot be converted to semantic version. Caused by '%{detail}'`)

	issue.Hard(NotSupportedByGoTimeLayout, `The format specifier '%{format_specifier}' "%{description}" can not be converted to a Go Time Layout`)

	issue.Hard2(OperatorNotApplicable, `Operator '%{operator}' is not applicable to %{left}`,
		issue.HF{`left`: issue.AnOrA})

	issue.Hard2(OperatorNotApplicableWhen,
		`Operator '%{operator}' is not applicable to %{left} when right side is %{right}`,
		issue.HF{`left`: issue.AnOrA, `right`: issue.AnOrA})

	issue.Hard(OverrideMemberMismatch, `%{member} attempts to override %{label}`)

	issue.Hard(OverriddenNotFound, `expected %{label} to override an inherited %{feature_type}, but no such %{feature_type} was found`)

	// TRANSLATOR 'override => true' is a puppet syntax and should not be translated
	issue.Hard(OverrideIsMissing, `%{member} attempts to override %{label} without having override => true`)

	issue.Hard(OverrideOfFinal, `%{member} attempts to override final %{label}`)

	issue.Hard(ParseError, `Unable to parse %{language}. Detail: %{detail}`)

	issue.Hard(SerializationAttributeNotFound, `%{label} serialization is referencing non existent attribute '%{attribute}'`)

	issue.Hard(SerializationNotAttribute, `{label} serialization is referencing %{attribute}. Only attribute references are allowed`)

	issue.Hard(SerializationBadKind, `%{label} equality is referencing {kind} %{attribute}.`)

	issue.Hard(SerializationDefaultConvertedToString, `%{path} contains the special value default. It will be converted to the String 'default'`)

	issue.Hard2(SerializationUnknownConvertedToString, `%{path} contains %{klass} value. It will be converted to the String '%{value}'`, issue.HF{`klass`: issue.AnOrA})

	issue.Hard(SerializationRequiredAfterOptional, `%{label} serialization is referencing required %{required} after optional %{optional}. Optional attributes must be last`)

	issue.Hard(TaskBadJson, `Unable to parse task metadata from '%{path}': %{detail}`)

	issue.Hard(TaskInitializerNotFound, `Unable to load the initializer for the Task data`)

	issue.Hard(TaskNoExecutableFound, `No source besides task metadata was found in directory %{directory} for task %{name}`)

	issue.Hard(TaskNotJsonObject, `The content of '%{path}' does not represent a JSON Object`)

	issue.Hard(TaskTooManyFiles, `Only one file can exists besides the .json file for task %{name} in directory %{directory}`)

	issue.Hard(TimespanBadFormatSpec, `Bad format specifier '%{expression}' in '%{format}', at position %{position}`)

	issue.Hard(CannotBeParsed, `Unable to parse Timespan '%{str}' using any of the formats %{formats}`)

	issue.Hard(TimespanFormatSpecNotHigher, `Format specifiers %L and %N denotes fractions and must be used together with a specifier of higher magnitude`)

	issue.Hard(TimestampCannotBeParsed, `Unable to parse Timestamp '%{str}' using any of the formats %{formats}`)

	issue.Hard(TimestampTzAmbiguity, `Parsed timezone '%{parsed}' conflicts with provided timezone argument %{given}`)

	issue.Hard(TypeMismatch, `Type mismatch: %{detail}`)

	issue.Hard(TypesetAliasCollides, `TypeSet '%{name}' references a TypeSet using alias '%{ref_alias}'. The alias collides with the name of a declared type`)

	issue.Hard(TypesetMissingNameAuthority, `No 'name_authority' is declared in TypeSet '%{name}' and it cannot be inferred`)

	issue.Hard(TypesetReferenceBadType, `TypeSet '%{name}' reference to TypeSet named %{ref_name} resoles to a %{type_name}`)

	issue.Hard(TypesetReferenceDuplicate, `TypeSet '%{name}' references a TypeSet using alias '%{ref_alias}' more than once`)

	issue.Hard(TypesetReferenceMismatch, `TypeSet '%{name}' reference to TypeSet named %{ref_name} resolves to an incompatible version. Expected %{version_range}, got %{version`)

	issue.Hard(TypesetReferenceOverlap, `TypeSet '%{name}' references TypeSet '%{ref_na}/%{ref_name}' more than once using overlapping version ranges`)

	issue.Hard(TypesetReferenceUnresolved, `TypeSet '%{name}' reference to TypeSet '%{ref_name}' cannot be resolved`)

	issue.Hard(UnableToDeserializeType, `Unable to deserialize a data type from hash %{hash}`)

	issue.Hard2(UnableToDeserializeValue, `Unable to deserialize an instance of %{type} from %{arg_type}`, issue.HF{`arg_type`: issue.AnOrA})

	issue.Hard(UnableToReadFile, `Unable to read file '%{path}': %{detail}`)

	issue.Hard(UnhandledExpression, `Evaluator cannot handle an expression of type %<expression>T`)

	issue.Hard(UnhandledPcoreVersion, `The pcore version for TypeSet '%{name}' is not understood by this runtime. Expected range %{expected_range}, got %{pcore_version}`)

	issue.Hard(UnknownFunction, `Unknown function: '%{name}'`)

	issue.Hard(UnknownVariable, `Unknown variable: '$%{name}'`)

	issue.Hard(UnreflectableType, `Unable to create a eval.Type from value of type '%{type}'`)

	issue.Hard(UnreflectableValue, `Unable to create a reflect.Value from value of type '%{type}'`)

	issue.Hard(UnresolvedType, `Reference to unresolved type '%{typeString}'`)

	issue.Hard(UnresolvedTypeOf, `Unable to resolve attribute '%{navigation}' of type '%{type}'`)

	issue.Hard(UnsupportedStringFormat, `Illegal format '%<format>c' specified for value of %{type} type - expected one of the characters '%{supported_formats}'`)

	issue.Hard(WrongDefinition, `The code loaded from %{source} produced %{type} with the wrong name, expected %{expected}, actual %{actual}`)
}
