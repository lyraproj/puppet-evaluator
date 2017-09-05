package types

import (
  "fmt"
  "math"
  "regexp"
  "reflect"
  "github.com/puppetlabs/go-parser/parser"
)

type (
  ArgumentsError struct {
    typeName string
    error string
  }

  IllegalArgument struct {
    typeName string
    expected string
    actual string
    index int
  }

  IllegalArgumentCount struct {
    typeName string
    expected string
    actual int
  }
)

// General error with the arguments such as min > max
func argumentsError(name string, error string) *ArgumentsError {
  return &ArgumentsError{name, error}
}

func illegalArgument(name string, index int, expected string, got interface{}) *IllegalArgument {
  return &IllegalArgument{name, expected, fmt.Sprintf("%T", got), index}
}

func illegalArgumentCount(name string, expected string, actual int) *IllegalArgumentCount {
  return &IllegalArgumentCount{name, expected, actual}
}

func ToInt(ifd interface{}) (value int64, ok bool) {
  ok = true
  switch ifd.(type) {
  case int:
    value = int64(ifd.(int))
  case int8:
    value = int64(ifd.(int8))
  case int16:
    value = int64(ifd.(int16))
  case int32:
    value = int64(ifd.(int32))
  case int64:
    value = ifd.(int64)
  case uint:
    value = int64(ifd.(uint))
  case uint8:
    value = int64(ifd.(uint8))
  case uint16:
    value = int64(ifd.(uint16))
  case uint32:
    value = int64(ifd.(uint32))
  case uint64:
    value = int64(ifd.(uint64))
  default:
    ok = false
  }
  return
}

func ToTypes(ifd interface{}) ([]PuppetType, int) {
  v := reflect.ValueOf(ifd)
  if !(v.Kind() == reflect.Slice || v.Kind() == reflect.Array) {
    return nil, 0
  }
  len := v.Len()
  types := make([]PuppetType, len)
  for idx := 0; idx < len; idx++ {
    pt, ok := v.Index(idx).Interface().(PuppetType)
    if !ok {
      return nil, idx
    }
    types[idx] = pt
  }
  return types, -1
}

func ToFloat(ifd interface{}) (value float64, ok bool) {
  ok = true
  switch ifd.(type) {
  case float32:
    value = float64(ifd.(float32))
  case float64:
    value = ifd.(float64)
  default:
    ok = false
  }
  return
}

func NewAnyType() *AnyType {
  return anyType_DEFAULT
}


func NewArrayType(args...interface{}) *ArrayType {
  nargs := len(args)
  if nargs == 0 {
    return arrayType_DEFAULT
  }
  element, ok := args[0].(PuppetType)
  if !ok {
    panic(illegalArgument(`Array`, 0, `Type`, args[0]))
  }

  var rng *IntegerType
  switch nargs {
  case 1:
    rng = integerType_POSITIVE
  case 2:
    sizeArg := args[1]
    if rng, ok = sizeArg.(*IntegerType); !ok {
      var sz int64
      sz, ok = ToInt(sizeArg)
      if !ok {
        panic(illegalArgument(`Array`, 1, `Integer or Type[Integer]`, args[0]))
      }
      rng = NewIntegerType(sz, sz)
    }
  case 3:
    var min, max int64
    if min, ok = ToInt(args[1]); !ok {
      panic(illegalArgument(`Array`, 1, `Integer`, args[1]))
    }
    if max, ok = ToInt(args[2]); !ok {
      panic(illegalArgument(`Array`, 2, `Integer`, args[2]))
    }
    rng = NewIntegerType(min, max)
  default:
    panic(illegalArgumentCount(`Array`, `0 - 2`, nargs))
  }

  if element == nil {
    element = anyType_DEFAULT
  }
  if *rng == *integerType_POSITIVE && element == anyType_DEFAULT {
    return arrayType_DEFAULT
  }
  if *rng == *integerType_ZERO && element == unitType_DEFAULT {
    return arrayType_EMPTY
  }
  return &ArrayType{rng, element}
}

func NewBinaryType() *BinaryType {
  return binaryType_DEFAULT
}

