package types

import (
  "testing"
  "github.com/puppetlabs/go-evaluator/utils"
  "reflect"
)

func toType(pt interface{}) PuppetType {
  return pt.(PuppetType)
}

func TestMisc(t *testing.T) {
  var (
    i8 interface{}
    i16 interface{}
    i32 interface{}
    i64 interface{}
  )
  i8 = int8(2)
  i16 = int16(2)
  i32 = int32(2)
  i64 = int64(2)

  var (
    i int64
    ok bool
  )

  if i, ok = i8.(int64); !ok {
    t.Errorf("Unable to convert %T to int64", i8)
  } else {
    t.Logf("Successfully converted %T to int64 %d", i8, i)
  }

  if i, ok = i16.(int64); !ok {
    t.Errorf("Unable to convert %T to int64", i16)
  } else {
    t.Logf("Successfully converted %T to int64 %d", i16, i)
  }

  if i, ok = i32.(int64); !ok {
    t.Errorf("Unable to convert %T to int64", i32)
  } else {
    t.Logf("Successfully converted %T to int64 %d", i32, i)
  }

  if i, ok = i64.(int64); !ok {
    t.Errorf("Unable to convert %T to int64", i64)
  } else {
    t.Logf("Successfully converted %T to int64 %d", i64, i)
  }
}

func TestAssignability(t *testing.T) {
  t1 := &AnyType{}
  t2 := &UnitType{}
  if !IsAssignable(t1, t2) {
    t.Error(`Unit not assignable to Any`)
  }
}

func TestIdentitySetZeroSizeSameType(t *testing.T) {
  x := &AnyType{}
  if1 := toType(x)
  if2 := toType(x)

  if if1 != if2 {
    t.Error(`Interfaces are not equal`)
  }

  idMap := utils.IdentitySet{}
  idMap.Add(if1)
  if idMap.Add(if2) {
    t.Error(`Value with the same identity was added twice`)
  }
}

func TestIdentitySetZeroSizeDifferentType(t *testing.T) {
  x := &AnyType{}
  y := &UndefType{}
  if1 := toType(x)
  if2 := toType(y)
  if if1 == if2 {
    t.Error(`Interfaces are equal`)
  }

  idMap := utils.IdentitySet{}
  idMap.Add(if1)
  if idMap.Include(if2) {
    t.Error(`Found entry using equal but different identity`)
  }
}

func TestIdentitySameInstance(t *testing.T) {
  x := NewTypeType(nil)
  if1 := toType(x)
  if2 := toType(x)

  if if1 != if2 {
    t.Error(`Interfaces are not equal`)
  }

  idMap := utils.IdentitySet{}
  idMap.Add(if1)
  if idMap.Add(if2) {
    t.Error(`Value with the same identity was added twice`)
  }
}

func TestIdentityDifferentButEqual(t *testing.T) {
  if1 := NewTypeType(nil)
  if2 := NewTypeType(nil)

  if !reflect.DeepEqual(if1, if2) {
    t.Error(`Interfaces are not equal`)
  }

  idMap := utils.IdentitySet{}
  idMap.Add(if1)
  if idMap.Include(if2) {
    t.Error(`Found entry using equal but different identity`)
  }
}

