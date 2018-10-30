package functions

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/types"
	"github.com/puppetlabs/go-issues/issue"
)

func init() {
	eval.NewGoFunction2(`match`,
		func(l eval.LocalTypes) {
			l.Type(`Patterns`, `Variant[Regexp, String, Type[Pattern], Type[Regexp], Array[Patterns]]`)
		},

		func(d eval.Dispatch) {
			d.Param(`String`)
			d.Param(`Patterns`)
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				return matchPatterns(c, args[0].String(), args[1])
			})
		},

		func(d eval.Dispatch) {
			d.Param(`Array[String]`)
			d.Param(`Patterns`)
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				p := args[1]
				return args[0].(*types.ArrayValue).Map(func(arg eval.Value) eval.Value {
					return matchPatterns(c, arg.String(), p)
				})
			})
		},
	)
}

func matchPatterns(c eval.Context, s string, v eval.Value) eval.Value {
	switch v.(type) {
	case *types.RegexpValue:
		return matchRegexp(c, s, v.(*types.RegexpValue))
	case *types.PatternType:
		return matchArray(c, s, v.(*types.PatternType).Patterns())
	case *types.RegexpType:
		return matchRegexp(c, s, types.WrapRegexp(v.(*types.RegexpType).PatternString()))
	case *types.ArrayType:
		return matchArray(c, s, v.(*types.ArrayValue))
	default:
		return matchRegexp(c, s, types.WrapRegexp(v.String()))
	}
}

func matchRegexp(c eval.Context, s string, rx *types.RegexpValue) eval.Value {
	if rx.PatternString() == `` {
		panic(eval.Error(eval.EVAL_MISSING_REGEXP_IN_TYPE, issue.NO_ARGS))
	}

	g := rx.Match(s)
	if g == nil {
		return eval.UNDEF
	}
	rs := make([]eval.Value, len(g))
	for i, s := range g {
		rs[i] = types.WrapString(s)
	}
	return types.WrapArray(rs)
}

func matchArray(c eval.Context, s string, ar *types.ArrayValue) eval.Value {
	result := eval.UNDEF
	ar.Find(func(p eval.Value) bool {
		r := matchPatterns(c, s, p)
		if r == eval.UNDEF {
			return false
		}
		result = r
		return true
	})
	return result
}