func NewBooleanType() *BooleanType {
  return booleanType_DEFAULT
}

func NewCollectionType(args...PuppetType) *CollectionType {
  switch len(args) {
  case 0:
    return collectionType_DEFAULT
  case 1:
    size, ok := args[0].(*IntegerType)
    if !ok {
      panic(illegalArgument(`Collection`, 0, `Type[Integer]`, args[0]))
    }
    if *size == *integerType_POSITIVE {
      return collectionType_DEFAULT
    }
    return &CollectionType{size}
  default:
    panic(illegalArgumentCount(`Collection`, `0 - 1`, len(args)))
  }
}

func NewDataType() *TypeAliasType {
  return dataType_DEFAULT
}

func NewDefaultType() *DefaultType {
  return defaultType_DEFAULT
}

func NewEnumType(args ...interface{}) *EnumType {
  if len(args) == 0 {
    return enumType_DEFAULT
  }
  top := len(args)
  enums := make([]string, top)
  for idx, arg := range args {
    str, ok := arg.(string)
    if !ok {
      panic(illegalArgument(`Enum`, idx, `String`, arg))
    }
    enums[idx] = str
  }
  return &EnumType{enums}
}

func NewFloatType(limits...interface{}) *FloatType {
  nargs := len(limits)
  if nargs == 0 {
    return floatType_DEFAULT
  }
  min, ok := ToFloat(limits[0])
  if !ok {
    panic(illegalArgument(`Float`, 0, `Float`, limits[0]))
  }

  var max float64
  switch nargs {
  case 1:
    max = math.MaxFloat64
  case 2:
    if max, ok = ToFloat(limits[1]); !ok {
      panic(illegalArgument(`Float`, 1, `Float`, limits[1]))
    }
  default:
    panic(illegalArgumentCount(`Float`, `0 - 2`, len(limits)))
  }

  if min == math.SmallestNonzeroFloat64 && max == math.MaxFloat64 {
    return floatType_DEFAULT
  }
  if min > max {
    panic(argumentsError(`Float`, `min is not allowed to be greater than max`))
  }
  return &FloatType{min, max}
}

func NewHashType(args...interface{}) *HashType {
  nargs := len(args)
  if nargs == 0 {
    return hashType_DEFAULT
  }

  if nargs == 1 || nargs > 4 {
    panic(illegalArgumentCount(`Hash`, `0, 2, or 3`, nargs))
  }

  keyType, ok := args[0].(PuppetType)
  if !ok {
    panic(illegalArgument(`Hash`, 0, `Type`, args[0]))
  }

  var valueType PuppetType
  valueType, ok = args[1].(PuppetType)
  if !ok {
    panic(illegalArgument(`Hash`, 1, `Type`, args[1]))
  }

  var rng *IntegerType
  switch nargs {
  case 2:
    rng = integerType_POSITIVE
  case 3:
    sizeArg := args[2]
    if rng, ok = sizeArg.(*IntegerType); !ok {
      var sz int64
      if sz, ok = ToInt(sizeArg); !ok {
        panic(illegalArgument(`Hash`, 2, `Integer or Type[Integer]`, args[2]))
      }
      rng = NewIntegerType(sz)
    }
  case 4:
    var min, max int64
    if min, ok = ToInt(args[2]); !ok {
      panic(illegalArgument(`Hash`, 2, `Integer`, args[2]))
    }
    if max, ok = ToInt(args[3]); !ok {
      panic(illegalArgument(`Hash`, 3, `Integer`, args[3]))
    }
    rng = NewIntegerType(min, max)
  }

  if keyType == nil {
    keyType = anyType_DEFAULT
  }
  if valueType == nil {
    valueType = anyType_DEFAULT
  }
  if keyType == anyType_DEFAULT && valueType == anyType_DEFAULT && rng == integerType_POSITIVE {
    return hashType_DEFAULT
  }
  return &HashType{rng, keyType, valueType}
}

