package types

import (
  "unsafe"
  "reflect"
  "github.com/puppetlabs/go-evaluator/pcore"
  "math"
  "regexp"
  "github.com/puppetlabs/go-evaluator/utils"
  "fmt"
  "github.com/puppetlabs/go-parser/parser"
  "time"
)

type (
  PuppetType interface {
    isInstance(o interface{}, guard map[visit]bool) bool

    isAssignable(t PuppetType, guard map[visit]bool) bool
  }

  TypeWithSize interface {
    Size() *IntegerType
  }

  // During possibly recursive operations, must keep track of checks that are
  // in progress. The comparison algorithm assumes that all checks in progress
  // are true when it reencounters them. Visited comparisons are stored in a map
  // indexed by visit.
  //
  // (algorithm copied from golang reflect/deepequal.go)
  visit struct {
    a1  unsafe.Pointer
    a2  unsafe.Pointer
    typ reflect.Type
  }

  typeContainerType struct {
    typ PuppetType
  }

  typesContainerType struct {
    types []PuppetType
    resolved bool
  }

  defaultInstance struct {}

  AnyType struct {}

  ArrayType struct {
    size *IntegerType
    typ PuppetType
  }

  BooleanType struct {}

  BinaryType struct {}

  CollectionType struct {
    size *IntegerType
  }

  DefaultType struct {}

  EnumType struct {
    values []string
  }

  FloatType struct {
    min float64
    max float64
  }

  HashType struct {
    size *IntegerType
    keyType PuppetType
    valueType PuppetType
  }

  IntegerType struct {
    min int64
    max int64
  }

  NotUndefType typeContainerType

  NumericType struct {}

  OptionalType typeContainerType

  PatternType struct {
    regexps []*RegexpType
  }

  RegexpType struct {
    pattern *regexp.Regexp
    patternString string
  }

  RuntimeType struct {
    name string
    runtime string
    pattern *RegexpType
  }

  ScalarDataType struct {}

  ScalarType struct {}

  StringType struct {
    size *IntegerType
    value string
  }

  StructElement struct {
    name string
    key PuppetType
    value PuppetType
  }

  StructType struct {
    size *IntegerType
    elements []*StructElement
    hashedMembers map[string]*StructElement
  }

  TimespanType struct {
    min time.Time
    max time.Time
  }

  TimestampType struct {
    min time.Duration
    max time.Duration
  }

  TupleType struct {
    size *IntegerType
    givenOrActualSize *IntegerType
    resolved bool
    types []PuppetType
  }

  TypeAliasType struct {
    name string
    typeExpression parser.Expression
    resolvedType PuppetType
    selfRecursion bool
    pcore pcore.Pcore
  }

  TypeReferenceType struct {
    typeString string
  }

  TypeType typeContainerType

  UnitType struct {}

  UndefType struct {}

  VariantType typesContainerType
)

func IsInstance(a PuppetType, o interface{}) bool {
  if(isRecursive(a)) {
    return isInstance(a, o, make(map[visit]bool))
  }
  return isInstance(a, o, nil)
}

func IsAssignable(a PuppetType, b PuppetType) bool {
  if(isRecursive(a)) {
    return isAssignable(a, b, make(map[visit]bool))
  }
  return isAssignable(a, b, nil)
}

func isRecursive(t PuppetType) bool {
  // TODO
  return false
}

func isInstance(a PuppetType, o interface{}, guard map[visit]bool) bool {
  return a.isInstance(o, guard)
}

func isAssignable(a PuppetType, b PuppetType, guard map[visit]bool) bool {
  switch b.(type) {
  case nil:
    return false
  case *AnyType, *UnitType:
    return true
  case *NotUndefType:
    nt := b.(*NotUndefType).typ
    if !isAssignable(nt, undefType_DEFAULT, guard) {
      return isAssignable(a, nt, guard)
    }
    return false
  case *OptionalType:
    if isAssignable(a, undefType_DEFAULT, guard) {
      ot := b.(*OptionalType).typ
      return ot == nil || isAssignable(a, ot, guard)
    }
    return false
  case *TypeAliasType:

  case *VariantType:
    return b.(*VariantType).allAssignableTo(a, guard)
  }
  return a.isAssignable(b, guard)
}

