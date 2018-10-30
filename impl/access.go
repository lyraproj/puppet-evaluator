package impl

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/types"
	"github.com/puppetlabs/go-issues/issue"
	"github.com/puppetlabs/go-parser/parser"
)

func (e *evaluator) eval_AccessExpression(expr *parser.AccessExpression, c eval.Context) (result eval.Value) {
	keys := expr.Keys()
	op := expr.Operand()
	if qr, ok := op.(*parser.QualifiedReference); ok {
		if (qr.Name() == `TypeSet` || qr.Name() == `Object`) && len(keys) == 1 {
			// Defer evaluation of the type parameter until type is resolved
			if hash, ok := keys[0].(*parser.LiteralHash); ok {
				name := ``
				ne := hash.Get(`name`)
				if ne != nil {
					name = e.eval(ne, c).String()
				}
				if qr.Name() == `Object` {
					return types.NewObjectType(name, nil, hash)
				}

				na := eval.RUNTIME_NAME_AUTHORITY
				ne = hash.Get(`name_authority`)
				if ne != nil {
					na = eval.URI(e.eval(ne, c).String())
				}
				return types.NewTypeSetType(na, name, hash)
			}
		}

		args := make([]eval.Value, len(keys))
		c.DoStatic(func() {
			for idx, key := range keys {
				args[idx] = e.eval(key, c)
			}
		})
		return e.eval_ParameterizedTypeExpression(qr, args, expr, c)
	}

	args := make([]eval.Value, len(keys))
	for idx, key := range keys {
		args[idx] = e.eval(key, c)
	}

	lhs := e.eval(op, c)

	switch lhs.(type) {
	case eval.List:
		return e.accessIndexedValue(expr, lhs.(eval.List), args, c)
	default:
		if tem, ok := lhs.Type().(eval.TypeWithCallableMembers); ok {
			if mbr, ok := tem.Member(`[]`); ok {
				return mbr.Call(c, lhs, nil, args)
			}

			if c.Language() == eval.LangJavaScript && len(args) == 1 {
				// In JavaScript, x['y'] is equal to x.y
				if s, ok := args[0].(*types.StringValue); ok {
					if mbr, ok := tem.Member(s.String()); ok {
						return mbr.Call(c, lhs, nil, []eval.Value{})
					}
					panic(e.evalError(eval.EVAL_ATTRIBUTE_NOT_FOUND, op, issue.H{`type`: lhs.Type(), `name`: s.String()}))
				}
			}
		}
	}
	panic(e.evalError(eval.EVAL_OPERATOR_NOT_APPLICABLE, op, issue.H{`operator`: `[]`, `left`: lhs.Type()}))
}

func (e *evaluator) accessIndexedValue(expr *parser.AccessExpression, lhs eval.List, args []eval.Value, c eval.Context) (result eval.Value) {
	nArgs := len(args)

	intArg := func(index int) int {
		key := args[index]
		if arg, ok := eval.ToInt(key); ok {
			return int(arg)
		}
		panic(e.evalError(eval.EVAL_ILLEGAL_ARGUMENT_TYPE, expr.Keys()[index],
			issue.H{`expression`: lhs.Type(), `number`: 0, `expected`: `Integer`, `actual`: key}))
	}

	indexArg := func(argIndex int) int {
		index := intArg(argIndex)
		if index < 0 {
			index = lhs.Len() + index
		}
		if index > lhs.Len() {
			index = lhs.Len()
		}
		return index
	}

	countArg := func(argIndex int, start int) (count int) {
		count = intArg(argIndex)
		if start < 0 {
			if count > 0 {
				count += start
				if count < 0 {
					count = 0
				}
			}
			start = 0
		}
		if count < 0 {
			count = 1 + (lhs.Len() + count) - start
			if count < 0 {
				count = 0
			}
		} else if start+count > lhs.Len() {
			count = lhs.Len() - start
		}
		return
	}

	if hv, ok := lhs.(*types.HashValue); ok {
		if hv.Len() == 0 {
			return eval.UNDEF
		}
		if nArgs == 0 {
			panic(e.evalError(eval.EVAL_ILLEGAL_ARGUMENT_COUNT, expr, issue.H{`expression`: lhs.Type(), `expected`: `at least one`, `actual`: nArgs}))
		}
		if nArgs == 1 {
			if v, ok := hv.Get(args[0]); ok {
				return v
			}
			return eval.UNDEF
		}
		el := make([]eval.Value, 0, nArgs)
		for _, key := range args {
			if v, ok := hv.Get(key); ok {
				el = append(el, v)
			}
		}
		return types.WrapArray(el)
	}

	if nArgs == 0 || nArgs > 2 {
		panic(e.evalError(eval.EVAL_ILLEGAL_ARGUMENT_COUNT, expr, issue.H{`expression`: lhs.Type(), `expected`: `1 or 2`, `actual`: nArgs}))
	}
	if nArgs == 2 {
		start := indexArg(0)
		count := countArg(1, start)
		if start < 0 {
			start = 0
		}
		if start == lhs.Len() || count == 0 {
			if _, ok := lhs.(*types.StringValue); ok {
				return eval.EMPTY_STRING
			}
			return eval.EMPTY_ARRAY
		}
		return lhs.Slice(start, start+count)
	}
	pos := intArg(0)
	if pos < 0 {
		pos = lhs.Len() + pos
		if pos < 0 {
			return eval.UNDEF
		}
	}
	if pos >= lhs.Len() {
		return eval.UNDEF
	}
	return lhs.At(pos)
}

