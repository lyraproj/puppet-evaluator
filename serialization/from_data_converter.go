package serialization

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/hash"
	"github.com/puppetlabs/go-evaluator/types"
	"github.com/puppetlabs/go-parser/issue"
	"github.com/puppetlabs/go-parser/parser"
)

type (
	richDataFunc func(hash eval.KeyedValue, typeValue eval.PValue) eval.PValue

	FromDataConverter struct {
		root            builder
		current         builder
		key             eval.PValue
		allowUnresolved bool
		loader          eval.DefiningLoader
		richDataFuncs   map[string]richDataFunc
		defaultFunc     richDataFunc
	}

	builder interface {
		get(key eval.PValue) builder
		put(key eval.PValue, value builder)
		resolve() eval.PValue
	}

	valueBuilder struct {
		value eval.PValue
	}

	hbEntry struct {
		key   eval.PValue
		value builder
	}

	hashBuilder struct {
		values   *hash.StringHash
		resolved eval.PValue
	}

	objectHashBuilder struct {
		hashBuilder
		object eval.ObjectValue
	}

	arrayBuilder struct {
		// *StringHash, []interface{}, or interface{}
		values   []builder
		resolved eval.PValue
	}
)

func (b *valueBuilder) get(key eval.PValue) builder {
	panic(`scalar indexed by string`)
}

func (b *valueBuilder) put(key eval.PValue, value builder) {
	panic(`scalar indexed by string`)
}

func (b *valueBuilder) resolve() eval.PValue {
	return b.value
}

func (b *hashBuilder) get(key eval.PValue) builder {
	if v, ok := b.values.Get(string(eval.ToKey(key)), nil).(*hbEntry); ok {
		return v.value
	}
	return nil
}

func (b *hashBuilder) put(key eval.PValue, value builder) {
	b.values.Put(string(eval.ToKey(key)), &hbEntry{key, value})
}

func (b *hashBuilder) resolve() eval.PValue {
	if b.resolved == nil {
		es := make([]*types.HashEntry, 0, b.values.Len())
		b.values.EachPair(func(key string, value interface{}) {
			hbe := value.(*hbEntry)
			es = append(es, types.WrapHashEntry(hbe.key, hbe.value.resolve()))
		})
		b.resolved = types.WrapHash(es)
	}
	return b.resolved
}

func (b *arrayBuilder) get(key eval.PValue) builder {
	if index, ok := eval.ToInt(key); ok && int(index) < len(b.values) {
		return b.values[index]
	}
	return nil
}

func (b *arrayBuilder) put(key eval.PValue, value builder) {
	if index, ok := eval.ToInt(key); ok {
		// Allow growth of one. Beyond that will result in an index out of bounds panic
		if int(index) == len(b.values) {
			b.values = append(b.values, value)
		} else {
			b.values[index] = value
		}
	}
}

func (b *arrayBuilder) resolve() eval.PValue {
	if b.resolved == nil {
		es := make([]eval.PValue, len(b.values))
		for i, v := range b.values {
			es[i] = v.resolve()
		}
		b.resolved = types.WrapArray(es)
	}
	return b.resolved
}

func (b *objectHashBuilder) resolve() eval.PValue {
	if b.resolved == nil {
		b.hashBuilder.resolve()
		b.object.InitFromHash(b.hashBuilder.resolve().(*types.HashValue))
		b.resolved = b.object
	}
	return b.resolved
}