func (t *AnyType) isAssignable(o PuppetType, guard map[visit]bool) bool {
  return true
}

func (t *AnyType) isInstance(o interface{}, guard map[visit]bool) bool {
  return true
}

var anyType_DEFAULT = &AnyType{}




func (t *ArrayType) isAssignable(o PuppetType, guard map[visit]bool) bool {
  switch o.(type) {
  case *ArrayType:
    oa := o.(*ArrayType)
    return isAssignable(t.size, oa.size, guard) && isAssignable(t.typ, oa.typ, guard)
  case *TupleType:
    ot := o.(*TupleType)
    return isAssignable(t.size, ot.size, guard) && allAssignableTo(ot.types, t.typ, guard)
  default:
    return false
  }
  return true
}

func (t *ArrayType) isInstance(o interface{}, guard map[visit]bool) bool {
  v := reflect.ValueOf(o)
  if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
    return false
  }

  osz := v.Len()
  if !isInstance(t.size, osz, guard) {
    return false
  }

  if t.typ == anyType_DEFAULT {
    return true
  }

  for idx := 0; idx < osz; idx++ {
    if !isInstance(t.typ, v.Index(idx).Interface(), guard) {
      return false
    }
  }
  return true
}

func (t *ArrayType) Size() *IntegerType {
  return t.size
}

var arrayType_DEFAULT = &ArrayType{integerType_POSITIVE,anyType_DEFAULT}
var arrayType_EMPTY = &ArrayType{integerType_ZERO, unitType_DEFAULT}

func (t *BinaryType) isAssignable(o PuppetType, guard map[visit]bool) bool {
  _, ok := o.(*BinaryType)
  return ok
}

func (t *BinaryType) isInstance(o interface{}, guard map[visit]bool) bool {
  _, ok := o.([]byte)
  return ok
}

var binaryType_DEFAULT = &BinaryType{}

func (t *BooleanType) isAssignable(o PuppetType, guard map[visit]bool) bool {
  _, ok := o.(*BooleanType)
  return ok
}

func (t *BooleanType) isInstance(o interface{}, guard map[visit]bool) bool {
  _, ok := o.(bool)
  return ok
}

var booleanType_DEFAULT = &BooleanType{}




func (t *CollectionType) isAssignable(o PuppetType, guard map[visit]bool) bool {
  switch o.(type) {
  case *CollectionType:
  case *ArrayType:
  case *HashType:
  case *TupleType:
  default:
    return false
  }
  return true
}

func (t *CollectionType) isInstance(o interface{}, guard map[visit]bool) bool {
  return true
}

func (t *CollectionType) Size() *IntegerType {
  return t.size
}

var collectionType_DEFAULT = &CollectionType{}




var dataArrayType_DEFAULT = NewArrayType(NewTypeReferenceType(`Data`))
var dataHashType_DEFAULT = NewHashType(NewStringType(), NewTypeReferenceType(`Data`))
var dataType_DEFAULT = NewTypeAliasType(`Data`, NewVariantType(
  NewScalarDataType(), NewUndefType(), dataArrayType_DEFAULT, dataHashType_DEFAULT))






var richKeyType_DEFAULT = NewVariantType(NewStringType(), NewNumericType())
var richDataArrayType_DEFAULT = NewArrayType(NewTypeReferenceType(`RichData`))
var richDataHashType_DEFAULT = NewHashType(richKeyType_DEFAULT, NewTypeReferenceType(`RichData`))
var richDataType_DEFAULT = NewTypeAliasType(`RichData`, NewVariantType(
  NewScalarType(), NewBinaryType(), NewDefaultType(), NewTypeType(), NewUndefType(), richDataArrayType_DEFAULT, richDataHashType_DEFAULT))

