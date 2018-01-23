package types

import (
	. "io"
	"math"

	. "github.com/puppetlabs/go-evaluator/evaluator"
)

type TupleType struct {
	size              *IntegerType
	givenOrActualSize *IntegerType
	types             []PType
}

func DefaultTupleType() *TupleType {
	return tupleType_DEFAULT
}

func EmptyTupleType() *TupleType {
	return tupleType_EMPTY
}

func NewTupleType(types []PType, size *IntegerType) *TupleType {
	var givenOrActualSize *IntegerType
	sz := int64(len(types))
	if size == nil {
		givenOrActualSize = NewIntegerType(sz, sz)
	} else {
		if sz == 0 {
			if *size == *integerType_POSITIVE {
				return DefaultTupleType()
			}
			if *size == *integerType_ZERO {
				return EmptyTupleType()
			}
		}
		givenOrActualSize = size
	}
	return &TupleType{size, givenOrActualSize, types}
}

func NewTupleType2(args ...PValue) *TupleType {
	return tupleFromArgs(false, args)
}

func tupleFromArgs(callable bool, args []PValue) *TupleType {
	argc := int64(len(args))
	if argc == 0 {
		return tupleType_DEFAULT
	}

	var rng, givenOrActualRng *IntegerType
	var ok bool
	var min int64

	last := args[argc-1]
	max := int64(-1)
	if _, ok = last.(*DefaultValue); ok {
		max = math.MaxInt64
	} else if n, ok := toInt(last); ok {
		max = n
	}
	if max >= 0 {
		if argc == 1 {
			rng = NewIntegerType(min, math.MaxInt64)
			argc = 0
		} else {
			if min, ok = toInt(args[argc-2]); ok {
				rng = NewIntegerType(min, max)
				argc -= 2
			} else {
				argc--
				rng = NewIntegerType(max, int64(argc))
			}
		}
		givenOrActualRng = rng
	} else {
		rng = nil
		givenOrActualRng = NewIntegerType(argc, argc)
	}

	if argc == 0 {
		if *rng == *integerType_ZERO {
			return tupleType_EMPTY
		}
		if callable {
			return &TupleType{rng, rng, []PType{DefaultUnitType()}}
		}
		if *rng == *integerType_POSITIVE {
			return tupleType_DEFAULT
		}
		return &TupleType{rng, rng, []PType{}}
	}

	var tupleTypes []PType
	ok = false
	failIdx := -1
	if argc == 1 {
		// One arg can be either array of types or a type
		tupleTypes, failIdx = toTypes(args[0])
		ok = failIdx < 0
	}

	if !ok {
		tupleTypes, failIdx = toTypes(args[0:argc]...)
		if failIdx >= 0 {
			name := `Tuple[]`
			if callable {
				name = `Callable[]`
			}
			panic(NewIllegalArgumentType2(name, failIdx, `Type`, args[failIdx]))
		}
	}
	return &TupleType{rng, givenOrActualRng, tupleTypes}
}

func (t *TupleType) Accept(v Visitor, g Guard) {
	v(t)
	t.size.Accept(v, g)
	for _, c := range t.types {
		c.Accept(v, g)
	}
}

func (t *TupleType) CommonElementType() PType {
	top := len(t.types)
	if top == 0 {
		return anyType_DEFAULT
	}
	cet := t.types[0]
	for idx := 1; idx < top; idx++ {
		cet = commonType(cet, t.types[idx])
	}
	return cet
}

func (t *TupleType) Default() PType {
	return tupleType_DEFAULT
}

func (t *TupleType) Equals(o interface{}, g Guard) bool {
	if ot, ok := o.(*TupleType); ok && len(t.types) == len(ot.types) && GuardedEquals(t.size, ot.size, g) {
		for idx, col := range t.types {
			if !col.Equals(ot.types[idx], g) {
				return false
			}
		}
		return true
	}
	return false
}

func (t *TupleType) Generic() PType {
	return NewTupleType(alterTypes(t.types, generalize), t.size)
}

func (t *TupleType) IsAssignable(o PType, g Guard) bool {
	switch o.(type) {
	case *ArrayType:
		at := o.(*ArrayType)
		if !GuardedIsInstance(t.givenOrActualSize, WrapInteger(at.size.Min()), g) {
			return false
		}
		top := len(t.types)
		if top == 0 {
			return true
		}
		elemType := at.typ
		for idx := 0; idx < top; idx++ {
			if !GuardedIsAssignable(t.types[idx], elemType, g) {
				return false
			}
		}
		return true

	case *TupleType:
		tt := o.(*TupleType)
		if !GuardedIsInstance(t.givenOrActualSize, WrapInteger(tt.givenOrActualSize.Min()), g) {
			return false
		}

		if len(t.types) > 0 {
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
				if !GuardedIsAssignable(t.types[myIdx], tt.types[idx], g) {
					return false
				}
			}
		}
		return true

	default:
		return false
	}
}

func (t *TupleType) IsInstance(v PValue, g Guard) bool {
	if iv, ok := v.(*ArrayValue); ok {
		return t.IsInstance2(iv.Elements(), g)
	}
	return false
}

func (t *TupleType) IsInstance2(vs []PValue, g Guard) bool {
	osz := len(vs)
	if !t.givenOrActualSize.IsInstance3(osz) {
		return false
	}

	last := len(t.types) - 1
	if last < 0 {
		return true
	}

	tdx := 0
	for idx := 0; idx < osz; idx++ {
		if !GuardedIsInstance(t.types[tdx], vs[idx], g) {
			return false
		}
		if tdx < last {
			tdx++
		}
	}
	return true
}

func (t *TupleType) Name() string {
	return `Tuple`
}

func (t *TupleType) Size() *IntegerType {
	return t.givenOrActualSize
}

func (t *TupleType) String() string {
	return ToString2(t, NONE)
}

func (t *TupleType) Parameters() []PValue {
	top := len(t.types)
	params := make([]PValue, 0, top+2)
	for _, c := range t.types {
		params = append(params, c)
	}
	if !(t.size == nil || top == 0 && *t.size == *integerType_POSITIVE) {
		params = append(params, t.size.SizeParameters()...)
	}
	return params
}

func (t *TupleType) ToString(b Writer, s FormatContext, g RDetect) {
	TypeToString(t, b, s, g)
}

func (t *TupleType) Type() PType {
	return &TypeType{t}
}

func (t *TupleType) Types() []PType {
	return t.types
}

var tupleType_DEFAULT = &TupleType{integerType_POSITIVE, integerType_POSITIVE, []PType{}}
var tupleType_EMPTY = &TupleType{integerType_ZERO, integerType_ZERO, []PType{}}