func NewFromDataConverter(loader eval.DefiningLoader, options eval.KeyedValue) *FromDataConverter {
	f := &FromDataConverter{}
	f.loader = loader
	f.allowUnresolved = options.Get5(`allow_unresolved`, types.Boolean_FALSE).(*types.BooleanValue).Bool()
	f.richDataFuncs = map[string]richDataFunc{
		PCORE_TYPE_HASH: func(hash eval.KeyedValue, typeValue eval.PValue) eval.PValue {
			value := hash.Get5(PCORE_VALUE_KEY, eval.EMPTY_ARRAY).(eval.IndexedValue)
			return f.buildHash(func() {
				top := value.Len()
				idx := 0
				for idx < top {
					key := f.withoutValue(func() eval.PValue {
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

		PCORE_TYPE_SENSITIVE: func(hash eval.KeyedValue, typeValue eval.PValue) eval.PValue {
			return f.build(&valueBuilder{types.WrapSensitive(f.Convert(hash.Get5(PCORE_VALUE_KEY, eval.UNDEF)))}, nil)
		},

		PCORE_TYPE_DEFAULT: func(hash eval.KeyedValue, typeValue eval.PValue) eval.PValue {
			return f.build(&valueBuilder{types.WrapDefault()}, nil)
		},

		PCORE_TYPE_SYMBOL: func(hash eval.KeyedValue, typeValue eval.PValue) eval.PValue {
			return f.build(&valueBuilder{types.WrapRuntime(Symbol(hash.Get5(PCORE_VALUE_KEY, eval.EMPTY_STRING).String()))}, nil)
		},

		PCORE_LOCAL_REF_SYMBOL: func(hash eval.KeyedValue, typeValue eval.PValue) eval.PValue {
			path := hash.Get5(PCORE_VALUE_KEY, eval.EMPTY_STRING).String()
			if resolved, ok := resolveJsonPath(f.root, path); ok {
				return f.build(&valueBuilder{resolved}, nil)
			}
			panic(eval.Error(eval.EVAL_BAD_JSON_PATH, issue.H{path: path}))
		},
	}
	f.defaultFunc = func(hash eval.KeyedValue, typeValue eval.PValue) eval.PValue {
		value := hash.Get6(PCORE_VALUE_KEY, func() eval.PValue {
			return hash.RejectPairs(func(k, v eval.PValue) bool {
				if s, ok := k.(*types.StringValue); ok {
					return s.String() == PCORE_TYPE_KEY
				}
				return false
			})
		})
		if typeHash, ok := typeValue.(*types.HashValue); ok {
			typ := f.withoutValue(func() eval.PValue { return f.Convert(typeHash) })
			if typ, ok := typeValue.(*types.HashValue); ok {
				if !f.allowUnresolved {
					panic(eval.Error(eval.EVAL_UNABLE_TO_DESERIALIZE_TYPE, issue.H{`hash`: typ.String()}))
				}
				return hash
			}
			return f.pcoreTypeHashToValue(typ.(eval.PType), value)
		}
		typ := eval.CurrentContext().ParseType(typeValue)
		if tr, ok := typ.(*types.TypeReferenceType); ok {
			if !f.allowUnresolved {
				panic(eval.Error(eval.EVAL_UNRESOLVED_TYPE, issue.H{`typeString`: tr.String()}))
			}
			return hash
		}
		return f.pcoreTypeHashToValue(typ.(eval.PType), value)
	}
	return f
}

func parseKeyword(lexer parser.Lexer) (eval.PValue, bool) {
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

func parseStringOrInteger(lexer parser.Lexer) (eval.PValue, bool) {
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

func resolveJsonPath(lhs builder, path string) (eval.PValue, bool) {
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
			return lhs.resolve(), true
		}
		return nil, false
	}
}

func (f *FromDataConverter) Convert(value eval.PValue) eval.PValue {
	if hash, ok := value.(*types.HashValue); ok {
		if pcoreType, ok := hash.Get4(PCORE_TYPE_KEY); ok {
			key := pcoreType.String()
			rdFunc, ok := f.richDataFuncs[key]
			if !ok {
				rdFunc = f.defaultFunc
			}
			return rdFunc(hash, pcoreType)
		}
		return f.buildHash(func() { hash.EachPair(func(k, v eval.PValue) { f.with(k, func() { f.Convert(v) }) }) })
	}
	if array, ok := value.(*types.ArrayValue); ok {
		return f.buildArray(func() {
			array.EachWithIndex(func(v eval.PValue, i int) { f.with(types.WrapInteger(int64(i)), func() { f.Convert(v) }) })
		})
	}
	return f.build(&valueBuilder{value}, nil)
}

func (f *FromDataConverter) buildHash(actor eval.Actor) eval.PValue {
	return f.build(&hashBuilder{hash.NewStringHash(31), nil}, actor)
}

func (f *FromDataConverter) buildObject(object eval.ObjectValue, actor eval.Actor) eval.PValue {
	return f.build(&objectHashBuilder{hashBuilder{hash.NewStringHash(31), nil}, object}, actor)
}

func (f *FromDataConverter) buildArray(actor eval.Actor) eval.PValue {
	return f.build(&arrayBuilder{make([]builder, 0, 32), nil}, actor)
}

func (f *FromDataConverter) build(vx builder, actor eval.Actor) eval.PValue {
	if f.current != nil {
		f.current.put(f.key, vx)
	}
	if actor != nil {
		f.withValue(vx, actor)
	}
	return vx.resolve()
}

func (f *FromDataConverter) with(key eval.PValue, actor eval.Actor) {
	parentKey := f.key
	f.key = key
	actor()
	f.key = parentKey
}

func (f *FromDataConverter) withValue(value builder, actor eval.Actor) builder {
	if f.root == nil {
		f.root = value
	}
	parent := f.current
	f.current = value
	actor()
	f.current = parent
	return value
}

func (f *FromDataConverter) withoutValue(producer eval.Producer) eval.PValue {
	parent := f.current
	f.current = nil
	value := producer()
	f.current = parent
	return value
}

func (f *FromDataConverter) pcoreTypeHashToValue(typ eval.PType, value eval.PValue) eval.PValue {
	if hash, ok := value.(*types.HashValue); ok {
		if ov, ok := f.allocate(typ); ok {
			return f.buildObject(ov, func() {
				hash.EachPair(func(key, elem eval.PValue) { f.with(key, func() { f.Convert(elem) }) })
			})
		}
		return f.create(typ, f.buildHash(func() {
			hash.EachPair(func(key, elem eval.PValue) { f.with(key, func() { f.Convert(elem) }) })
		}))
	}
	if str, ok := value.(*types.StringValue); ok {
		return f.create(typ, str)
	}
	panic(eval.Error(eval.EVAL_UNABLE_TO_DESERIALIZE_VALUE, issue.H{`type`: typ.Name(), `arg_type`: value.Type().Name()}))
}

func (f *FromDataConverter) allocate(typ eval.PType) (eval.ObjectValue, bool) {
	if allocator, ok := eval.Load(f.loader, eval.NewTypedName(eval.ALLOCATOR, typ.Name())); ok {
		return allocator.(eval.Lambda).Call(nil, nil).(eval.ObjectValue), true
	}
	return nil, false
}

func (f *FromDataConverter) create(typ eval.PType, arg eval.PValue) eval.PValue {
	if ctor, ok := eval.Load(f.loader, eval.NewTypedName(eval.CONSTRUCTOR, typ.Name())); ok {
		return ctor.(eval.Function).Call(eval.CurrentContext(), nil, arg)
	}
	panic(eval.Error(eval.EVAL_CTOR_NOT_FOUND, issue.H{`type`: typ.Name()}))
}