func init() {
  // "resolve" the dataType and richDataType
  dataArrayType_DEFAULT.typ = dataType_DEFAULT
  dataHashType_DEFAULT.valueType = dataType_DEFAULT
  richDataArrayType_DEFAULT.typ = richDataType_DEFAULT
  richDataHashType_DEFAULT.valueType = richDataType_DEFAULT
}




func (t *DefaultType) isAssignable(o PuppetType, guard map[visit]bool) bool {
  return o == defaultType_DEFAULT
}

func (t *DefaultType) isInstance(o interface{}, guard map[visit]bool) bool {
  return o == DEFAULT_INSTANCE
}

var DEFAULT_INSTANCE = &defaultInstance{}

var defaultType_DEFAULT = &DefaultType{}


func (t *EnumType) isAssignable(o PuppetType, guard map[visit]bool) bool {
  if len(t.values) == 0 {
    switch o.(type) {
    case *StringType, *EnumType, *PatternType:
      return true
    }
    return false
  }

  if st, ok := o.(*StringType); ok {
    return utils.ContainsString(t. values, st.value)
  }

  if en, ok := o.(*EnumType); ok {
    oEnums := en.values
    return len(oEnums) > 0 && utils.ContainsAllStrings(t.values, oEnums)
  }
  return false
}

func (t *EnumType) isInstance(o interface{}, guard map[visit]bool) bool {
  str, ok := o.(string)
  return ok && (len(t.values) == 0 || utils.ContainsString(t.values, str))
}

var enumType_DEFAULT = &EnumType{[]string{}}




func (t *FloatType) isAssignable(o PuppetType, guard map[visit]bool) bool {
  if ft, ok := o.(*FloatType); ok {
    return t.min <= ft.min && t.max >= ft.max;
  }
  return false
}

func (t *FloatType) isInstance(o interface{}, guard map[visit]bool) bool {
  switch o.(type) {
  case float32, float64:
    vf := reflect.ValueOf(o).Float()
    return t.min <= vf && vf <= t.max
  default:
    return false
  }
}

func (t *FloatType) IsUnbounded() bool {
  return t.min == math.SmallestNonzeroFloat64 && t.max == math.MaxFloat64;
}

var floatType_DEFAULT = &FloatType{math.MinInt64, math.MaxInt64}




func (t *HashType) isAssignable(o PuppetType, guard map[visit]bool) bool {
  switch o.(type) {
  case *HashType:
    if t.size.min == 0 && o == hashType_EMPTY {
      return true
    }
    ht := o.(*HashType)
    return isAssignable(t.size, ht.size, guard) && isAssignable(t.keyType, ht.keyType, guard) && isAssignable(t.valueType, ht.valueType, guard)
  case *StructType:
    st := o.(*StructType)
    if !isAssignable(t.size, st.size, guard) {
      return false
    }
    for _, element := range st.elements {
      if !(isAssignable(t.keyType, element.ActualKeyType(), guard) && isAssignable(t.valueType, element.value, guard)) {
        return false
      }
    }
    return true
  default:
    return false
  }
}

func (t *HashType) isInstance(o interface{}, guard map[visit]bool) bool {
  v := reflect.ValueOf(o)
  if v.Kind() != reflect.Map {
    return false
  }
  top := v.Len()
  if !isInstance(t.size, top, guard) {
    return false
  }
  for _, k := range v.MapKeys() {
    if !(isInstance(t.keyType, k.Interface(), guard) && isInstance(t.valueType, v.MapIndex(k).Interface(), guard)) {
      return false
    }
  }
  return true
}

var hashType_EMPTY = &HashType{integerType_ZERO, unitType_DEFAULT, unitType_DEFAULT}
var hashType_DEFAULT = &HashType{integerType_POSITIVE, anyType_DEFAULT, anyType_DEFAULT}




