package serialization

import (
	. "github.com/puppetlabs/go-evaluator/evaluator"
	. "github.com/puppetlabs/go-evaluator/hash"
	. "github.com/puppetlabs/go-evaluator/types"
	. "github.com/puppetlabs/go-parser/parser"
	"github.com/puppetlabs/go-parser/issue"
)

type (
	richDataFunc func(hash KeyedValue, typeValue PValue) PValue

	FromDataConverter struct {
		root builder
		current builder
		key PValue
		allowUnresolved bool
		loader DefiningLoader
		richDataFuncs map[string]richDataFunc
		defaultFunc richDataFunc
	}

	builder interface {
		get(key PValue) builder
		put(key PValue, value builder)
		resolve() PValue
	}

	valueBuilder struct {
		value PValue
	}

	hbEntry struct {
		key PValue
		value builder
	}

	hashBuilder struct {
    values *StringHash
    resolved PValue
	}

	objectHashBuilder struct {
		hashBuilder
		object ObjectValue
	}

	arrayBuilder struct {
		// *StringHash, []interface{}, or interface{}
		values []builder
		resolved PValue
	}
)

func (b *valueBuilder) get(key PValue) builder {
	panic(`scalar indexed by string`)
}

func (b *valueBuilder) put(key PValue, value builder) {
	panic(`scalar indexed by string`)
}

func (b *valueBuilder) resolve() PValue {
	return b.value
}

func (b *hashBuilder) get(key PValue) builder {
	if v, ok := b.values.Get(string(ToKey(key)), nil).(*hbEntry); ok {
		return v.value
	}
	return nil
}

func (b *hashBuilder) put(key PValue, value builder) {
	b.values.Put(string(ToKey(key)), &hbEntry{key, value})
}

func (b *hashBuilder) resolve() PValue {
	if b.resolved == nil {
		es := make([]*HashEntry, 0, b.values.Size())
		b.values.EachPair(func(key string, value interface{}) {
			hbe := value.(*hbEntry)
			es = append(es, WrapHashEntry(hbe.key, hbe.value.resolve()))
		})
		b.resolved =  WrapHash(es)
	}
	return b.resolved
}

func (b *arrayBuilder) get(key PValue) builder {
	if index, ok := ToInt(key); ok && int(index) < len(b.values) {
		return b.values[index]
	}
	return nil
}

func (b *arrayBuilder) put(key PValue, value builder) {
	if index, ok := ToInt(key); ok {
		// Allow growth of one. Beyond that will result in an index out of bounds panic
		if int(index) == len(b.values) {
			b.values = append(b.values, value)
		} else {
			b.values[index] = value
		}
	}
}

func (b *arrayBuilder) resolve() PValue {
	if b.resolved == nil {
		es := make([]PValue, len(b.values))
		for i, v := range b.values {
			es[i] = v.resolve()
		}
		b.resolved =  WrapArray(es)
	}
	return b.resolved
}

func (b *objectHashBuilder) resolve() PValue {
	if b.resolved == nil {
		b.hashBuilder.resolve()
		b.object.InitFromHash(b.hashBuilder.resolve().(*HashValue))
		b.resolved = b.object
	}
	return b.resolved
}