func NewIntegerType(limits...interface{}) *IntegerType {
  nargs := len(limits)
  if nargs == 0 {
    return integerType_DEFAULT
  }
  min, ok := ToInt(limits[0])
  if !ok {
    panic(illegalArgument(`Integer`, 0, `Integer`, limits[0]))
  }

  var max int64
  switch len(limits) {
  case 1:
    max = math.MaxInt64
  case 2:
    max, ok = ToInt(limits[1])
    if !ok {
      panic(illegalArgument(`Integer`, 1, `Integer`, limits[1]))
    }
  default:
    panic(illegalArgumentCount(`Integer`, `0 - 2`, len(limits)))
  }

  if min == math.MinInt64 {
    if max == math.MaxInt64 {
      return integerType_DEFAULT
    }
  } else if min == 0 {
    if max == math.MaxInt64 {
      return integerType_POSITIVE
    } else if max == 0 {
      return integerType_ZERO
    }
  } else if min == 1 && max == 1 {
    return integerType_ONE
  }
  if min > max {
    panic(argumentsError(`Integer`, `min is not allowed to be greater than max`))
  }
  return &IntegerType{min, max}
}

func NewNotUndefType(args...interface{}) *NotUndefType {
  switch len(args) {
  case 0:
    return notUndefType_DEFAULT
  case 1:
    containedType, ok := args[0].(PuppetType)
    if !ok {
      panic(illegalArgument(`NotUndef`, 0, `Type`, args[0]))
    }
    if containedType == anyType_DEFAULT {
      return notUndefType_DEFAULT
    }
    return &NotUndefType{typ:containedType}
  default:
    panic(illegalArgumentCount(`NotUndef`, `0 - 1`, len(args)))
  }
}

func NewNumericType() PuppetType {
  return numericType_DEFAULT
}

func NewOptionalType(args...interface{}) *OptionalType {
  switch len(args) {
  case 0:
    return optionalType_DEFAULT
  case 1:
    containedType, ok := args[0].(PuppetType)
    if !ok {
      panic(illegalArgument(`Optional`, 0, `Type`, args[0]))
    }
    if containedType == anyType_DEFAULT {
      return optionalType_DEFAULT
    }
    return &OptionalType{typ:containedType}
  default:
    panic(illegalArgumentCount(`Optional`, `0 - 1`, len(args)))
  }
}

func NewPatternType(regexps...interface{}) *PatternType {
  cnt := len(regexps)
  if cnt > 0 {
    rs := make([]*RegexpType, cnt)
    for idx, arg := range regexps {
      rx, ok := arg.(*RegexpType)
      if !ok {
        var pattern regexp.Regexp
        pattern, ok = arg.(regexp.Regexp)
        if !ok {
          var str string
          str, ok = arg.(string)
          if !ok {
            panic(illegalArgument(`Pattern`, idx, `Type[Regexp], Regexp, or String`, rs))
          }
          rx = NewRegexpType(str)
        } else {
          rx = NewRegexpType(pattern)
        }
      }
      rs[idx] = rx
    }
    return &PatternType{rs}
  }
  return patternType_DEFAULT
}

func NewRegexpType(args...interface{}) *RegexpType {
  switch len(args) {
  case 0:
    return regexpType_DEFAULT
  case 1:
    rx := args[0]
    if str, ok := rx.(string); ok {
      if str == regexpType_DEFAULT_PATTERN {
        return regexpType_DEFAULT
      }
      return &RegexpType{nil, str}
    }
    if regexp, ok := rx.(*regexp.Regexp); ok {
      str := regexp.String()
      if str == regexpType_DEFAULT_PATTERN {
        return regexpType_DEFAULT
      }
      return &RegexpType{regexp, str}
    }
    panic(illegalArgument(`Regexp`, 0, `Regexp, or String`, rx))
  default:
    panic(illegalArgumentCount(`Regexp`, `0 - 1`, len(args)))
  }
}

func NewRichDataType() *TypeAliasType {
  return richDataType_DEFAULT
}