func (t *IntegerType) isAssignable(o PuppetType, guard map[visit]bool) bool {
  if it, ok := o.(*IntegerType); ok {
    return t.min <= it.min && t.max >= it.max;
  }
  return false
}

func (t *IntegerType) isInstance(o interface{}, guard map[visit]bool) bool {
  switch o.(type) {
  case int, int8, int16, int32, int64:
    vs := reflect.ValueOf(o).Int()
    return t.min <= vs && vs <= t.max
  case uint, uint8, uint16, uint32, uint64:
    vu := reflect.ValueOf(o).Uint()
    return t.min < 0 || uint64(t.min) <= vu && t.max >= 0 && vu <= uint64(t.max)
  default:
    return false
  }
}

func (t *IntegerType) IsUnbounded() bool {
  return t.min == math.MinInt64 && t.max == math.MaxInt64;
}

var integerType_DEFAULT = &IntegerType{math.MinInt64, math.MaxInt64}
var integerType_POSITIVE = &IntegerType{0, math.MaxInt64}
var integerType_ZERO = &IntegerType{0,0}
var integerType_ONE = &IntegerType{1,1}



func (t *NotUndefType) isAssignable(o PuppetType, guard map[visit]bool) bool {
  return !isAssignable(o, undefType_DEFAULT, guard) && isAssignable(t.typ, o, guard)
}

func (t *NotUndefType) isInstance(o interface{}, guard map[visit]bool) bool {
  return o != nil && isInstance(t.typ, o, guard)
}

var notUndefType_DEFAULT = &NotUndefType{typ:anyType_DEFAULT}





func (t *NumericType) isAssignable(o PuppetType, guard map[visit]bool) bool {
  switch o.(type) {
  case *IntegerType, *FloatType:
    return true
  default:
    return false
  }
}

func (t *NumericType) isInstance(o interface{}, guard map[visit]bool) bool {
  switch o.(type) {
  case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
    return true
  default:
    return false
  }
}

var numericType_DEFAULT = &NumericType{}




func (t *OptionalType) isAssignable(o PuppetType, guard map[visit]bool) bool {
  return isAssignable(o, undefType_DEFAULT, guard) || isAssignable(t.typ, o, guard)
}

func (t *OptionalType) isInstance(o interface{}, guard map[visit]bool) bool {
  return o == nil || isInstance(t.typ, o, guard)
}

var optionalType_DEFAULT = &OptionalType{typ:anyType_DEFAULT}




func (t *PatternType) isAssignable(o PuppetType, guard map[visit]bool) bool {
  if st, ok := o.(*StringType); ok {
    if len(t.regexps) == 0 {
      return true
    }
    str := st.value
    return str != `` && utils.MatchesString(MapToRegexps(t.regexps), str)
  }

  if et, ok := o.(*EnumType); ok {
    if len(t.regexps) == 0 {
      return true
    }
    enums := et.values
    return len(enums) > 0 && utils.MatchesAllStrings(MapToRegexps(t.regexps), enums)
  }
  return false
}

func (t *PatternType) isInstance(o interface{}, guard map[visit]bool) bool {
  str, ok := o.(string)
  return ok && (len(t.regexps) == 0 || utils.MatchesString(MapToRegexps(t.regexps), str))
}

var patternType_DEFAULT = &PatternType{[]*RegexpType{}}



func (t *RegexpType) isAssignable(o PuppetType, guard map[visit]bool) bool {
  rx, ok := o.(*RegexpType)
  return ok && (t.patternString == regexpType_DEFAULT_PATTERN || t.patternString == rx.patternString)
}

func (t *RegexpType) isInstance(o interface{}, guard map[visit]bool) bool {
  rx, ok := o.(*regexp.Regexp)
  return ok && (t.patternString == regexpType_DEFAULT_PATTERN || t.patternString == rx.String())
}

