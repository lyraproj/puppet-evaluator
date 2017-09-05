package eval

import (
  . "github.com/puppetlabs/go-evaluator/types"
  . "github.com/puppetlabs/go-parser/parser"
  . "github.com/puppetlabs/go-parser/issue"
)

var coreTypes = map[string]PuppetType {
  `any`: NewAnyType(),
  `array`: NewArrayType(),
  `binary`: NewBinaryType(),
  `boolean`: NewBooleanType(),
  `collection`: NewCollectionType(),
  `data`: NewDataType(),
  `default`: NewDefaultType(),
  `enum`: NewEnumType(),
  `float`: NewFloatType(),
  `hash`: NewHashType(),
  `integer`: NewIntegerType(),
  `notundef`: NewNotUndefType(),
  `numeric`: NewNumericType(),
  `optional`: NewOptionalType(),
  `pattern`: NewPatternType(),
  `regexp`: NewRegexpType(),
  `richdata`: NewRichDataType(),
  `runtime`: NewRuntimeType(),
  `scalardata`: NewScalarDataType(),
  `scalar`: NewScalarType(),
  `string`: NewStringType(),
  `struct`: NewStructType(),
  `tuple`: NewTupleType(),
  `typealias`: NewTypeAliasType(),
  `typereference`: NewTypeReferenceType(),
  `type`: NewTypeType(),
  `undef`: NewUndefType(),
  `unit`: NewUnitType(),
  `variant`: NewVariantType(),
}

type evaluator struct {
}

func (e *evaluator) evalError(code IssueCode, semantic Expression, args...interface{}) *ReportedIssue {
  return NewReportedIssue(code, SEVERITY_ERROR, args, semantic)
}

func (e *evaluator) eval(expr Expression) interface{} {
  switch e.(type) {
  case *AccessExpression:
    return e.eval_AccessExpression(expr.(*AccessExpression))
  case *BlockExpression:
    return e.eval_BlockExpression(expr.(*BlockExpression))
  default:
    panic(e.evalError(EVAL_UNHANDLED_EXPRESSION, expr, expr))
  }
}

func (e *evaluator) eval_AccessExpression(expr *AccessExpression) (result interface{}) {
  qr, ok := expr.Operand().(*QualifiedReference)
  if !ok {
    panic(e.evalError(EVAL_LHS_MUST_BE_QREF, expr))
  }
  dcName := qr.DowncasedName()
  keys := expr.Keys()
  top  := len(keys)
  args := make([]interface{}, top)
  for idx, key := range keys {
    args[idx] = e.eval(key)
  }
  switch dcName {
  case `array`:
    return NewArrayType(args...)
  case `enum`:
    return NewEnumType(args...)
  case `float`:
    return NewFloatType(args...)
  case `hash`:
    return NewHashType(args...)
  case `integer`:
    return NewIntegerType(args...)
  case `notundef`:
    return NewNotUndefType(args...)
  case `optional`:
    return NewOptionalType(args...)
  case `pattern`:
    return NewPatternType(args...)
  case `regexp`:
    return NewRegexpType(args...)
  case `runtime`:
    return NewRuntimeType(args...)
  case `string`:
    return NewStringType(args...)
  case `struct`:
    return NewStringType(args...)
  case `tuple`:
    return NewTupleType(args...)
  case `type`:
    return NewTypeType(args...)
  case `typereference`:
    return NewTypeReferenceType(args...)
  case `variant`:
    return NewTypeReferenceType(args...)

  }
  return result
}

func (e *evaluator) eval_BlockExpression(expr *BlockExpression) (result interface{}) {
  for _, statement := range expr.Statements() {
    result = e.eval(statement)
  }
  return result
}

