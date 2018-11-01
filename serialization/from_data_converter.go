package serialization

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/hash"
	"github.com/puppetlabs/go-evaluator/types"
	"github.com/puppetlabs/go-issues/issue"
	"github.com/puppetlabs/go-parser/parser"
)

type (
	richDataFunc func(hash eval.OrderedMap, typeValue eval.Value) eval.Value

	FromDataConverter struct {
		root            builder
		current         builder
		key             eval.Value
		allowUnresolved bool
		context         eval.Context
		richDataFuncs   map[string]richDataFunc
		defaultFunc     richDataFunc
	}

	builder interface {
		get(key eval.Value) builder
		put(key eval.Value, value builder)
		resolve(c eval.Context) eval.Value
	}

	valueBuilder struct {
		value eval.Value
	}

	hbEntry struct {
		key   eval.Value
		value builder
	}

	hashBuilder struct {
		values   *hash.StringHash
		resolved eval.Value
	}

	objectHashBuilder struct {
		hashBuilder
		object eval.Object
	}

	arrayBuilder struct {
		// *StringHash, []interface{}, or interface{}
		values   []builder
		resolved eval.Value
	}
)

func (b *valueBuilder) get(key eval.Value) builder {
	panic(`scalar indexed by string`)
}

func (b *valueBuilder) put(key eval.Value, value builder) {
	panic(`scalar indexed by string`)
}

func (b *valueBuilder) resolve(c eval.Context) eval.Value {
	return b.value
}

func (b *hashBuilder) get(key eval.Value) builder {
	if v, ok := b.values.Get(string(eval.ToKey(key)), nil).(*hbEntry); ok {
		return v.value
	}
	return nil
}

func (b *hashBuilder) put(key eval.Value, value builder) {
	b.values.Put(string(eval.ToKey(key)), &hbEntry{key, value})
}

func (b *hashBuilder) resolve(c eval.Context) eval.Value {
	if b.resolved == nil {
		es := make([]*types.HashEntry, 0, b.values.Len())
		b.values.EachPair(func(key string, value interface{}) {
			hbe := value.(*hbEntry)
			es = append(es, types.WrapHashEntry(hbe.key, hbe.value.resolve(c)))
		})
		b.resolved = types.WrapHash(es)
	}
	return b.resolved
}

func (b *arrayBuilder) get(key eval.Value) builder {
	if index, ok := eval.ToInt(key); ok && int(index) < len(b.values) {
		return b.values[index]
	}
	return nil
}

func (b *arrayBuilder) put(key eval.Value, value builder) {
	if index, ok := eval.ToInt(key); ok {
		// Allow growth of one. Beyond that will result in an index out of bounds panic
		if int(index) == len(b.values) {
			b.values = append(b.values, value)
		} else {
			b.values[index] = value
		}
	}
}

func (b *arrayBuilder) resolve(c eval.Context) eval.Value {
	if b.resolved == nil {
		es := make([]eval.Value, len(b.values))
		for i, v := range b.values {
			es[i] = v.resolve(c)
		}
		b.resolved = types.WrapArray(es)
	}
	return b.resolved
}

func (b *objectHashBuilder) resolve(c eval.Context) eval.Value {
	if b.resolved == nil {
		b.hashBuilder.resolve(c)
		b.object.InitFromHash(c, b.hashBuilder.resolve(c).(*types.HashValue))
		b.resolved = b.object
	}
	return b.resolved
}

