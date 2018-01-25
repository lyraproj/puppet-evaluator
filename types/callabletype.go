package types

import (
	. "io"

	. "github.com/puppetlabs/go-evaluator/evaluator"
	"strconv"
)

type CallableType struct {
	paramsType *TupleType
	returnType PType
	blockType  PType // Callable or Optional[Callable]
}

func DefaultCallableType() *CallableType {
	return callableType_DEFAULT
}

func NewCallableType(paramsType *TupleType, returnType PType, blockType PType) *CallableType {
	return &CallableType{paramsType, returnType, blockType}
}

func NewCallableType2(args ...PValue) *CallableType {
	argc := len(args)
	if argc == 0 {
		return DefaultCallableType()
	}

	var (
		rt    PType
		block PType
		ok    bool
	)

	if argc == 2 {
		// check for [[params, block], return]
		var iv IndexedValue
		if iv, ok = args[0].(IndexedValue); ok {
			argc = iv.Len()
			if argc < 0 || argc > 2 {
				panic(NewIllegalArgumentType2(`Callable[]`, 0, `Tuple[Type[Tuple], Type[CallableType, 1, 2]]]`, args[0]))
			}

			if rt, ok = args[1].(PType); !ok {
				panic(NewIllegalArgumentType2(`Callable[]`, 1, `Type`, args[1]))
			}

			args = iv.Elements()
		}
	}

	last := args[argc-1]
	block, ok = last.(*CallableType)
	if !ok {
		block = nil
		var ob *OptionalType
		if ob, ok = last.(*OptionalType); ok {
			if _, ok = ob.typ.(*CallableType); ok {
				block = ob
			}
		}
	}
	if ok {
		argc--
		args = args[0:argc]
		last = args[argc-1]
	}
	return NewCallableType(tupleFromArgs(true, args), rt, block)
}
func (t *CallableType) BlockType() PType {
	if t.blockType == nil {
		return nil // Return untyped nil
	}
	return t.blockType
}

func (t *CallableType) CallableWith(args []PValue, block Lambda) bool {
	if block != nil {
		cb := t.blockType
		switch cb.(type) {
		case nil:
			return false
		case *OptionalType:
			cb = cb.(*OptionalType).ContainedType()
		}
		if block.Type() == nil {
			return false
		}
		if !isAssignable(block.Type(), cb) {
			return false
		}
	} else if t.blockType != nil && !isAssignable(t.blockType, anyType_DEFAULT) {
		// Required block but non provided
		return false
	}
	return t.paramsType.IsInstance2(args, nil)
}

func (t *CallableType) Accept(v Visitor, g Guard) {
	v(t)
	if t.paramsType != nil {
		t.paramsType.Accept(v, g)
	}
	if t.blockType != nil {
		t.blockType.Accept(v, g)
	}
	if t.returnType != nil {
		t.returnType.Accept(v, g)
	}
}

func (t *CallableType) BlockName() string {
	return `block`
}

func (t *CallableType) Default() PType {
	return callableType_DEFAULT
}

func (t *CallableType) Equals(o interface{}, g Guard) bool {
	_, ok := o.(*CallableType)
	return ok
}

func (t *CallableType) Generic() PType {
	return callableType_DEFAULT
}

func (t *CallableType) IsAssignable(o PType, g Guard) bool {
	oc, ok := o.(*CallableType)
	if !ok {
		return false
	}
	if t.returnType != nil {
		or := oc.returnType
		if or == nil {
			or = anyType_DEFAULT
		}
		if !isAssignable(t.returnType, or) {
			return false
		}
	}

	// NOTE: these tests are made in reverse as it is calling the callable that is constrained
	// (it's lower bound), not its upper bound
	if oc.paramsType != nil && (t.paramsType == nil || !isAssignable(oc.paramsType, t.paramsType)) {
		return false
	}

	if t.blockType == nil {
		if oc.blockType != nil {
			return false
		}
		return true
	}
	if oc.blockType == nil {
		return false
	}
	return isAssignable(oc.blockType, t.blockType)
}

func (t *CallableType) IsInstance(o PValue, g Guard) bool {
	if l, ok := o.(Lambda); ok {
		return isAssignable(t, l.Type())
	}
	// TODO: Maybe check Go func using reflection
	return false
}

func (t *CallableType) Name() string {
	return `Callable`
}

func (t *CallableType) ParameterNames() []string {
	n := len(t.paramsType.types)
	r := make([]string, 0, n)
	for i := 0; i < n; {
		i++
		r = append(r, strconv.Itoa(i))
	}
	return r
}

func (t *CallableType) Parameters() (params []PValue) {
	if *t == *callableType_DEFAULT {
		return EMPTY_VALUES
	}
	tupleParams := t.paramsType.Parameters()
	if len(tupleParams) == 0 {
		params = []PValue{ZERO, ZERO}
	} else {
		params = Select1(tupleParams, func(p PValue) bool { _, ok := p.(*UnitType); return !ok })
	}
	if t.blockType != nil {
		params = append(params, t.blockType)
	}
	if t.returnType != nil {
		params = []PValue{WrapArray(params), t.returnType}
	}
	return params
}

func (t *CallableType) ParametersType() PType {
	if t.paramsType == nil {
		return nil // Return untyped nil
	}
	return t.paramsType
}

func (t *CallableType) ReturnType() PType {
	return t.returnType
}

func (t *CallableType) String() string {
	return ToString2(t, NONE)
}

func (t *CallableType) Type() PType {
	return &TypeType{t}
}

func (t *CallableType) ToString(b Writer, s FormatContext, g RDetect) {
	TypeToString(t, b, s, g)
}

var callableType_DEFAULT = &CallableType{paramsType:nil,blockType:nil,returnType:nil}
