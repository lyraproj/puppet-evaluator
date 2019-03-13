package evaluator

import (
	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
	"github.com/lyraproj/puppet-evaluator/pdsl"
	"github.com/lyraproj/puppet-parser/parser"
)

func evalAccessExpression(e pdsl.Evaluator, expr *parser.AccessExpression) (result px.Value) {
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
					return objectTypeFromAST(e, name, nil, hash)
				}

				na := e.Loader().NameAuthority()
				ne = hash.Get(`name_authority`)
				if ne != nil {
					na = px.URI(e.Eval(ne).String())
				}
				return typeSetTypeFromAST(e, na, name, hash)
			}
		}

		args := make([]px.Value, len(keys))
		e.DoStatic(func() {
			for idx, key := range keys {
				args[idx] = e.Eval(key)
			}
		})
		return evalParameterizedTypeExpression(e, qr, args, expr)
	}

	args := make([]px.Value, len(keys))
	for idx, key := range keys {
		args[idx] = e.Eval(key)
	}

	lhs := e.Eval(op)

	switch lhs := lhs.(type) {
	case px.List:
		return accessIndexedValue(expr, lhs, args)
	default:
		if tem, ok := lhs.PType().(px.TypeWithCallableMembers); ok {
			if mbr, ok := tem.Member(`[]`); ok {
				return mbr.Call(e, lhs, nil, args)
			}
		}
	}
	panic(evalError(pdsl.OperatorNotApplicable, op, issue.H{`operator`: `[]`, `left`: lhs.PType()}))
}

func accessIndexedValue(expr *parser.AccessExpression, lhs px.List, args []px.Value) (result px.Value) {
	nArgs := len(args)

	intArg := func(index int) int {
		key := args[index]
		if arg, ok := px.ToInt(key); ok {
			return int(arg)
		}
		panic(evalError(pdsl.IllegalArgumentType, expr.Keys()[index],
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

	if hv, ok := lhs.(*types.Hash); ok {
		if hv.Len() == 0 {
			return px.Undef
		}
		if nArgs == 0 {
			panic(evalError(pdsl.IllegalArgumentCount, expr, issue.H{`expression`: lhs.PType(), `expected`: `at least one`, `actual`: nArgs}))
		}
		if nArgs == 1 {
			if v, ok := hv.Get(args[0]); ok {
				return v
			}
			return px.Undef
		}
		el := make([]px.Value, 0, nArgs)
		for _, key := range args {
			if v, ok := hv.Get(key); ok {
				el = append(el, v)
			}
		}
		return types.WrapValues(el)
	}

	if nArgs == 0 || nArgs > 2 {
		panic(evalError(pdsl.IllegalArgumentCount, expr, issue.H{`expression`: lhs.PType(), `expected`: `1 or 2`, `actual`: nArgs}))
	}
	if nArgs == 2 {
		start := indexArg(0)
		count := countArg(1, start)
		if start < 0 {
			start = 0
		}
		if start == lhs.Len() || count == 0 {
			if _, ok := lhs.(px.StringValue); ok {
				return px.EmptyString
			}
			return px.EmptyArray
		}
		return lhs.Slice(start, start+count)
	}
	pos := intArg(0)
	if pos < 0 {
		pos = lhs.Len() + pos
		if pos < 0 {
			return px.Undef
		}
	}
	if pos >= lhs.Len() {
		return px.Undef
	}
	return lhs.At(pos)
}

func evalParameterizedTypeExpression(e pdsl.Evaluator, qr *parser.QualifiedReference, args []px.Value, expr *parser.AccessExpression) (tp px.Type) {
	return types.ResolveWithParams(e, qr.Name(), args)
}