func NewRuntimeType(args...interface{}) *RuntimeType {
  top := len(args)
  if top > 3 {
    panic(illegalArgumentCount(`Runtime`, `0 - 3`, len(args)))
  }
  if top == 0 {
    return runtimeType_DEFAULT
  }

  name, ok := args[0].(string)
  if !ok {
    panic(illegalArgument(`Runtime`, 0, `String`, args[0]))
  }

  var runtimeName string
  if top == 1 {
    runtimeName = ``
  } else {
    runtimeName, ok = args[1].(string)
    if !ok {
      panic(illegalArgument(`Runtime`, 1, `String`, args[1]))
    }
  }

  var pattern *RegexpType
  if top == 2 {
    pattern = nil
  } else {
    pattern, ok = args[2].(*RegexpType)
    if !ok {
      panic(illegalArgument(`Runtime`, 2, `Type[Regexp]`, args[2]))
    }
  }
  if runtimeName == `` && name == `` && pattern == nil {
    return runtimeType_DEFAULT
  }
  return &RuntimeType{runtimeName, name, pattern}
}

func NewScalarDataType() *ScalarDataType {
  return scalarDataType_DEFAULT
}

func NewScalarType() *ScalarType {
  return scalarType_DEFAULT
}

func NewStringType(args...interface{}) PuppetType {
  var rng *IntegerType
  var ok bool
  switch len(args) {
  case 0:
    return stringType_DEFAULT
  case 1:
    if value, ok := args[0].(string); ok {
      if value == `` {
        return stringType_DEFAULT
      }
      sz := int64(len(value))
      return &StringType{NewIntegerType(sz, sz), value}
    }
    rng, ok = args[0].(*IntegerType)
    if !ok {
      var min int64
      min, ok = ToInt(args[0])
      if !ok {
        panic(illegalArgument(`String`, 0, `String, Integer or Type[Integer]`, args[0]))
      }
      rng = NewIntegerType(min)
    }
  case 2:
    var min, max int64
    min, ok = ToInt(args[0])
    if !ok {
      panic(illegalArgument(`String`, 0, `Integer`, args[0]))
    }
    max, ok = ToInt(args[1])
    if !ok {
      panic(illegalArgument(`String`, 1, `Integer`, args[1]))
    }
    rng = NewIntegerType(min, max)
  default:
    panic(illegalArgumentCount(`Regexp`, `0 - 2`, len(args)))
  }
  if rng == integerType_POSITIVE {
    return stringType_DEFAULT
  }
  return &StringType{rng, ``}
}

func NewStructElement(key interface{}, valueType PuppetType) *StructElement {

  var (
    name string
    keyType PuppetType
  )

  switch key.(type) {
  case string:
    name = key.(string)
    keyType = NewStringType(name)
  case *StringType:
    strType := key.(*StringType)
    keyType = strType
    name = strType.value
  case *OptionalType:
    optType := key.(*OptionalType)
    strType := optType.typ.(*StringType)
    name = strType.value
    keyType = optType
  default:
    panic("Struct key must be a string, a String type or an Optional[String] type")
  }
  return &StructElement{name, keyType, valueType}
}

func NewStructType(args...interface{}) PuppetType {
  switch len(args) {
  case 0:
    return structType_DEFAULT
  case 1:
    var elems []*StructElement
    elems, ok := args[0].([]*StructElement)
    if !ok {
      arg := reflect.ValueOf(args[0])
      if arg.Kind() != reflect.Map {
        panic(illegalArgument(`Struct`, 0, `Map[String,Type], or Map[Type,Type]`, args[0]))
      }
      top := arg.Len()
      elems = make([]*StructElement, top)
      for idx, key := range arg.MapKeys() {
        elems[idx] = NewStructElement(key.Interface(), arg.MapIndex(key).Interface().(PuppetType))
      }
    }

    required := 0
    for _, elem := range elems {
      if !IsAssignable(elem.key, undefType_DEFAULT) {
        required++
      }
    }
    size := NewIntegerType(int64(required), int64(len(elems)))
    return &StructType{size, elems, nil}
  default:
    panic(illegalArgumentCount(`Struct`, `0 - 1`, len(args)))
  }
}

