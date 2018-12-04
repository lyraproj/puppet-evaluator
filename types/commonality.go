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

	// Deal with mergable types of different type
	switch a.(type) {
	case *EnumType:
		switch b.(type) {
		case *StringType:
			str := b.(*StringType).value
			if str != `` {
				ea := a.(*EnumType)
				return NewEnumType(utils.Unique(append(ea.values, str)), ea.caseInsensitive)
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
			ea := a.(*EnumType)
			eb := b.(*EnumType)
			return NewEnumType(utils.Unique(append(ea.values, eb.values...)), ea.caseInsensitive || eb.caseInsensitive)

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

		case *StringType:
			as := a.(*StringType)
			bs := b.(*StringType)
			if as.value == `` || bs.value == `` {
				return NewStringType(commonType(as.size, bs.size).(*IntegerType), ``)
			}
			return NewEnumType([]string{as.value, bs.value}, false)

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

func isCommonNumeric(a eval.Type, b eval.Type) bool {
	return isAssignable(numericType_DEFAULT, a) && isAssignable(numericType_DEFAULT, b)
}

func isCommonScalarData(a eval.Type, b eval.Type) bool {
	return isAssignable(scalarDataType_DEFAULT, a) && isAssignable(scalarDataType_DEFAULT, b)
}

func isCommonScalar(a eval.Type, b eval.Type) bool {
	return isAssignable(scalarType_DEFAULT, a) && isAssignable(scalarType_DEFAULT, b)
}

func isCommonData(a eval.Type, b eval.Type) bool {
	return isAssignable(dataType_DEFAULT, a) && isAssignable(dataType_DEFAULT, b)
}

func isCommonRichData(a eval.Type, b eval.Type) bool {
	return isAssignable(richDataType_DEFAULT, a) && isAssignable(richDataType_DEFAULT, b)
}

func init() {
	eval.CommonType = commonType
}
