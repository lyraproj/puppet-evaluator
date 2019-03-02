package types

import (
	"math"
	"reflect"

	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/utils"
)

// CommonType returns a type that both a and b are assignable to
func commonType(a eval.Type, b eval.Type) eval.Type {
	if isAssignable(a, b) {
		return a
	}
	if isAssignable(b, a) {
		return a
	}

	// Deal with mergable string types
	switch a.(type) {
	case *EnumType:
		switch b.(type) {
		case *vcStringType:
			str := b.(*vcStringType).value
			ea := a.(*EnumType)
			return NewEnumType(utils.Unique(append(ea.values, str)), ea.caseInsensitive)

		case eval.StringType:
			return DefaultStringType()

		case *EnumType:
			ea := a.(*EnumType)
			eb := b.(*EnumType)
			return NewEnumType(utils.Unique(append(ea.values, eb.values...)), ea.caseInsensitive || eb.caseInsensitive)
		}

	case *scStringType:
		switch b.(type) {
		case *scStringType:
			as := a.(*scStringType)
			bs := b.(*scStringType)
			return NewStringType(commonType(as.Size(), bs.Size()).(*IntegerType), ``)

		case eval.StringType, *EnumType:
			return DefaultStringType()
		}

	case *vcStringType:
		switch b.(type) {
		case *vcStringType:
			as := a.(*vcStringType)
			bs := b.(*vcStringType)
			return NewEnumType([]string{as.value, bs.value}, false)

		case eval.StringType:
			return DefaultStringType()

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

		case *FloatType:
			af := a.(*FloatType)
			bf := b.(*FloatType)
			return NewFloatType(math.Min(af.min, bf.min), math.Max(af.max, bf.max))

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
			return NewVariantType(UniqueTypes(append(ap.Types(), bp.Types()...))...)
		}
	}

	if isCommonNumeric(a, b) {
		return numericTypeDefault
	}
	if isCommonScalarData(a, b) {
		return scalarDataTypeDefault
	}
	if isCommonScalar(a, b) {
		return scalarTypeDefault
	}
	if isCommonData(a, b) {
		return dataTypeDefault
	}
	if isCommonRichData(a, b) {
		return richDataTypeDefault
	}
	return anyTypeDefault
}

func isCommonNumeric(a eval.Type, b eval.Type) bool {
	return isAssignable(numericTypeDefault, a) && isAssignable(numericTypeDefault, b)
}

func isCommonScalarData(a eval.Type, b eval.Type) bool {
	return isAssignable(scalarDataTypeDefault, a) && isAssignable(scalarDataTypeDefault, b)
}

func isCommonScalar(a eval.Type, b eval.Type) bool {
	return isAssignable(scalarTypeDefault, a) && isAssignable(scalarTypeDefault, b)
}

func isCommonData(a eval.Type, b eval.Type) bool {
	return isAssignable(dataTypeDefault, a) && isAssignable(dataTypeDefault, b)
}

func isCommonRichData(a eval.Type, b eval.Type) bool {
	return isAssignable(richDataTypeDefault, a) && isAssignable(richDataTypeDefault, b)
}

func init() {
	eval.CommonType = commonType
}