func (t *RegexpType) Regexp() *regexp.Regexp {
  // TODO this is not thread safe. Add mutex
  if t.pattern == nil {
    t.pattern = regexp.MustCompile(t.patternString)
  }
  return t.pattern
}

func MapToRegexps(regexpTypes []*RegexpType) []*regexp.Regexp {
  top :=  len(regexpTypes)
  result := make([]*regexp.Regexp, top)
  for idx := 0; idx < top; idx++ {
    result[idx] = regexpTypes[idx].Regexp()
  }
  return result
}

var regexpType_DEFAULT_PATTERN = `.*`
var regexpType_DEFAULT = &RegexpType{regexp.MustCompile(regexpType_DEFAULT_PATTERN), regexpType_DEFAULT_PATTERN}








func (t *RuntimeType) isAssignable(o PuppetType, guard map[visit]bool) bool {
  if rt, ok := o.(*RuntimeType); ok {
    if t.runtime == `` {
      return true
    }
    if t.runtime != rt.runtime {
      return false
    }
    if t.name == `` {
      return true
    }
    if t.pattern != nil {
      return t.name == rt.name && rt.pattern != nil && t.pattern.patternString == rt.pattern.patternString
    }
    if t.name == rt.name {
      return true
    }
    // There is no way to turn a string into a Type and then check assignability in Go
  }
  return false
}

func (t *RuntimeType) isInstance(o interface{}, guard map[visit]bool) bool {
  if(o == nil || t.runtime != `go` || t.name == `` || t.pattern != nil) {
    return false
  }
  return reflect.TypeOf(o).Name() == t.name
}

var runtimeType_DEFAULT = &RuntimeType{``, ``, nil}




func (t *ScalarType) isAssignable(o PuppetType, guard map[visit]bool) bool {
  switch o.(type) {
  case *ScalarType, *ScalarDataType:
    return true
  default:
    return isAssignable(stringType_DEFAULT, o, guard) ||
        isAssignable(numericType_DEFAULT, o, guard) ||
        isAssignable(booleanType_DEFAULT, o, guard) ||
        isAssignable(regexpType_DEFAULT, o, guard)
  }
}

func (t *ScalarType) isInstance(o interface{}, guard map[visit]bool) bool {
  switch o.(type) {
  case string, bool,
      int, int8, int16, int32, int64,
      uint, uint8, uint16, uint32, uint64,
      float32, float64,
      regexp.Regexp:
    return true
  }
  return false
}

var scalarType_DEFAULT = &ScalarType{}




func (t *ScalarDataType) isAssignable(o PuppetType, guard map[visit]bool) bool {
  switch o.(type) {
  case *ScalarDataType:
    return true
  default:
    return isAssignable(stringType_DEFAULT, o, guard) ||
        isAssignable(integerType_DEFAULT, o, guard) ||
        isAssignable(booleanType_DEFAULT, o, guard) ||
        isAssignable(floatType_DEFAULT, o, guard)
  }
}

func (t *ScalarDataType) isInstance(o interface{}, guard map[visit]bool) bool {
  switch o.(type) {
  case string, bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
    return true
  }
  return false
}

var scalarDataType_DEFAULT = &ScalarDataType{}




func (t *StringType) isAssignable(o PuppetType, guard map[visit]bool) bool {
  if st, ok := o.(*StringType); ok {
    if t.value == `` {
      return t.size.isAssignable(st.size, guard)
    }
    return t.value == st.value
  }

  if et, ok := o.(*EnumType); ok {
    if t.value == `` {
      if *t.size == *integerType_POSITIVE {
        return true
      }
      for _, str := range et.values {
        if !t.size.isInstance(str, guard) {
          return false
        }
      }
      return true
    }
  }

  if _, ok := o.(*PatternType); ok {
    // Pattern is only assignable to the default string
    return *t == *stringType_DEFAULT
  }
  return false
}

