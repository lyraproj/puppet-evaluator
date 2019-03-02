package impl

import (
	"reflect"
	"testing"

	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/types"
)

func toType(pt interface{}) eval.Type {
	return pt.(eval.Type)
}

func TestIsAssignable(t *testing.T) {
	t1 := &types.AnyType{}
	t2 := &types.UnitType{}
	if !eval.IsAssignable(t1, t2) {
		t.Error(`Unit not assignable to Any`)
	}
}

func TestIdentitySetZeroSizeSameType(t *testing.T) {
	x := &types.AnyType{}
	if1 := toType(x)
	if2 := toType(x)

	if if1 != if2 {
		t.Error(`Interfaces are not equal`)
	}

	idMap := IdentitySet{}
	idMap.Add(if1)
	if idMap.Add(if2) {
		t.Error(`Value with the same identity was added twice`)
	}
}

func TestIdentitySetZeroSizeDifferentType(t *testing.T) {
	x := &types.AnyType{}
	y := &types.UndefType{}
	if1 := toType(x)
	if2 := toType(y)
	if if1 == if2 {
		t.Error(`Interfaces are equal`)
	}

	idMap := IdentitySet{}
	idMap.Add(if1)
	if idMap.Include(if2) {
		t.Error(`Found entry using equal but different identity`)
	}
}

func TestIdentitySameInstance(t *testing.T) {
	x := types.DefaultTypeType()
	if1 := toType(x)
	if2 := toType(x)

	if if1 != if2 {
		t.Error(`Interfaces are not equal`)
	}

	idMap := IdentitySet{}
	idMap.Add(if1)
	if idMap.Add(if2) {
		t.Error(`Value with the same identity was added twice`)
	}
}

func TestIdentitySameNil(t *testing.T) {
	idMap := IdentitySet{}
	idMap.Add(nil)
	if idMap.Add(nil) {
		t.Error(`Value with the same identity was added twice`)
	}
}

func TestIdentitySameIfdNil(t *testing.T) {
	var if1 interface{}
	var if2 interface{}
	if1 = 32
	if2 = if1

	if if1 != if2 {
		t.Error(`interfaces are not equal`)
	}

	idMap := IdentitySet{}
	idMap.Add(if1)
	if idMap.Add(if2) {
		t.Error(`Value with the same identity was added twice`)
	}
}

func TestIdentityDifferentButEqual(t *testing.T) {
	if1 := types.NewTypeType(types.NewIntegerType(1, 3))
	if2 := types.NewTypeType(types.NewIntegerType(1, 3))

	if !reflect.DeepEqual(if1, if2) {
		t.Error(`Interfaces are not equal`)
	}

	idMap := IdentitySet{}
	idMap.Add(if1)
	if idMap.Include(if2) {
		t.Error(`Found entry using equal but different identity`)
	}
}
