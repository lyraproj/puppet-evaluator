package types

import (
	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/puppet-evaluator/eval"
	"io"
)

type InitType struct {
	typ      eval.Type
	initArgs *ArrayValue
	ctor     eval.Function
}

var InitMetaType eval.ObjectType

func init() {
	InitMetaType = newObjectType(`Pcore::Init`, `Pcore::AnyType {
	attributes => {
    type => { type => Optional[Type], value => undef },
		init_args => { type => Array, value => [] }
	}
}`, func(ctx eval.Context, args []eval.Value) eval.Value {
		return NewInitType2(args...)
	})
}

func DefaultInitType() *InitType {
	return initType_DEFAULT
}

func NewInitType(typ eval.Value, args eval.Value) *InitType {
	tp, ok := typ.(eval.Type)
	if !ok {
		tp = nil
	}
	aa, ok := args.(*ArrayValue)
	if !ok {
		aa = _EMPTY_ARRAY
	}
	if typ == nil && aa.Len() == 0 {
		return initType_DEFAULT
	}
	return &InitType{typ: tp, initArgs: aa}
}

func NewInitType2(args ...eval.Value) *InitType {
	switch len(args) {
	case 0:
		return DefaultInitType()
	case 1:
		return NewInitType(args[0], nil)
	default:
		return NewInitType(args[0], WrapValues(args[1:]))
	}
}

func (t *InitType) Accept(v eval.Visitor, g eval.Guard) {
	v(t)
	t.typ.Accept(v, g)
}

func (t *InitType) CanSerializeAsString() bool {
	return canSerializeAsString(t.typ)
}

func (t *InitType) SerializationString() string {
	return t.String()
}

func (t *InitType) Default() eval.Type {
	return initType_DEFAULT
}

func (t *InitType) Equals(o interface{}, g eval.Guard) bool {
	if ot, ok := o.(*InitType); ok {
		return eval.Equals(t.typ, ot.typ) && eval.Equals(t.initArgs, ot.initArgs)
	}
	return false
}

func (t *InitType) Get(key string) (eval.Value, bool) {
	switch key {
	case `type`:
		if t.typ == nil {
			return _UNDEF, true
		}
		return t.typ, true
	case `init_args`:
		return t.initArgs, true
	default:
		return nil, false
	}
}

func (t *InitType) EachSignature(doer func(signature eval.Signature)) {
	t.assertInitialized()
	if t.ctor != nil {
		for _, lambda := range t.ctor.Dispatchers() {
			doer(lambda.Signature())
		}
	}
}

func (t *InitType) anySignature(doer func(signature eval.Signature) bool) bool {
	t.assertInitialized()
	if t.ctor != nil {
		for _, lambda := range t.ctor.Dispatchers() {
			if doer(lambda.Signature()) {
				return true
			}
		}
	}
	return false
}

// IsAssignable answers the question if a value of the given type can be used when
// instantiating an instance of the contained type
func (t *InitType) IsAssignable(o eval.Type, g eval.Guard) bool {
	if t.typ == nil {
		return richDataType_DEFAULT.IsAssignable(o, g)
	}

	if !t.initArgs.IsEmpty() {
		ts := append(make([]eval.Type, 0, t.initArgs.Len()+1), o)
		t.initArgs.Each(func(v eval.Value) { ts = append(ts, v.PType()) })
		tp := NewTupleType(ts, nil)
		return t.anySignature(func(s eval.Signature) bool { return s.IsAssignable(tp, g) })
	}

	// First test if the given value matches a single value constructor
	tp := NewTupleType([]eval.Type{o}, nil)
	return t.anySignature(func(s eval.Signature) bool { return s.IsAssignable(tp, g) })

	// If the given value is tuple, check if it matches the signature tuple verbatim
	if tp, ok := o.(*TupleType); ok {
		return t.anySignature(func(s eval.Signature) bool { return s.IsAssignable(tp, g) })
	}
	return false
}