func (e *evaluator) eval_ParameterizedTypeExpression(qr *parser.QualifiedReference, args []eval.Value, expr *parser.AccessExpression, c eval.Context) (tp eval.Type) {
	dcName := qr.DowncasedName()
	defer func() {
		if err := recover(); err != nil {
			e.convertCallError(err, expr, expr.Keys())
		}
	}()

	switch dcName {
	case `array`:
		tp = types.NewArrayType2(args...)
	case `boolean`:
		tp = types.NewBooleanType2(args...)
	case `callable`:
		tp = types.NewCallableType2(args...)
	case `collection`:
		tp = types.NewCollectionType2(args...)
	case `enum`:
		tp = types.NewEnumType2(args...)
	case `float`:
		tp = types.NewFloatType2(args...)
	case `hash`:
		tp = types.NewHashType2(args...)
	case `integer`:
		tp = types.NewIntegerType2(args...)
	case `iterable`:
		tp = types.NewIterableType2(args...)
	case `iterator`:
		tp = types.NewIteratorType2(args...)
	case `like`:
		tp = types.NewLikeType2(args...)
	case `notundef`:
		tp = types.NewNotUndefType2(args...)
	case `object`:
		tp = types.NewObjectType2(c, args...)
	case `optional`:
		tp = types.NewOptionalType2(args...)
	case `pattern`:
		tp = types.NewPatternType2(args...)
	case `regexp`:
		tp = types.NewRegexpType2(args...)
	case `runtime`:
		tp = types.NewRuntimeType2(args...)
	case `semver`:
		tp = types.NewSemVerType2(args...)
	case `sensitive`:
		tp = types.NewSensitiveType2(args...)
	case `string`:
		tp = types.NewStringType2(args...)
	case `struct`:
		tp = types.NewStructType2(args...)
	case `timespan`:
		tp = types.NewTimespanType2(args...)
	case `timestamp`:
		tp = types.NewTimestampType2(args...)
	case `tuple`:
		tp = types.NewTupleType2(args...)
	case `type`:
		tp = types.NewTypeType2(args...)
	case `typereference`:
		tp = types.NewTypeReferenceType2(args...)
	case `uri`:
		tp = types.NewUriType2(args...)
	case `variant`:
		tp = types.NewVariantType2(args...)
	case `any`:
	case `binary`:
	case `catalogentry`:
	case `data`:
	case `default`:
	case `numeric`:
	case `scalar`:
	case `semverrange`:
	case `typealias`:
	case `undef`:
	case `unit`:
		panic(e.evalError(eval.EVAL_NOT_PARAMETERIZED_TYPE, expr, issue.H{`type`: expr}))
	default:
		oe := e.eval(qr, c)
		if oo, ok := oe.(eval.ObjectType); ok && oo.IsParameterized() {
			tp = types.NewObjectTypeExtension(c, oo, args)
		} else {
			tp = types.NewTypeReferenceType(expr.String())
		}
	}
	return
}
