package impl

import (
	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/types"
	"github.com/lyraproj/puppet-parser/parser"
)

func evalAccessExpression(e eval.Evaluator, expr *parser.AccessExpression) (result eval.Value) {
	keys := expr.Keys()
	op := expr.Operand()
	if qr, ok := op.(*parser.QualifiedReference); ok {
		if (qr.Name() == `TypeSet` || qr.Name() == `Object`) && len(keys) == 1 {
			// Defer evaluation of the type parameter until type is resolved
			if hash, ok := keys[0].(*parser.LiteralHash); ok {
				name := ``
				ne := hash.Get(`name`)
				if ne != nil {
					name = e.Eval(ne).String()
				}
				if qr.Name() == `Object` {
					return types.NewObjectType(name, nil, hash)
				}

				na := eval.RUNTIME_NAME_AUTHORITY
				ne = hash.Get(`name_authority`)
				if ne != nil {
					na = eval.URI(e.Eval(ne).String())
				}
				return types.NewTypeSetType(na, name, hash)
			}
		}

		args := make([]eval.Value, len(keys))
		e.DoStatic(func() {
			for idx, key := range keys {
				args[idx] = e.Eval(key)
			}
		})
		return eval_ParameterizedTypeExpression(e, qr, args, expr)
	}

	args := make([]eval.Value, len(keys))
	for idx, key := range keys {
		args[idx] = e.Eval(key)
	}

	lhs := e.Eval(op)

	switch lhs.(type) {
	case eval.List:
		return accessIndexedValue(expr, lhs.(eval.List), args)
	default:
		if tem, ok := lhs.PType().(eval.TypeWithCallableMembers); ok {
			if mbr, ok := tem.Member(`[]`); ok {
				return mbr.Call(e, lhs, nil, args)
			}
		}
	}
	panic(evalError(eval.EVAL_OPERATOR_NOT_APPLICABLE, op, issue.H{`operator`: `[]`, `left`: lhs.PType()}))
}

func accessIndexedValue(expr *parser.AccessExpression, lhs eval.List, args []eval.Value) (result eval.Value) {
	nArgs := len(args)

	intArg := func(index int) int {
		key := args[index]
		if arg, ok := eval.ToInt(key); ok {
			return int(arg)
		}
		panic(evalError(eval.EVAL_ILLEGAL_ARGUMENT_TYPE, expr.Keys()[index],
			issue.H{`expression`: lhs.PType(), `number`: 0, `expected`: `Integer`, `actual`: key}))
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
			panic(evalError(eval.EVAL_ILLEGAL_ARGUMENT_COUNT, expr, issue.H{`expression`: lhs.PType(), `expected`: `at least one`, `actual`: nArgs}))
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
		return types.WrapValues(el)
	}

	if nArgs == 0 || nArgs > 2 {
		panic(evalError(eval.EVAL_ILLEGAL_ARGUMENT_COUNT, expr, issue.H{`expression`: lhs.PType(), `expected`: `1 or 2`, `actual`: nArgs}))
	}
	if nArgs == 2 {
		start := indexArg(0)
		count := countArg(1, start)
		if start < 0 {
			start = 0
		}
		if start == lhs.Len() || count == 0 {
			if _, ok := lhs.(eval.StringValue); ok {
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

func eval_ParameterizedTypeExpression(e eval.Evaluator, qr *parser.QualifiedReference, args []eval.Value, expr *parser.AccessExpression) (tp eval.Type) {
	defer func() {
		if err := recover(); err != nil {
			convertCallError(err, expr, expr.Keys())
		}
	}()
	return types.ResolveWithParams(e, qr.Name(), args)
}
