package eval

import (
	. "github.com/puppetlabs/go-evaluator/evaluator"
	. "github.com/puppetlabs/go-evaluator/types"
	. "github.com/puppetlabs/go-parser/parser"
	. "github.com/puppetlabs/go-parser/issue"
)

func (e *evaluator) eval_AccessExpression(expr *AccessExpression, c EvalContext) (result PValue) {
	op := expr.Operand()
	keys := expr.Keys()
	nArgs := len(keys)
	args := make([]PValue, nArgs)
	for idx, key := range keys {
		args[idx] = e.eval(key, c)
	}

	if qr, ok := op.(*QualifiedReference); ok {
		return e.eval_ParameterizedTypeExpression(qr, args, expr, c.Loader())
	}

	lhs := e.eval(op, c)

	var opV SizedValue
	if sv, ok := lhs.(SizedValue); ok {
		opV = sv
	} else {
		panic(e.evalError(EVAL_OPERATOR_NOT_APPLICABLE, op, H{`operator`: `[]`, `left`: lhs.Type()}))
	}

	intArg := func(index int) int {
		key := args[index]
		if arg, ok := ToInt(key); ok {
			return int(arg)
		}
		panic(e.evalError(EVAL_ILLEGAL_ARGUMENT_TYPE, keys[index],
			H{`expression`: opV.Type(), `number`: 0, `expected`: `Integer`, `actual`: key}))
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
	case *HashValue:
		if opV.Len() == 0 {
			return UNDEF
		}
		hv := opV.(*HashValue)
		if nArgs == 0 {
			panic(e.evalError(EVAL_ILLEGAL_ARGUMENT_COUNT, expr, H{`expression`: opV.Type(), `expected`: `at least one`, `actual`: nArgs}))
		}
		if nArgs == 1 {
			if v, ok := hv.Get(args[0]); ok {
				return v
			}
			return UNDEF
		}
		el := make([]PValue, 0, nArgs)
		for _, key := range args {
			if v, ok := hv.Get(key); ok {
				el = append(el, v)
			}
		}
		return WrapArray(el)

	case *ArrayValue:
		if nArgs == 0 || nArgs > 2 {
			panic(e.evalError(EVAL_ILLEGAL_ARGUMENT_COUNT, expr, H{`expression`: opV.Type(), `expected`: `1 or 2`, `actual`: nArgs}))
		}
		if nArgs == 2 {
			start := indexArg(0)
			if start < 0 {
				start = 0
			}
			count := countArg(1, start)
			if start == opV.Len() || count == 0 {
				return EMPTY_ARRAY
			}
			return opV.(*ArrayValue).Slice(start, start+count)
		}
		pos := intArg(0)
		if pos < 0 {
			pos = opV.Len() + pos
			if pos < 0 {
				return UNDEF
			}
		}
		if pos >= opV.Len() {
			return UNDEF
		}
		return opV.(*ArrayValue).At(pos)

	case *StringValue:
		if nArgs == 0 || nArgs > 2 {
			panic(e.evalError(EVAL_ILLEGAL_ARGUMENT_COUNT, expr, H{`expression`: opV.Type(), `expected`: `1 or 2`, `actual`: nArgs}))
		}
		if nArgs == 2 {
			start := indexArg(0)
			count := countArg(1, start)
			if start < 0 {
				start = 0
			}
			if start == opV.Len() || count == 0 {
				return EMPTY_STRING
			}
			return opV.(*StringValue).Slice(start, start+count)
		}
		pos := intArg(0)
		if pos < 0 {
			pos = opV.Len() + pos
			if pos < 0 {
				return nil
			}
		}
		if pos >= opV.Len() {
			return nil
		}
		return opV.(*StringValue).Slice(pos, pos+1)

	default:
		panic(e.evalError(EVAL_OPERATOR_NOT_APPLICABLE, op, H{`operator`: `[]`, `left`: opV.Type()}))
	}
}

func (e *evaluator) eval_ParameterizedTypeExpression(qr *QualifiedReference, args []PValue, expr *AccessExpression, loader Loader) (tp PType) {
	dcName := qr.DowncasedName()
	defer func() {
		if err := recover(); err != nil {
			e.convertCallError(err, expr, expr.Keys())
		}
	}()

	switch dcName {
	case `array`:
		tp = NewArrayType2(args...)
	case `callable`:
		tp = NewCallableType2(args...)
	case `collection`:
		tp = NewCollectionType2(args...)
	case `enum`:
		tp = NewEnumType2(args...)
	case `float`:
		tp = NewFloatType2(args...)
	case `hash`:
		tp = NewHashType2(args...)
	case `integer`:
		tp = NewIntegerType2(args...)
	case `iterable`:
		tp = NewIterableType2(args...)
	case `iterator`:
		tp = NewIteratorType2(args...)
	case `notundef`:
		tp = NewNotUndefType2(args...)
	case `optional`:
		tp = NewOptionalType2(args...)
	case `pattern`:
		tp = NewPatternType2(args...)
	case `regexp`:
		tp = NewRegexpType2(args...)
	case `runtime`:
		tp = NewRuntimeType2(args...)
	case `semver`:
		tp = NewSemVerType2(args...)
	case `string`:
		tp = NewStringType2(args...)
	case `struct`:
		tp = NewStructType2(args...)
	case `timespan`:
		tp = NewTimespanType2(args...)
	case `timestamp`:
		tp = NewTimestampType2(args...)
	case `tuple`:
		tp = NewTupleType2(args...)
	case `type`:
		tp = NewTypeType2(args...)
	case `typereference`:
		tp = NewTypeReferenceType2(args...)
	case `variant`:
		tp = NewVariantType2(args...)
	case `any`:
	case `binary`:
	case `boolean`:
	case `catalogentry`:
	case `data`:
	case `default`:
	case `numeric`:
	case `scalar`:
	case `semverrange`:
	case `typealias`:
	case `undef`:
	case `unit`:
		panic(e.evalError(EVAL_NOT_PARAMETERIZED_TYPE, expr, H{`type`: expr}))
	default:
		tp = NewTypeReferenceType(expr.String())
	}
	return
}
