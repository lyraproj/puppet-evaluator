package types

import (
	"io"
	"strconv"

	"github.com/puppetlabs/go-evaluator/eval"
)

type CallableType struct {
	paramsType *TupleType
	returnType eval.PType
	blockType  eval.PType // Callable or Optional[Callable]
}

var Callable_Type eval.ObjectType

func init() {
	Callable_Type = newObjectType(`Pcore::CallableType`,
		`Pcore::AnyType {
  attributes => {
    param_types => {
      type => Optional[Type[Tuple]],
      value => undef
    },
    block_type => {
      type => Optional[Type[Callable]],
      value => undef
    },
    return_type => {
      type => Optional[Type],
      value => undef
    }
  }
}`, func(ctx eval.Context, args []eval.PValue) eval.PValue {
			return NewCallableType2(args...)
		})
}

func DefaultCallableType() *CallableType {
	return callableType_DEFAULT
}

func NewCallableType(paramsType *TupleType, returnType eval.PType, blockType eval.PType) *CallableType {
	return &CallableType{paramsType, returnType, blockType}
}

func NewCallableType2(args ...eval.PValue) *CallableType {
	return NewCallableType3(WrapArray(args))
}

func NewCallableType3(args eval.IndexedValue) *CallableType {
	argc := args.Len()
	if argc == 0 {
		return DefaultCallableType()
	}

	var (
		rt    eval.PType
		block eval.PType
		ok    bool
	)

	if argc == 2 {
		// check for [[params, block], return]
		var iv eval.IndexedValue
		if iv, ok = args.At(0).(eval.IndexedValue); ok {
			argc = iv.Len()
			if argc < 0 || argc > 2 {
				panic(NewIllegalArgumentType2(`Callable[]`, 0, `Tuple[Type[Tuple], Type[CallableType, 1, 2]]]`, args.At(0)))
			}

			if rt, ok = args.At(1).(eval.PType); !ok {
				panic(NewIllegalArgumentType2(`Callable[]`, 1, `Type`, args.At(1)))
			}

			args = iv
		}
	}

	last := args.At(argc - 1)
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
		args = args.Slice(0, argc)
		last = args.At(argc)
	}
	return NewCallableType(tupleFromArgs(true, args), rt, block)
}

func (t *CallableType) BlockType() eval.PType {
	if t.blockType == nil {
		return nil // Return untyped nil
	}
	return t.blockType
}

func (t *CallableType) CallableWith(args []eval.PValue, block eval.Lambda) bool {
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
	return t.paramsType.IsInstance3(args, nil)
}

func (t *CallableType) Accept(v eval.Visitor, g eval.Guard) {
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

func (t *CallableType) Default() eval.PType {
	return callableType_DEFAULT
}

func (t *CallableType) Equals(o interface{}, g eval.Guard) bool {
	_, ok := o.(*CallableType)
	return ok
}

func (t *CallableType) Generic() eval.PType {
	return callableType_DEFAULT
}

func (t *CallableType) Get(key string) (eval.PValue, bool) {
	switch key {
	case `param_types`:
		if t.paramsType == nil {
			return eval.UNDEF, true
		}
		return t.paramsType, true
	case `return_type`:
		if t.returnType == nil {
			return eval.UNDEF, true
		}
		return t.returnType, true
	case `block_type`:
		if t.blockType == nil {
			return eval.UNDEF, true
		}
		return t.blockType, true
	default:
		return nil, false
	}
}

func (t *CallableType) IsAssignable(o eval.PType, g eval.Guard) bool {
	oc, ok := o.(*CallableType)
	if !ok {
		return false
	}
	if t.returnType == nil && t.paramsType == nil && t.blockType == nil {
		return true
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

func (t *CallableType) IsInstance(o eval.PValue, g eval.Guard) bool {
	if l, ok := o.(eval.Lambda); ok {
		return isAssignable(t, l.Type())
	}
	// TODO: Maybe check Go func using reflection
	return false
}

func (t *CallableType) MetaType() eval.ObjectType {
	return Callable_Type
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

func (t *CallableType) Parameters() (params []eval.PValue) {
	if *t == *callableType_DEFAULT {
		return eval.EMPTY_VALUES
	}
	tupleParams := t.paramsType.Parameters()
	if len(tupleParams) == 0 {
		params = []eval.PValue{ZERO, ZERO}
	} else {
		params = eval.Select1(tupleParams, func(p eval.PValue) bool { _, ok := p.(*UnitType); return !ok })
	}
	if t.blockType != nil {
		params = append(params, t.blockType)
	}
	if t.returnType != nil {
		params = []eval.PValue{WrapArray(params), t.returnType}
	}
	return params
}

func (t *CallableType) ParametersType() eval.PType {
	if t.paramsType == nil {
		return nil // Return untyped nil
	}
	return t.paramsType
}

func (t *CallableType) Resolve(c eval.Context) eval.PType {
	if t.paramsType != nil {
		t.paramsType = resolve(c, t.paramsType).(*TupleType)
	}
	if t.returnType != nil {
		t.returnType = resolve(c, t.returnType)
	}
	if t.blockType != nil {
		t.blockType = resolve(c, t.blockType)
	}
	return t
}

func (t *CallableType) ReturnType() eval.PType {
	return t.returnType
}

func (t *CallableType) String() string {
	return eval.ToString2(t, NONE)
}

func (t *CallableType) Type() eval.PType {
	return &TypeType{t}
}

func (t *CallableType) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	TypeToString(t, b, s, g)
}

var callableType_DEFAULT = &CallableType{paramsType: nil, blockType: nil, returnType: nil}