// IsInstance answers the question if the given value can be used when
// instantiating an instance of the contained type
func (t *InitType) IsInstance(o eval.Value, g eval.Guard) bool {
	if t.typ == nil {
		return richDataType_DEFAULT.IsInstance(o, g)
	}

	if !t.initArgs.IsEmpty() {
		// The init arguments must be combined with the given value in an array. Here, it doesn't
		// matter if the given value is an array or not. It must match as a single value regardless.
		vs := append(make([]eval.Value, 0, t.initArgs.Len()+1), o)
		vs = t.initArgs.AppendTo(vs)
		return t.anySignature(func(s eval.Signature) bool { return s.CallableWith(vs, nil) })
	}

	// First test if the given value matches a single value constructor.
	vs := []eval.Value{o}
	if t.anySignature(func(s eval.Signature) bool { return s.CallableWith(vs, nil) }) {
		return true
	}

	// If the given value is an array, expand it and check if it matches.
	if a, ok := o.(*ArrayValue); ok {
		vs = a.AppendTo(make([]eval.Value, 0, a.Len()))
		return t.anySignature(func(s eval.Signature) bool { return s.CallableWith(vs, nil) })
	}
	return false
}

func (t *InitType) MetaType() eval.ObjectType {
	return InitMetaType
}

func (t *InitType) Name() string {
	return `Init`
}

func (t *InitType) New(c eval.Context, args []eval.Value) eval.Value {
	t.Resolve(c)
	if t.ctor == nil {
		panic(eval.Error(eval.EVAL_INSTANCE_DOES_NOT_RESPOND, issue.H{`type`: t, `message`: `new`}))
	}

	if !t.initArgs.IsEmpty() {
		// The init arguments must be combined with the given value in an array. Here, it doesn't
		// matter if the given value is an array or not. It must match as a single value regardless.
		vs := append(make([]eval.Value, 0, t.initArgs.Len()+len(args)), args...)
		vs = t.initArgs.AppendTo(vs)
		return t.ctor.Call(c, nil, vs...)
	}

	// First test if the given value matches a single value constructor.
	if t.anySignature(func(s eval.Signature) bool { return s.CallableWith(args, nil) }) {
		return t.ctor.Call(c, nil, args...)
	}

	// If the given value is an array, expand it and check if it matches.
	if len(args) == 1 {
		arg := args[0]
		if a, ok := arg.(*ArrayValue); ok {
			vs := a.AppendTo(make([]eval.Value, 0, a.Len()))
			return t.ctor.Call(c, nil, vs...)
		}
	}

	// Provoke argument error
	return t.ctor.Call(c, nil, args...)
}

func (t *InitType) Resolve(c eval.Context) eval.Type {
	if t.typ != nil && t.ctor == nil {
		if ctor, ok := eval.Load(c, NewTypedName(eval.NsConstructor, t.typ.Name())); ok {
			t.ctor = ctor.(eval.Function)
		} else {
			panic(eval.Error(eval.EVAL_CTOR_NOT_FOUND, issue.H{`type`: t.typ.Name()}))
		}
	}
	return t
}

func (t *InitType) String() string {
	return eval.ToString2(t, NONE)
}

func (t *InitType) Parameters() []eval.Value {
	t.assertInitialized()
	if t.initArgs.Len() == 0 {
		if t.typ == nil {
			return eval.EMPTY_VALUES
		}
		return []eval.Value{t.typ}
	}
	ps := []eval.Value{_UNDEF, t.initArgs}
	if t.typ != nil {
		ps[1] = t.typ
	}
	return ps
}

func (t *InitType) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *InitType) Type() eval.Type {
	return t.typ
}

func (t *InitType) PType() eval.Type {
	return &TypeType{t}
}

func (t *InitType) assertInitialized() {
	if t.typ != nil && t.ctor == nil {
		t.Resolve(eval.CurrentContext())
	}
}

var initType_DEFAULT = &InitType{typ: nil, initArgs: _EMPTY_ARRAY}