func (t *StringType) isInstance(o interface{}, guard map[visit]bool) bool {
  str, ok := o.(string)
  return ok && t.size.isInstance(len(str), guard) && (t.value == `` || t.value == str)
}

var stringType_DEFAULT = &StringType{integerType_POSITIVE, ``}


func (s *StructElement) ActualKeyType() PuppetType {
  if ot, ok := s.key.(*OptionalType); ok {
    return ot.typ
  }
  return s.key
}


func (t *StructType) HashedMembers() map[string]*StructElement {
  // TODO this is not thread safe. Add mutex
  if t.hashedMembers == nil {
    t.hashedMembers = make(map[string]*StructElement, len(t.elements))
    for _, elem := range t.elements {
      t.hashedMembers[elem.name] = elem
    }
  }
  return t.hashedMembers
}

func (t *StructType) isAssignable(o PuppetType, guard map[visit]bool) bool {
  switch o.(type) {
  case *StructType:
    st := o.(*StructType)
    hm := st.HashedMembers()
    matched := 0
    for _, e1 := range t.elements {
      e2 := hm[e1.name]
      if e2 == nil {
        if !isAssignable(e1.key, undefType_DEFAULT, guard) {
          return false
        }
      } else {
        if !isAssignable(e1.key, e2.key, guard) && isAssignable(e1.value, e2.value, guard) {
          return false
        }
        matched++
      }
    }
    return matched == len(hm)
  case *HashType:
    ht := o.(*HashType)
    required := 0
    for _, e := range t.elements {
      if !isAssignable(e.key, undefType_DEFAULT, guard) {
        if !isAssignable(e.value, ht.valueType, guard) {
          return false
        }
        required++
      }
    }
    if required > 0 && !isAssignable(stringType_DEFAULT, ht.keyType, guard) {
      return false
    }
    return isAssignable(NewIntegerType(int64(required), int64(len(t.elements))), ht.size, guard)
  default:
    return false
  }
}

func (t *StructType) isInstance(o interface{}, guard map[visit]bool) bool {
  ov := reflect.ValueOf(o)
  if !(ov.Kind() == reflect.Map && ov.Type().Key().Kind() == reflect.String) {
    return false
  }
  matched := 0
  for _, element := range t.elements {
    key := reflect.ValueOf(element.name)
    v := ov.MapIndex(key)
    if !v.IsValid() {
      if !isAssignable(element.key, undefType_DEFAULT, guard) {
        return false
      }
    } else {
      if !isInstance(element.value, v.Interface(), guard) {
        return false
      }
      matched++
    }
  }
  return matched == ov.Len()
}

var structType_DEFAULT = &StructType{integerType_POSITIVE, []*StructElement{}, nil}

func NewEmptyTupleType() PuppetType {
  return tupleType_EMPTY
}

func (t *TupleType) isAssignable(o PuppetType, guard map[visit]bool) bool {
  switch o.(type) {
  case *ArrayType:
    at := o.(*ArrayType)
    if !isAssignable(t.givenOrActualSize, at.size, guard) {
      return false
    }
    top := len(t.types)
    if top == 0 {
      return true
    }
    elemType := at.typ
    for idx := 0; idx < top; idx++ {
      if !isAssignable(t.types[idx], elemType, guard) {
        return false
      }
    }
    return true

  case *TupleType:
    tt := o.(*TupleType)
    if !isAssignable(t.givenOrActualSize, tt.givenOrActualSize, guard) {
      return false
    }
    top := len(tt.types)
    if top == 0 {
      return t.givenOrActualSize.min == 0
    }

    last := len(t.types) - 1
    for idx := 0; idx < top; idx++ {
      myIdx := idx
      if myIdx > last {
        myIdx = last
      }
      if !isAssignable(t.types[myIdx], tt.types[idx], guard) {
        return false
      }
    }
    return true

  default:
    return false
  }
}

