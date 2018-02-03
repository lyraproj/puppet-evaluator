package functions

import (
	"fmt"
	"strconv"

	"github.com/puppetlabs/go-evaluator/errors"
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/semver"
	"github.com/puppetlabs/go-evaluator/types"
)

func callNew(c eval.EvalContext, name string, args []eval.PValue, block eval.Lambda) eval.PValue {
	// Always make an attempt to load the named type
	// TODO: This should be a properly checked load but it currently isn't because some receivers in the PSpec
	// evaluator are not proper types yet.
	eval.Load(c.Loader(), eval.NewTypedName(eval.TYPE, name))

	tn := eval.NewTypedName(eval.CONSTRUCTOR, name)
	if ctor, ok := eval.Load(c.Loader(), tn); ok {
		r := ctor.(eval.Function).Call(c, nil, args...)
		if block != nil {
			r = block.Call(c, nil, r)
		}
		return r
	}
	panic(errors.NewArgumentsError(`new`, fmt.Sprintf(`Creation of new instance of type '%s' is not supported`, name)))
}

func init() {
	eval.NewGoFunction(`new`,
		func(d eval.Dispatch) {
			d.Param(`String`)
			d.RepeatedParam(`Any`)
			d.OptionalBlock(`Callable[1,1]`)
			d.Function2(func(c eval.EvalContext, args []eval.PValue, block eval.Lambda) eval.PValue {
				return callNew(c, args[0].String(), args[1:], block)
			})
		},

		func(d eval.Dispatch) {
			d.Param(`Type`)
			d.RepeatedParam(`Any`)
			d.OptionalBlock(`Callable[1,1]`)
			d.Function2(func(c eval.EvalContext, args []eval.PValue, block eval.Lambda) eval.PValue {
				pt := args[0].(eval.PType)
				return assertType(c, pt, callNew(c, pt.Name(), args[1:], block), nil)
			})
		},
	)

	eval.NewGoConstructor2(`Binary`,
		func(t eval.LocalTypes) {
			t.Type(`ByteInteger`, `Integer[0,255]`)
			t.Type(`Base64Format`, `Enum['%b', '%u', '%B', '%s', '%r']`)
			t.Type(`StringHash`, `Struct[value => String, format => Optional[Base64Format]]`)
			t.Type(`ArrayHash`, `Struct[value => Array[ByteInteger]]`)
		},

		func(d eval.Dispatch) {
			d.Param(`String`)
			d.OptionalParam(`Base64Format`)
			d.Function(func(c eval.EvalContext, args []eval.PValue) eval.PValue {
				str := args[0].String()
				f := `%B`
				if len(args) > 1 {
					f = args[1].String()
				}
				return types.BinaryFromString(str, f)
			})
		},

		func(d eval.Dispatch) {
			d.Param(`Array[ByteInteger]`)
			d.Function(func(c eval.EvalContext, args []eval.PValue) eval.PValue {
				return types.BinaryFromArray(args[0].(eval.IndexedValue))
			})
		},

		func(d eval.Dispatch) {
			d.Param(`StringHash`)
			d.Function(func(c eval.EvalContext, args []eval.PValue) eval.PValue {
				hv := args[0].(eval.KeyedValue)
				return types.BinaryFromString(hv.Get5(`value`, eval.UNDEF).String(), hv.Get5(`format`, eval.UNDEF).String())
			})
		},

		func(d eval.Dispatch) {
			d.Param(`ArrayHash`)
			d.Function(func(c eval.EvalContext, args []eval.PValue) eval.PValue {
				return types.BinaryFromArray(args[0].(eval.IndexedValue))
			})
		},
	)

	eval.NewGoConstructor2(`Numeric`,
		func(t eval.LocalTypes) {
			t.Type(`Convertible`, `Variant[Undef, Integer, Float, Boolean, String, Timespan, Timestamp]`)
			t.Type(`NamedArgs`, `Struct[from => Convertible, Optional[abs] => Boolean]`)
		},

		func(d eval.Dispatch) {
			d.Param(`NamedArgs`)
			d.Function(func(c eval.EvalContext, args []eval.PValue) eval.PValue {
				h := args[0].(*types.HashValue)
				n := fromConvertible(h.Get5(`from`, eval.UNDEF))
				a := h.Get5(`abs`, nil)
				if a != nil && a.(*types.BooleanValue).Bool() {
					n = n.Abs()
				}
				return n
			})
		},

		func(d eval.Dispatch) {
			d.Param(`Convertible`)
			d.OptionalParam(`Boolean`)
			d.Function(func(c eval.EvalContext, args []eval.PValue) eval.PValue {
				n := fromConvertible(args[0])
				if len(args) > 1 && args[1].(*types.BooleanValue).Bool() {
					n = n.Abs()
				}
				return n
			})
		},
	)

	eval.NewGoConstructor2(`SemVer`,
		func(t eval.LocalTypes) {
			t.Type(`PositiveInteger`, `Integer[0,default]`)
			t.Type(`SemVerQualifier`, `Pattern[/\A(?<part>[0-9A-Za-z-]+)(?:\.\g<part>)*\Z/]`)
			t.Type(`SemVerString`, `String[1]`)
			t.Type(`SemVerHash`, `Struct[major=>PositiveInteger,minor=>PositiveInteger,patch=>PositiveInteger,Optional[prerelease]=>SemVerQualifier,Optional[build]=>SemVerQualifier]`)
		},

		func(d eval.Dispatch) {
			d.Param(`SemVerString`)
			d.Function(func(c eval.EvalContext, args []eval.PValue) eval.PValue {
				v, err := semver.ParseVersion(args[0].String())
				if err != nil {
					panic(errors.NewIllegalArgument(`SemVer`, 0, err.Error()))
				}
				return types.WrapSemVer(v)
			})
		},

		func(d eval.Dispatch) {
			d.Param(`PositiveInteger`)
			d.Param(`PositiveInteger`)
			d.Param(`PositiveInteger`)
			d.OptionalParam(`SemVerQualifier`)
			d.OptionalParam(`SemVerQualifier`)
			d.Function(func(c eval.EvalContext, args []eval.PValue) eval.PValue {
				argc := len(args)
				major := args[0].(*types.IntegerValue).Int()
				minor := args[1].(*types.IntegerValue).Int()
				patch := args[2].(*types.IntegerValue).Int()
				preRelease := ``
				build := ``
				if argc > 3 {
					preRelease = args[3].String()
					if argc > 4 {
						build = args[4].String()
					}
				}
				v, err := semver.NewVersion3(int(major), int(minor), int(patch), preRelease, build)
				if err != nil {
					panic(errors.NewArgumentsError(`SemVer`, err.Error()))
				}
				return types.WrapSemVer(v)
			})
		},

		func(d eval.Dispatch) {
			d.Param(`SemVerHash`)
			d.Function(func(c eval.EvalContext, args []eval.PValue) eval.PValue {
				hash := args[0].(*types.HashValue)
				major := hash.Get5(`major`, types.ZERO).(*types.IntegerValue).Int()
				minor := hash.Get5(`minor`, types.ZERO).(*types.IntegerValue).Int()
				patch := hash.Get5(`patch`, types.ZERO).(*types.IntegerValue).Int()
				preRelease := ``
				build := ``
				ev := hash.Get5(`prerelease`, nil)
				if ev != nil {
					preRelease = ev.String()
				}
				ev = hash.Get5(`build`, nil)
				if ev != nil {
					build = ev.String()
				}
				v, err := semver.NewVersion3(int(major), int(minor), int(patch), preRelease, build)
				if err != nil {
					panic(errors.NewArgumentsError(`SemVer`, err.Error()))
				}
				return types.WrapSemVer(v)
			})
		},
	)

	eval.NewGoConstructor2(`SemVerRange`,
		func(t eval.LocalTypes) {
			t.Type(`SemVerRangeString`, `String[1]`)
			t.Type(`SemVerRangeHash`, `Struct[min=>Variant[Default,SemVer],Optional[max]=>Variant[Default,SemVer],Optional[exclude_max]=>Boolean]`)
		},

		func(d eval.Dispatch) {
			d.Param(`SemVerRangeString`)
			d.Function(func(c eval.EvalContext, args []eval.PValue) eval.PValue {
				v, err := semver.ParseVersionRange(args[0].String())
				if err != nil {
					panic(errors.NewIllegalArgument(`SemVerRange`, 0, err.Error()))
				}
				return types.WrapSemVerRange(v)
			})
		},

		func(d eval.Dispatch) {
			d.Param(`Variant[Default,SemVer]`)
			d.Param(`Variant[Default,SemVer]`)
			d.OptionalParam(`Boolean`)
			d.Function(func(c eval.EvalContext, args []eval.PValue) eval.PValue {
				var start *semver.Version
				if _, ok := args[0].(*types.DefaultValue); ok {
					start = semver.MIN
				} else {
					start = args[0].(*types.SemVerValue).Version()
				}
				var end *semver.Version
				if _, ok := args[1].(*types.DefaultValue); ok {
					end = semver.MAX
				} else {
					end = args[1].(*types.SemVerValue).Version()
				}
				excludeEnd := false
				if len(args) > 2 {
					excludeEnd = args[2].(*types.BooleanValue).Bool()
				}
				return types.WrapSemVerRange(semver.FromVersions(start, false, end, excludeEnd))
			})
		},

		func(d eval.Dispatch) {
			d.Param(`SemVerRangeHash`)
			d.Function(func(c eval.EvalContext, args []eval.PValue) eval.PValue {
				hash := args[0].(*types.HashValue)
				start := hash.Get5(`min`, nil).(*types.SemVerValue).Version()

				var end *semver.Version
				ev := hash.Get5(`max`, nil)
				if ev == nil {
					end = semver.MAX
				} else {
					end = ev.(*types.SemVerValue).Version()
				}

				excludeEnd := false
				ev = hash.Get5(`excludeMax`, nil)
				if ev != nil {
					excludeEnd = ev.(*types.BooleanValue).Bool()
				}
				return types.WrapSemVerRange(semver.FromVersions(start, false, end, excludeEnd))
			})
		},
	)

	eval.NewGoConstructor(`Sensitive`,
		func(d eval.Dispatch) {
			d.Param(`Any`)
			d.Function(func(c eval.EvalContext, args []eval.PValue) eval.PValue {
				return types.WrapSensitive(args[0])
			})
		})

	eval.NewGoConstructor2(`String`,
		func(t eval.LocalTypes) {
			t.Type2(`Format`, types.NewPatternType([]*types.RegexpType{types.NewRegexpTypeR(eval.FORMAT_PATTERN)}))
			t.Type(`ContainerFormat`, `Struct[{
          Optional[format]         => Format,
          Optional[separator]      => String,
          Optional[separator2]     => String,
          Optional[string_formats] => Hash[Type, Format]
        }]`)
			t.Type(`TypeMap`, `Hash[Type, Variant[Format, ContainerFormat]]`)
			t.Type(`Formats`, `Variant[Default, String[1], TypeMap]`)
		},

		func(d eval.Dispatch) {
			d.Param(`Any`)
			d.OptionalParam(`Formats`)
			d.Function(func(c eval.EvalContext, args []eval.PValue) eval.PValue {
				fmt := types.NONE
				if len(args) > 1 {
					var err error
					fmt, err = eval.NewFormatContext3(args[0], args[1])
					if err != nil {
						panic(errors.NewIllegalArgument(`String`, 1, err.Error()))
					}
				}

				// Convert errors on first argument to argument errors
				defer func() {
					if r := recover(); r != nil {
						if ge, ok := r.(errors.GenericError); ok {
							panic(errors.NewIllegalArgument(`String`, 0, ge.Error()))
						}
						panic(r)
					}
				}()
				return types.WrapString(eval.ToString2(args[0], fmt))
			})
		},
	)

	eval.NewGoConstructor(`Unit`,
		func(d eval.Dispatch) {
			d.Param(`Any`)
			d.Function(func(c eval.EvalContext, args []eval.PValue) eval.PValue {
				return args[0]
			})
		},
	)

	eval.NewGoConstructor(`Type`,
		func(d eval.Dispatch) {
			d.Param(`String`)
			d.Function(func(c eval.EvalContext, args []eval.PValue) eval.PValue {
				return c.ParseType(args[0])
			})
		},
	)
}

func fromConvertible(c eval.PValue) eval.NumericValue {
	switch c.(type) {
	case *types.UndefValue:
		panic(`undefined_value`)
	case eval.NumericValue:
		return c.(eval.NumericValue)
	case *types.TimestampValue:
		return types.WrapFloat(c.(*types.TimestampValue).Float())
	case *types.TimespanValue:
		return types.WrapFloat(c.(*types.TimespanValue).Float())
	case *types.BooleanValue:
		b := c.(*types.BooleanValue).Bool()
		if b {
			return types.WrapInteger(1)
		}
		return types.WrapInteger(0)
	case *types.StringValue:
		s := c.String()
		if i, err := strconv.ParseInt(s, 0, 64); err == nil {
			return types.WrapInteger(i)
		}
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return types.WrapFloat(f)
		}
		if len(s) > 2 && s[0] == '0' && (s[1] == 'b' || s[1] == 'B') {
			if i, err := strconv.ParseInt(s[2:], 2, 64); err == nil {
				return types.WrapInteger(i)
			}
		}
	}
	panic(errors.NewArgumentsError(`Numeric`, fmt.Sprintf(`Value of type %s cannot be converted to an Number`, c.Type().String())))
}
