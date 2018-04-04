package impl

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/types"
	"github.com/puppetlabs/go-parser/issue"
	"github.com/puppetlabs/go-parser/parser"
)

func (e *evaluator) eval_AccessExpression(expr *parser.AccessExpression, c eval.Context) (result eval.PValue) {
	op := expr.Operand()
	keys := expr.Keys()
	nArgs := len(keys)
	args := make([]eval.PValue, nArgs)
	for idx, key := range keys {
		args[idx] = e.eval(key, c)
	}

	if qr, ok := op.(*parser.QualifiedReference); ok {
		return e.eval_ParameterizedTypeExpression(qr, args, expr, c)
	}

	lhs := e.eval(op, c)

	var opV eval.SizedValue
	if sv, ok := lhs.(eval.SizedValue); ok {
		opV = sv
	} else {
		panic(e.evalError(eval.EVAL_OPERATOR_NOT_APPLICABLE, op, issue.H{`operator`: `[]`, `left`: lhs.Type()}))
	}

	intArg := func(index int) int {
		key := args[index]
		if arg, ok := eval.ToInt(key); ok {
			return int(arg)
		}
		panic(e.evalError(eval.EVAL_ILLEGAL_ARGUMENT_TYPE, keys[index],
			issue.H{`expression`: opV.Type(), `number`: 0, `expected`: `Integer`, `actual`: key}))
	}

	indexArg := func(argIndex int) int {
		index := intArg(argIndex)
		if index < 0 {
			index = opV.Len() + index
		}
		if index > opV.Len() {
			index = opV.Len()
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
			count = 1 + (opV.Len() + count) - start
			if count < 0 {
				count = 0
			}
		} else if start+count > opV.Len() {
			count = opV.Len() - start
		}
		return
	}

	switch opV.(type) {
	case *types.HashValue:
		if opV.Len() == 0 {
			return eval.UNDEF
		}
		hv := opV.(*types.HashValue)
		if nArgs == 0 {
			panic(e.evalError(eval.EVAL_ILLEGAL_ARGUMENT_COUNT, expr, issue.H{`expression`: opV.Type(), `expected`: `at least one`, `actual`: nArgs}))
		}
		if nArgs == 1 {
			if v, ok := hv.Get(args[0]); ok {
				return v
			}
			return eval.UNDEF
		}
		el := make([]eval.PValue, 0, nArgs)
		for _, key := range args {
			if v, ok := hv.Get(key); ok {
				el = append(el, v)
			}
		}
		return types.WrapArray(el)

	case eval.IndexedValue:
		if nArgs == 0 || nArgs > 2 {
			panic(e.evalError(eval.EVAL_ILLEGAL_ARGUMENT_COUNT, expr, issue.H{`expression`: opV.Type(), `expected`: `1 or 2`, `actual`: nArgs}))
		}
		if nArgs == 2 {
			start := indexArg(0)
			count := countArg(1, start)
			if start < 0 {
				start = 0
			}
			if start == opV.Len() || count == 0 {
				if _, ok := opV.(*types.StringValue); ok {
					return eval.EMPTY_STRING
				}
				return eval.EMPTY_ARRAY
			}
			return opV.(eval.IndexedValue).Slice(start, start+count)
		}
		pos := intArg(0)
		if pos < 0 {
			pos = opV.Len() + pos
			if pos < 0 {
				return eval.UNDEF
			}
		}
		if pos >= opV.Len() {
			return eval.UNDEF
		}
		return opV.(eval.IndexedValue).At(pos)

	default:
		panic(e.evalError(eval.EVAL_OPERATOR_NOT_APPLICABLE, op, issue.H{`operator`: `[]`, `left`: opV.Type()}))
	}
}

func (e *evaluator) eval_ParameterizedTypeExpression(qr *parser.QualifiedReference, args []eval.PValue, expr *parser.AccessExpression, c eval.Context) (tp eval.PType) {
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
	case `notundef`:
		tp = types.NewNotUndefType2(args...)
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