func NewFromDataConverter(ctx eval.Context, options eval.OrderedMap) *FromDataConverter {
	f := &FromDataConverter{}
	f.context = ctx
	f.allowUnresolved = options.Get5(`allow_unresolved`, types.Boolean_FALSE).(*types.BooleanValue).Bool()
	f.richDataFuncs = map[string]richDataFunc{
		PCORE_TYPE_HASH: func(hash eval.OrderedMap, typeValue eval.Value) eval.Value {
			value := hash.Get5(PCORE_VALUE_KEY, eval.EMPTY_ARRAY).(eval.List)
			return f.buildHash(func() {
				top := value.Len()
				idx := 0
				for idx < top {
					key := f.withoutValue(func() eval.Value {
						return f.Convert(value.At(idx))
					})
					idx++
					f.with(key, func() {
						f.Convert(value.At(idx))
					})
					idx++
				}
			})
		},

		PCORE_TYPE_SENSITIVE: func(hash eval.OrderedMap, typeValue eval.Value) eval.Value {
			return f.buildValue(types.WrapSensitive(f.Convert(hash.Get5(PCORE_VALUE_KEY, eval.UNDEF))))
		},

		PCORE_TYPE_DEFAULT: func(hash eval.OrderedMap, typeValue eval.Value) eval.Value {
			return f.buildValue(types.WrapDefault())
		},

		PCORE_TYPE_SYMBOL: func(hash eval.OrderedMap, typeValue eval.Value) eval.Value {
			return f.buildValue(types.WrapRuntime(Symbol(hash.Get5(PCORE_VALUE_KEY, eval.EMPTY_STRING).String())))
		},

		PCORE_LOCAL_REF_SYMBOL: func(hash eval.OrderedMap, typeValue eval.Value) eval.Value {
			path := hash.Get5(PCORE_VALUE_KEY, eval.EMPTY_STRING).String()
			if resolved, ok := resolveJsonPath(f.context, f.root, path); ok {
				return f.buildValue(resolved)
			}
			panic(eval.Error(eval.EVAL_BAD_JSON_PATH, issue.H{path: path}))
		},
	}
	f.defaultFunc = func(hash eval.OrderedMap, typeValue eval.Value) eval.Value {
		value := hash.Get6(PCORE_VALUE_KEY, func() eval.Value {
			return hash.RejectPairs(func(k, v eval.Value) bool {
				if s, ok := k.(*types.StringValue); ok {
					return s.String() == PCORE_TYPE_KEY
				}
				return false
			})
		})
		if typeHash, ok := typeValue.(*types.HashValue); ok {
			typ := f.withoutValue(func() eval.Value { return f.Convert(typeHash) })
			if typ, ok := typeValue.(*types.HashValue); ok {
				if !f.allowUnresolved {
					panic(eval.Error(eval.EVAL_UNABLE_TO_DESERIALIZE_TYPE, issue.H{`hash`: typ.String()}))
				}
				return hash
			}
			return f.pcoreTypeHashToValue(typ.(eval.Type), value)
		}
		typ := f.context.ParseType(typeValue)
		if tr, ok := typ.(*types.TypeReferenceType); ok {
			if !f.allowUnresolved {
				panic(eval.Error(eval.EVAL_UNRESOLVED_TYPE, issue.H{`typeString`: tr.String()}))
			}
			return hash
		}
		return f.pcoreTypeHashToValue(typ.(eval.Type), value)
	}
	return f
}

func parseKeyword(lexer parser.Lexer) (eval.Value, bool) {
	t := lexer.CurrentToken()
	switch t {
	case parser.TOKEN_IDENTIFIER, parser.TOKEN_TYPE_NAME:
		s := lexer.TokenString()
		if s != `null` {
			return types.WrapString(lexer.TokenString()), true
		}
	default:
		if parser.IsKeywordToken(lexer.CurrentToken()) {
			return types.WrapString(lexer.TokenString()), true
		}
	}
	return nil, false
}

func parseStringOrInteger(lexer parser.Lexer) (eval.Value, bool) {
	t := lexer.CurrentToken()
	switch t {
	case parser.TOKEN_INTEGER:
		return types.WrapInteger(lexer.TokenValue().(int64)), true
	case parser.TOKEN_STRING:
		s := lexer.TokenString()
		if s != `null` {
			return types.WrapString(lexer.TokenString()), true
		}
	default:
		if parser.IsKeywordToken(lexer.CurrentToken()) {
			return types.WrapString(lexer.TokenString()), true
		}
	}
	return nil, false
}

func resolveJsonPath(c eval.Context, lhs builder, path string) (eval.Value, bool) {
	lexer := parser.NewSimpleLexer(``, path)
	lexer.NextToken()
	lexer.AssertToken(parser.TOKEN_VARIABLE)
	for {
		lexer.NextToken()
		switch lexer.CurrentToken() {
		case parser.TOKEN_DOT:
			lexer.NextToken()
			if key, ok := parseKeyword(lexer); ok {
				lhs = lhs.get(key)
				if lhs == nil {
					return nil, false
				}
				continue
			}
		case parser.TOKEN_LB:
			lexer.NextToken()
			if key, ok := parseStringOrInteger(lexer); ok {
				lexer.NextToken()
				lexer.AssertToken(parser.TOKEN_RB)
				lhs = lhs.get(key)
				if lhs == nil {
					return nil, false
				}
				continue
			}
		case parser.TOKEN_END:
			return lhs.resolve(c), true
		}
		return nil, false
	}
}

