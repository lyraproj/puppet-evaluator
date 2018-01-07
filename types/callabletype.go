package types

import (
	. "io"

	. "github.com/puppetlabs/go-evaluator/evaluator"
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
func (c *CallableType) BlockType() PType {
	return c.blockType
}

func (c *CallableType) CallableWith(args []PValue, block Lambda) bool {
	if block != nil {
		cb := c.blockType
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
	} else if c.blockType != nil && !isAssignable(c.blockType, anyType_DEFAULT) {
		// Required block but non provided
		return false
	}
	return c.paramsType.IsInstance2(args, nil)
}

func (t *CallableType) Default() PType {
	return callableType_DEFAULT
}

func (c *CallableType) Equals(o interface{}, g Guard) bool {
	_, ok := o.(*CallableType)
	return ok
}

func (c *CallableType) Generic() PType {
	return collectionType_DEFAULT
}

func (c *CallableType) IsAssignable(o PType, g Guard) bool {
	oc, ok := o.(*CallableType)
	if !ok {
		return ok
	}
	if c.returnType != nil {
		or := oc.returnType
		if or == nil {
			or = anyType_DEFAULT
		}
		if !isAssignable(c.returnType, or) {
			return false
		}
	}

	// NOTE: these tests are made in reverse as it is calling the callable that is constrained
	// (it's lower bound), not its upper bound
	if !(oc.paramsType == nil || isAssignable(oc.paramsType, c.paramsType)) {
		return false
	}
	if c.blockType == nil {
		if oc.blockType != nil {
			return false
		}
		return true
	}
	if oc.blockType == nil {
		return false
	}
	return isAssignable(oc.blockType, c.blockType)
}

func (c *CallableType) IsInstance(o PValue, g Guard) bool {
	if l, ok := o.(Lambda); ok {
		return isAssignable(c, l.Type())
	}
	// TODO: Maybe check Go func using reflection
	return false
}

func (c *CallableType) Name() string {
	return `Callable`
}

func (c *CallableType) Parameters() (params []PValue) {
	if *c == *callableType_DEFAULT {
		return EMPTY_VALUES
	}
	tupleParams := c.paramsType.Parameters()
	if len(tupleParams) == 0 {
		params = []PValue{ZERO, ZERO}
	} else {
		params = Select(tupleParams, func(p PValue) bool { _, ok := p.(*UnitType); return !ok })
	}
	if c.blockType != nil {
		params = append(params, c.blockType)
	}
	if c.returnType != nil {
		params = []PValue{WrapArray(params), c.returnType}
	}
	return params
}

func (c *CallableType) ParametersType() PType {
	return c.paramsType
}

func (c *CallableType) ReturnType() PType {
	return c.returnType
}

func (c *CallableType) String() string {
	return ToString2(c, NONE)
}

func (c *CallableType) Type() PType {
	return &TypeType{c}
}

func (c *CallableType) ToString(b Writer, s FormatContext, g RDetect) {
	TypeToString(c, b, s, g)
}

var callableType_DEFAULT = &CallableType{}