func (t *TupleType) isInstance(o interface{}, guard map[visit]bool) bool {
  v := reflect.ValueOf(o)
  if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
    return false
  }

  osz := v.Len()
  if !isInstance(t.givenOrActualSize, osz, guard) {
    return false
  }

  last := len(t.types) - 1
  if last < 0 {
    return true
  }

  tdx := 0
  for idx := 0; idx < osz; idx++ {
    if !isInstance(t.types[tdx], v.Index(idx).Interface(), guard) {
      return false
    }
    if tdx < last {
      tdx++
    }
  }
  return true
}

func (t *TupleType) Size() *IntegerType {
  return t.size
}

var tupleType_DEFAULT = &TupleType{integerType_POSITIVE, integerType_POSITIVE, true, []PuppetType{}}
var tupleType_EMPTY = &TupleType{integerType_ZERO, integerType_ZERO, true, []PuppetType{}}




func (t *TypeAliasType) isAssignable(o PuppetType, guard map[visit]bool) bool {
  return isAssignable(t.ResolvedType(), o, guard)
}

func (t *TypeAliasType) isInstance(o interface{}, guard map[visit]bool) bool {
  return isInstance(t.ResolvedType(), o, guard)
}

func (t *TypeAliasType) ResolvedType() PuppetType {
  if t.resolvedType == nil {
    panic(fmt.Sprintf("Reference to unresolved type '%s'", t.name))
  }
  return t.resolvedType
}

var typeAliasType_DEFAULT = &TypeAliasType{`UnresolvedAlias`, nil, defaultType_DEFAULT, false, nil}







func (t *TypeReferenceType) isAssignable(o PuppetType, guard map[visit]bool) bool {
  tr, ok := o.(*TypeReferenceType)
  return ok && t.typeString == tr.typeString
}

func (t *TypeReferenceType) isInstance(o interface{}, guard map[visit]bool) bool {
  return false
}

var typeReferenceType_DEFAULT = &TypeReferenceType{`UnresolvedReference`}




func (t *TypeType) isAssignable(o PuppetType, guard map[visit]bool) bool {
  if ot, ok := o.(*TypeType); ok {
    return isAssignable(t.typ, ot.typ, guard)
  }
  return false
}

func (t *TypeType) isInstance(o interface{}, guard map[visit]bool) bool {
  if ot, ok := o.(PuppetType); ok {
    return isAssignable(t.typ, ot, guard)
  }
  return false
}

var typeType_DEFAULT = &TypeType{typ:anyType_DEFAULT}


func (t *UndefType) isAssignable(o PuppetType, guard map[visit]bool) bool {
  return true
}

func (t *UndefType) isInstance(o interface{}, guard map[visit]bool) bool {
  return true
}

var undefType_DEFAULT = &UndefType{}



func (t *UnitType) isAssignable(o PuppetType, guard map[visit]bool) (ok bool) {
  _, ok = o.(*UnitType)
  return
}

func (t *UnitType) isInstance(o interface{}, guard map[visit]bool) bool {
  return o == nil
}

var unitType_DEFAULT = &UnitType{}


func (t *VariantType) isAssignable(o PuppetType, guard map[visit]bool) bool {
  for _, v := range t.types {
    if isAssignable(v, o, guard) {
      return true
    }
  }
  return false
}

func (t *VariantType) isInstance(o interface{}, guard map[visit]bool) bool {
  for _, v := range t.types {
    if isInstance(v, o, guard) {
      return true
    }
  }
  return false
}

func (t *VariantType) allAssignableTo(o PuppetType, guard map[visit]bool) bool {
  return allAssignableTo(t.types, o, guard)
}

var variantType_DEFAULT = &VariantType{types:[]PuppetType{}, resolved:true}


func allAssignableTo(types []PuppetType, o PuppetType, guard map[visit]bool) bool {
  for _, v := range types {
    if !isAssignable(o, v, guard) {
      return false
    }
  }
  return true
}
