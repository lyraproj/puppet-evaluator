package types

import "fmt"

func Name(t PuppetType) string {
  switch t.(type) {
  case *AnyType:
    return `Any`
  case *ArrayType:
    return `Array`
  case *BooleanType:
    return `Boolean`
  case *CollectionType:
    return `Collection`
  case *DefaultType:
    return `Default`
  case *EnumType:
    return `Enum`
  case *FloatType:
    return `Float`
  case *HashType:
    return `Hash`
  case *IntegerType:
    return `Integer`
  case *NotUndefType:
    return `NotUndef`
  case *NumericType:
    return `Numeric`
  case *OptionalType:
    return `Optional`
  case *PatternType:
    return `Pattern`
  case *RegexpType:
    return `Regexp`
  case *ScalarType:
    return `Scalar`
  case *StringType:
    return `String`
  case *StructType:
    return `Struct`
  case *TupleType:
    return `Tuple`
  case *TypeAliasType:
    return `TypeAlias`
  case *TypeType:
    return `Type`
  case *UnitType:
    return `Unit`
  case *UndefType:
    return `Undef`
  case *VariantType:
    return `Variant`
  default:
    panic(fmt.Sprintf("Unknown type %T", t))
  }
}