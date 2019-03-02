package functions

import (
	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/types"
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
				return matchPatterns(args[0].String(), args[1])
			})
		},

		func(d eval.Dispatch) {
			d.Param(`Array[String]`)
			d.Param(`Patterns`)
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				p := args[1]
				return args[0].(*types.ArrayValue).Map(func(arg eval.Value) eval.Value {
					return matchPatterns(arg.String(), p)
				})
			})
		},
	)
}

func matchPatterns(s string, v eval.Value) eval.Value {
	switch v := v.(type) {
	case *types.RegexpValue:
		return matchRegexp(s, v)
	case *types.PatternType:
		return matchArray(s, v.Patterns())
	case *types.RegexpType:
		return matchRegexp(s, types.WrapRegexp(v.PatternString()))
	case *types.ArrayValue:
		return matchArray(s, v)
	default:
		return matchRegexp(s, types.WrapRegexp(v.String()))
	}
}

func matchRegexp(s string, rx *types.RegexpValue) eval.Value {
	if rx.PatternString() == `` {
		panic(eval.Error(eval.MissingRegexpInType, issue.NO_ARGS))
	}

	g := rx.Match(s)
	if g == nil {
		return eval.Undef
	}
	rs := make([]eval.Value, len(g))
	for i, s := range g {
		rs[i] = types.WrapString(s)
	}
	return types.WrapValues(rs)
}

func matchArray(s string, ar *types.ArrayValue) eval.Value {
	result := eval.Undef
	ar.Find(func(p eval.Value) bool {
		r := matchPatterns(s, p)
		if r == eval.Undef {
			return false
		}
		result = r
		return true
	})
	return result
}