func NewFromDataConverter(loader DefiningLoader, options KeyedValue) *FromDataConverter {
	f := &FromDataConverter{}
	f.loader = loader
	f.allowUnresolved = options.Get5(`allow_unresolved`, Boolean_FALSE).(*BooleanValue).Bool()
	f.richDataFuncs = map[string]richDataFunc {
		PCORE_TYPE_HASH: func(hash KeyedValue, typeValue PValue) PValue {
			value := hash.Get5(PCORE_VALUE_KEY, EMPTY_ARRAY).(IndexedValue)
			return f.buildHash(func() {
				top := value.Len()
				idx := 0
				for idx < top {
					key := f.withoutValue(func() PValue {
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

		PCORE_TYPE_SENSITIVE: func(hash KeyedValue, typeValue PValue) PValue {
			return f.build(&valueBuilder{WrapSensitive(f.Convert(hash.Get5(PCORE_VALUE_KEY, UNDEF)))}, nil)
		},

		PCORE_TYPE_DEFAULT: func(hash KeyedValue, typeValue PValue) PValue {
			return f.build(&valueBuilder{WrapDefault()}, nil)
		},

		PCORE_TYPE_SYMBOL: func(hash KeyedValue, typeValue PValue) PValue {
			return f.build(&valueBuilder{WrapRuntime(Symbol(hash.Get5(PCORE_VALUE_KEY, EMPTY_STRING).String()))}, nil)
		},

		PCORE_LOCAL_REF_SYMBOL: func(hash KeyedValue, typeValue PValue) PValue {
			path := hash.Get5(PCORE_VALUE_KEY, EMPTY_STRING).String()
			if resolved, ok := resolveJsonPath(f.root, path); ok {
				return f.build(&valueBuilder{resolved}, nil)
			}
			panic(Error(EVAL_BAD_JSON_PATH, issue.H{path: path}))
		},
	}
	f.defaultFunc = func(hash KeyedValue, typeValue PValue) PValue {
		value := hash.Get6(PCORE_VALUE_KEY, func() PValue {
			return hash.RejectPairs(func(k, v PValue) bool {
				if s, ok := k.(*StringValue); ok {
					return s.String() == PCORE_TYPE_KEY
				}
				return false
			})
		})
		if typeHash, ok := typeValue.(*HashValue); ok {
			typ := f.withoutValue(func() PValue { return f.Convert(typeHash) })
			if typ, ok := typeValue.(*HashValue); ok {
				if !f.allowUnresolved {
					panic(Error(EVAL_UNABLE_TO_DESERIALIZE_TYPE, issue.H{`hash`: typ.String()}))
				}
				return hash
			}
			return f.pcoreTypeHashToValue(typ.(PType), value)
		}
		typ := CurrentContext().ParseType(typeValue)
		if tr, ok := typ.(*TypeReferenceType); ok {
			if !f.allowUnresolved {
				panic(Error(EVAL_UNRESOLVED_TYPE, issue.H{`typeString`: tr.String()}))
			}
			return hash
		}
		return f.pcoreTypeHashToValue(typ.(PType), value)
	}
	return f
}

func parseKeyword(lexer Lexer) (PValue, bool) {
	t := lexer.CurrentToken()
	switch t {
	case TOKEN_IDENTIFIER, TOKEN_TYPE_NAME:
		s := lexer.TokenString()
		if s != `null` {
			return WrapString(lexer.TokenString()), true
		}
	default:
		if IsKeywordToken(lexer.CurrentToken()) {
			return WrapString(lexer.TokenString()), true
		}
	}
	return nil, false
}

func parseStringOrInteger(lexer Lexer) (PValue, bool) {
	t := lexer.CurrentToken()
	switch t {
	case TOKEN_INTEGER:
		return WrapInteger(lexer.TokenValue().(int64)), true
	case TOKEN_STRING:
		s := lexer.TokenString()
		if s != `null` {
			return WrapString(lexer.TokenString()), true
		}
	default:
		if IsKeywordToken(lexer.CurrentToken()) {
			return WrapString(lexer.TokenString()), true
		}
	}
	return nil, false
}

func resolveJsonPath(lhs builder, path string) (PValue, bool) {
	lexer := NewSimpleLexer(``, path)
	lexer.NextToken()
	lexer.AssertToken(TOKEN_VARIABLE)
	for {
		lexer.NextToken()
		switch lexer.CurrentToken() {
		case TOKEN_DOT:
			lexer.NextToken()
			if key, ok := parseKeyword(lexer); ok {
				lhs = lhs.get(key)
				if lhs == nil {
					return nil, false
				}
				continue
			}
		case TOKEN_LB:
			lexer.NextToken()
			if key, ok := parseStringOrInteger(lexer); ok {
				lexer.NextToken()
				lexer.AssertToken(TOKEN_RB)
				lhs = lhs.get(key)
				if lhs == nil {
					return nil, false
				}
				continue
			}
		case TOKEN_END:
			return lhs.resolve(), true
		}
		return nil, false
	}
}

func (f *FromDataConverter) Convert(value PValue) PValue {
  if hash, ok := value.(*HashValue); ok {
  	if pcoreType, ok := hash.Get4(PCORE_TYPE_KEY); ok {
  		key := pcoreType.String()
  		rdFunc, ok := f.richDataFuncs[key]
  		if !ok {
  			rdFunc = f.defaultFunc
		  }
  		return rdFunc(hash, pcoreType)
	  }
	  return f.buildHash(func() { hash.EachPair(func(k, v PValue) { f.with(k, func() { f.Convert(v) }) }) })
  }
  if array, ok := value.(*ArrayValue); ok {
  	return f.buildArray(func() {
  		array.EachWithIndex (func(v PValue, i int) { f.with(WrapInteger(int64(i)), func() { f.Convert(v) }) })
  	})
  }
  return f.build(&valueBuilder{value}, nil)
}

func (f *FromDataConverter) buildHash(actor Actor) PValue {
	return f.build(&hashBuilder{NewStringHash(31), nil}, actor)
}

func (f *FromDataConverter) buildObject(object ObjectValue, actor Actor) PValue {
	return f.build(&objectHashBuilder{hashBuilder{NewStringHash(31), nil}, object}, actor)
}

func (f *FromDataConverter) buildArray(actor Actor) PValue {
	return f.build(&arrayBuilder{make([]builder, 0, 32), nil}, actor)
}

func (f *FromDataConverter) build(vx builder, actor Actor) PValue {
	if f.current != nil {
		f.current.put(f.key, vx)
	}
	if actor != nil {
		f.withValue(vx, actor)
	}
	return vx.resolve()
}

func (f *FromDataConverter) with(key PValue, actor Actor) {
  parentKey := f.key
  f.key = key
  actor()
  f.key = parentKey
}

func (f *FromDataConverter) withValue(value builder, actor Actor) builder {
	if f.root == nil {
		f.root = value
	}
	parent := f.current
	f.current = value
	actor()
	f.current = parent
	return value
}

func (f *FromDataConverter) withoutValue(producer Producer) PValue {
	parent := f.current
	f.current = nil
	value := producer()
	f.current = parent
	return value
}

func (f *FromDataConverter) pcoreTypeHashToValue(typ PType, value PValue) PValue {
	if hash, ok := value.(*HashValue); ok {
		if ov, ok := f.allocate(typ); ok {
			return f.buildObject(ov, func() {
				hash.EachPair(func(key, elem PValue) { f.with(key, func() { f.Convert(elem) }) })
			})
		}
		return f.create(typ, f.buildHash(func() {
			hash.EachPair(func(key, elem PValue) { f.with(key, func() { f.Convert(elem) }) })
		}))
	}
	if str, ok := value.(*StringValue); ok {
		return f.create(typ, str)
	}
	panic(Error(EVAL_UNABLE_TO_DESERIALIZE_VALUE, issue.H{`type`: typ.Name(), `arg_type`: value.Type().Name()}))
}

func (f *FromDataConverter) allocate(typ PType) (ObjectValue, bool) {
	if allocator, ok := Load(f.loader, NewTypedName(ALLOCATOR, typ.Name())); ok {
		return allocator.(Lambda).Call(nil, nil).(ObjectValue), true
	}
	return nil, false
}

func (f *FromDataConverter) create(typ PType, arg PValue) PValue {
	if ctor, ok := Load(f.loader, NewTypedName(CONSTRUCTOR, typ.Name())); ok {
		return ctor.(Function).Call(CurrentContext(), nil, arg)
	}
	panic(Error(EVAL_CTOR_NOT_FOUND, issue.H{`type`: typ.Name()}))
}