func NewTupleType(args...interface{}) PuppetType {
  nargs := len(args)
  if nargs == 0 {
    return tupleType_DEFAULT
  }

  var rng, givenOrActualRng *IntegerType
  var ok bool
  var min, max int64

  last := args[nargs - 1]
  if max, ok = ToInt(last); ok {
    if nargs == 1 {
      panic(illegalArgument(`Tuple`, 0, `Type`, last))
    }
    if min, ok = ToInt(args[nargs - 2]); ok {
      rng = NewIntegerType(min, max)
      nargs -= 2
    } else {
      min = max
      nargs--
      rng = NewIntegerType(min, int64(nargs))
    }
    args = args[0:nargs]
    givenOrActualRng = rng
  } else {
    if rng, ok = last.(*IntegerType); ok {
      nargs--
      args = args[0:nargs]
      givenOrActualRng = rng
    } else {
      rng = nil
      givenOrActualRng = NewIntegerType(nargs, nargs)
    }
  }

  if nargs == 0 {
    panic(illegalArgument(`Tuple`, 0, `Type`, last))
  }

  var types []PuppetType
  ok = false
  failIdx := -1
  if nargs == 1 {
    // One arg can be either array of types or a type
    types, failIdx = ToTypes(args[0])
    ok = failIdx < 0
  }

  if !ok {
    types, failIdx = ToTypes(args)
    if failIdx >= 0 {
      panic(illegalArgument(`Tuple`, failIdx, `Type`, args[failIdx]))
    }
  }
  return &TupleType{rng, givenOrActualRng, false, types}
}

func NewTypeAliasType(args...interface{}) *TypeAliasType {
  switch len(args) {
  case 0:
    return typeAliasType_DEFAULT
  case 2:
    name, ok := args[0].(string)
    if !ok {
      panic(illegalArgument(`TypeAlias`, 0, `String`, args[0]))
    }
    var pt PuppetType
    if pt, ok = args[1].(PuppetType); ok {
      return &TypeAliasType{name, nil, pt, true, nil}
    }
    var ex parser.Expression
    if ex, ok = args[1].(parser.Expression); ok {
      return  &TypeAliasType{name, ex, nil, false, nil}
    }
    panic(illegalArgument(`TypeAlias`, 1, `Type or Expression`, args[1]))
  default:
    panic(illegalArgumentCount(`TypeAlias`, `0 or 2`, len(args)))
  }
}

func NewTypeReferenceType(args...interface{}) *TypeReferenceType {
  switch len(args) {
  case 0:
    return typeReferenceType_DEFAULT
  case 1:
    if str, ok := args[0].(string); ok {
      return &TypeReferenceType{str}
    }
    panic(illegalArgument(`TypeReference`, 0, `String`, args[0]))
  default:
    panic(illegalArgumentCount(`TypeReference`, `0 - 1`, len(args)))
  }
}

func NewTypeType(args...interface{}) *TypeType {
  switch len(args) {
  case 0:
    return typeType_DEFAULT
  case 1:
    if containedType, ok := args[0].(PuppetType); ok {
      if containedType == anyType_DEFAULT {
        return typeType_DEFAULT
      }
      return &TypeType{typ:containedType}
    }
    panic(illegalArgument(`Type`, 0, `Type`, args[0]))
  default:
    panic(illegalArgumentCount(`TypeAlias`, `0 or 1`, len(args)))
  }
}

func NewUndefType() PuppetType {
  return undefType_DEFAULT
}

func NewUnitType() PuppetType {
  return undefType_DEFAULT
}

func NewVariantType(args...interface{}) PuppetType {
  nargs := len(args)
  if nargs == 0 {
    return variantType_DEFAULT
  }

  var types []PuppetType
  failIdx := -1
  if nargs == 1 {
    types, failIdx = ToTypes(args[0])
    if failIdx >= 0 {
      panic(illegalArgument(`Variant`, 0, `Type or Array[Type]`, args[0]))
    }
  } else {
    types, failIdx = ToTypes(args)
    if failIdx >= 0 {
      panic(illegalArgument(`Variant`, failIdx, `Type`, args[failIdx]))
    }
  }
  switch len(types) {
  case 0:
    return variantType_DEFAULT
  case 1:
    return types[0]
  default:
    return &VariantType{types: types, resolved:false}
  }
}
