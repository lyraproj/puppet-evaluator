package evaluator

type TypeMismatchDescriber interface {
	DescribeMismatch(expected PType, actual PType) string

	ValidateParameters(subject string, paramsStruct PType, parameters KeyedValue, missingOK bool)

	ValidateParameterValue(subject string, parameterName string, parameterType PType, value PValue)
}