func (f *FromDataConverter) Convert(value eval.Value) eval.Value {
	if hash, ok := value.(*types.HashValue); ok {
		if pcoreType, ok := hash.Get4(PCORE_TYPE_KEY); ok {
			key := pcoreType.String()
			rdFunc, ok := f.richDataFuncs[key]
			if !ok {
				rdFunc = f.defaultFunc
			}
			return rdFunc(hash, pcoreType)
		}
		return f.buildHash(func() { hash.EachPair(func(k, v eval.Value) { f.with(k, func() { f.Convert(v) }) }) })
	}
	if array, ok := value.(*types.ArrayValue); ok {
		return f.buildArray(func() {
			array.EachWithIndex(func(v eval.Value, i int) { f.with(types.WrapInteger(int64(i)), func() { f.Convert(v) }) })
		})
	}
	return f.buildValue(value)
}

func (f *FromDataConverter) buildHash(doer eval.Doer) *types.HashValue {
	return f.build(&hashBuilder{hash.NewStringHash(31), nil}, doer).(*types.HashValue)
}

func (f *FromDataConverter) buildObject(object eval.Object, doer eval.Doer) eval.Value {
	return f.build(&objectHashBuilder{hashBuilder{hash.NewStringHash(31), nil}, object}, doer)
}

func (f *FromDataConverter) buildArray(doer eval.Doer) eval.Value {
	return f.build(&arrayBuilder{make([]builder, 0, 32), nil}, doer)
}

func (f *FromDataConverter) buildValue(value eval.Value) eval.Value {
	return f.build(&valueBuilder{value}, nil)
}

func (f *FromDataConverter) build(vx builder, doer eval.Doer) eval.Value {
	if f.current != nil {
		f.current.put(f.key, vx)
	}
	if doer != nil {
		f.withValue(vx, doer)
	}
	return vx.resolve(f.context)
}

func (f *FromDataConverter) with(key eval.Value, doer eval.Doer) {
	parentKey := f.key
	f.key = key
	doer()
	f.key = parentKey
}

func (f *FromDataConverter) withValue(value builder, doer eval.Doer) builder {
	if f.root == nil {
		f.root = value
	}
	parent := f.current
	f.current = value
	doer()
	f.current = parent
	return value
}

func (f *FromDataConverter) withoutValue(producer eval.Producer) eval.Value {
	parent := f.current
	f.current = nil
	value := producer()
	f.current = parent
	return value
}

func (f *FromDataConverter) pcoreTypeHashToValue(typ eval.Type, value eval.Value) eval.Value {
	if hash, ok := value.(*types.HashValue); ok {
		if ov, ok := f.allocate(typ); ok {
			return f.buildObject(ov, func() {
				hash.EachPair(func(key, elem eval.Value) { f.with(key, func() { f.Convert(elem) }) })
			})
		}
		hash = f.buildHash(func() {
			hash.EachPair(func(key, elem eval.Value) { f.with(key, func() { f.Convert(elem) }) })
		})
		if ot, ok := typ.(eval.ObjectType); ok {
			if ot.HasHashConstructor() {
				return f.buildValue(eval.New(f.context, typ, hash))
			}
			return f.buildValue(eval.New(f.context, typ, ot.AttributesInfo().PositionalFromHash(hash)...))
		}
		return f.buildValue(eval.New(f.context, typ, hash))
	}
	if str, ok := value.(*types.StringValue); ok {
		return f.buildValue(eval.New(f.context, typ, str))
	}
	panic(eval.Error(eval.EVAL_UNABLE_TO_DESERIALIZE_VALUE, issue.H{`type`: typ.Name(), `arg_type`: value.PType().Name()}))
}

func (f *FromDataConverter) allocate(typ eval.Type) (eval.Object, bool) {
	if allocator, ok := eval.Load(f.context, eval.NewTypedName(eval.ALLOCATOR, typ.Name())); ok {
		return allocator.(eval.Lambda).Call(nil, nil).(eval.Object), true
	}
	return nil, false
}
