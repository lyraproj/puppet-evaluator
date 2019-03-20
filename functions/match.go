package functions

import (
	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
	"github.com/lyraproj/puppet-evaluator/pdsl"
)

func init() {
	px.NewGoFunction2(`match`,
		func(l px.LocalTypes) {
			l.Type(`Patterns`, `Variant[Regexp, String, Type[Pattern], Type[Regexp], Array[Patterns]]`)
		},

		func(d px.Dispatch) {
			d.Param(`String`)
			d.Param(`Patterns`)
			d.Function(func(c px.Context, args []px.Value) px.Value {
				return matchPatterns(args[0].String(), args[1])
			})
		},

		func(d px.Dispatch) {
			d.Param(`Array[String]`)
			d.Param(`Patterns`)
			d.Function(func(c px.Context, args []px.Value) px.Value {
				p := args[1]
				return args[0].(*types.Array).Map(func(arg px.Value) px.Value {
					return matchPatterns(arg.String(), p)
				})
			})
		},
	)
}

func matchPatterns(s string, v px.Value) px.Value {
	switch v := v.(type) {
	case *types.Regexp:
		return matchRegexp(s, v)
	case *types.PatternType:
		return matchArray(s, v.Patterns())
	case *types.RegexpType:
		return matchRegexp(s, types.WrapRegexp(v.PatternString()))
	case *types.Array:
		return matchArray(s, v)
	default:
		return matchRegexp(s, types.WrapRegexp(v.String()))
	}
}

func matchRegexp(s string, rx *types.Regexp) px.Value {
	if rx.PatternString() == `` {
		panic(px.Error(pdsl.MissingRegexpInType, issue.NoArgs))
	}

	g := rx.Match(s)
	if g == nil {
		return px.Undef
	}
	rs := make([]px.Value, len(g))
	for i, s := range g {
		rs[i] = types.WrapString(s)
	}
	return types.WrapValues(rs)
}

func matchArray(s string, ar *types.Array) px.Value {
	result := px.Undef
	ar.Find(func(p px.Value) bool {
		r := matchPatterns(s, p)
		if r == px.Undef {
			return false
		}
		result = r
		return true
	})
	return result
}
