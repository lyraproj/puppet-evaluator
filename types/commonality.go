package types

import (
	. "math"
	"reflect"

	. "github.com/puppetlabs/go-evaluator/utils"
	. "github.com/puppetlabs/go-evaluator/evaluator"
)

// CommonType returns a type that both a and b are assignable to
func commonType(a PType, b PType) PType {
	if isAssignable(a, b) {
		return a
	}
	if isAssignable(b, a) {
		return a
	}

	// Deal with mergable types of different type
	switch a.(type) {
	case *EnumType:
		switch b.(type) {
		case *StringType:
			str := b.(*StringType).value
			if str != `` {
				return NewEnumType(Unique(append(a.(*EnumType).values, str)))
			}
		}
	case *StringType:
		switch b.(type) {
		case *EnumType:
			return commonType(b, a)
		}
	}

	// Deal with mergable types same type
	if reflect.TypeOf(a) == reflect.TypeOf(b) {
		switch a.(type) {
		case *ArrayType:
			aa := a.(*ArrayType)
			ba := b.(*ArrayType)
			return NewArrayType(commonType(aa.typ, ba.typ), commonType(aa.size, ba.size).(*IntegerType))

		case *EnumType:
			return NewEnumType(Unique(append(a.(*EnumType).values, b.(*EnumType).values...)))

		case *FloatType:
			af := a.(*FloatType)
			bf := b.(*FloatType)
			return NewFloatType(Min(af.min, bf.min), Max(af.max, bf.max))

		case *IntegerType:
			ai := a.(*IntegerType)
			bi := b.(*IntegerType)
			min := ai.min
			if bi.min < min {
				min = bi.min
			}
			max := ai.max
			if bi.max > max {
				max = bi.max
			}
			return NewIntegerType(min, max)

		case *IterableType:
			an := a.(*IterableType)
			bn := b.(*IterableType)
			return NewIterableType(commonType(an.ElementType(), bn.ElementType()))

		case *IteratorType:
			an := a.(*IteratorType)
			bn := b.(*IteratorType)
			return NewIteratorType(commonType(an.ElementType(), bn.ElementType()))

		case *NotUndefType:
			an := a.(*NotUndefType)
			bn := b.(*NotUndefType)
			return NewNotUndefType(commonType(an.ContainedType(), bn.ContainedType()))

		case *PatternType:
			ap := a.(*PatternType)
			bp := b.(*PatternType)
			return NewPatternType(UniqueRegexps(append(ap.regexps, bp.regexps...)))

		case *RuntimeType:
			ar := a.(*RuntimeType)
			br := b.(*RuntimeType)
			if ar.runtime == br.runtime {
				return NewRuntimeType(ar.runtime, ``, nil)
			}
			return DefaultRuntimeType()

		case *StringType:
			as := a.(*StringType)
			bs := b.(*StringType)
			if as.value == `` || bs.value == `` {
				return NewStringType(commonType(as.size, bs.size).(*IntegerType), ``)
			}
			return NewEnumType([]string{as.value, bs.value})

		case *TupleType:
			at := a.(*TupleType)
			bt := b.(*TupleType)
			return NewArrayType(commonType(at.CommonElementType(), bt.CommonElementType()), commonType(at.Size(), bt.Size()).(*IntegerType))

		case *TypeType:
			at := a.(*TypeType)
			bt := b.(*TypeType)
			return NewTypeType(commonType(at.ContainedType(), bt.ContainedType()))

		case *VariantType:
			ap := a.(*VariantType)
			bp := b.(*VariantType)
			return NewVariantType(UniqueTypes(append(ap.Types(), bp.Types()...)))
		}
	}

	if isCommonNumeric(a, b) {
		return numericType_DEFAULT
	}
	if isCommonScalarData(a, b) {
		return scalarDataType_DEFAULT
	}
	if isCommonScalar(a, b) {
		return scalarType_DEFAULT
	}
	if isCommonData(a, b) {
		return dataType_DEFAULT
	}
	if isCommonRichData(a, b) {
		return richDataType_DEFAULT
	}
	return anyType_DEFAULT
}

func isCommonNumeric(a PType, b PType) bool {
	return isAssignable(numericType_DEFAULT, a) && isAssignable(numericType_DEFAULT, b)
}

func isCommonScalarData(a PType, b PType) bool {
	return isAssignable(scalarDataType_DEFAULT, a) && isAssignable(scalarDataType_DEFAULT, b)
}

func isCommonScalar(a PType, b PType) bool {
	return isAssignable(scalarType_DEFAULT, a) && isAssignable(scalarType_DEFAULT, b)
}

func isCommonData(a PType, b PType) bool {
	return isAssignable(dataType_DEFAULT, a) && isAssignable(dataType_DEFAULT, b)
}

func isCommonRichData(a PType, b PType) bool {
	return isAssignable(richDataType_DEFAULT, a) && isAssignable(richDataType_DEFAULT, b)
}

func init() {
	CommonType = commonType
}
