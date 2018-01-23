package functions

import (
	"fmt"
	"strconv"

	. "github.com/puppetlabs/go-evaluator/errors"
	. "github.com/puppetlabs/go-evaluator/evaluator"
	"github.com/puppetlabs/go-evaluator/semver"
	. "github.com/puppetlabs/go-evaluator/types"
)

func callNew(c EvalContext, name string, args []PValue, block Lambda) PValue {
	// Always make an attempt to load the named type
	// TODO: This should be a properly checked load but it currently isn't because some receivers in the PSpec
	// evaluator are not proper types yet.
	Load(c.Loader(), NewTypedName(TYPE, name))

	tn := NewTypedName(CONSTRUCTOR, name)
	if ctor, ok := Load(c.Loader(), tn); ok {
		r := ctor.(Function).Call(c, nil, args...)
		if block != nil {
			r = block.Call(c, nil, r)
		}
		return r
	}
	panic(NewArgumentsError(`new`, fmt.Sprintf(`Creation of new instance of type '%s' is not supported`, name)))
}

func init() {
	NewGoFunction(`new`,
		func(d Dispatch) {
			d.Param(`String`)
			d.RepeatedParam(`Any`)
			d.OptionalBlock(`Callable[1,1]`)
			d.Function2(func(c EvalContext, args []PValue, block Lambda) PValue {
				return callNew(c, args[0].String(), args[1:], block)
			})
		},

		func(d Dispatch) {
			d.Param(`Type`)
			d.RepeatedParam(`Any`)
			d.OptionalBlock(`Callable[1,1]`)
			d.Function2(func(c EvalContext, args []PValue, block Lambda) PValue {
				pt := args[0].(PType)
				return assertType(c, pt, callNew(c, pt.Name(), args[1:], block), nil)
			})
		},
	)

	NewGoConstructor2(`Binary`,
		func(t LocalTypes) {
			t.Type(`ByteInteger`, `Integer[0,255]`)
			t.Type(`Base64Format`, `Enum['%b', '%u', '%B', '%s', '%r']`)
			t.Type(`StringHash`, `Struct[value => String, format => Optional[Base64Format]]`)
			t.Type(`ArrayHash`, `Struct[value => Array[ByteInteger]]`)
		},

		func(d Dispatch) {
			d.Param(`String`)
			d.OptionalParam(`Base64Format`)
			d.Function(func(c EvalContext, args []PValue) PValue {
				str := args[0].String()
				f := `%B`
				if len(args) > 1 {
					f = args[1].String()
				}
				return BinaryFromString(str, f)
			})
		},

		func(d Dispatch) {
			d.Param(`Array[ByteInteger]`)
			d.Function(func(c EvalContext, args []PValue) PValue {
				return BinaryFromArray(args[0].(IndexedValue))
			})
		},

		func(d Dispatch) {
			d.Param(`StringHash`)
			d.Function(func(c EvalContext, args []PValue) PValue {
				hv := args[0].(KeyedValue)
				return BinaryFromString(hv.Get2(`value`, UNDEF).String(), hv.Get2(`format`, UNDEF).String())
			})
		},

		func(d Dispatch) {
			d.Param(`ArrayHash`)
			d.Function(func(c EvalContext, args []PValue) PValue {
				return BinaryFromArray(args[0].(IndexedValue))
			})
		},
	)

	NewGoConstructor2(`Numeric`,
		func(t LocalTypes) {
			t.Type(`Convertible`, `Variant[Undef, Integer, Float, Boolean, String, Timespan, Timestamp]`)
			t.Type(`NamedArgs`, `Struct[from => Convertible, Optional[abs] => Boolean]`)
		},

		func(d Dispatch) {
			d.Param(`NamedArgs`)
			d.Function(func(c EvalContext, args []PValue) PValue {
				h := args[0].(*HashValue)
				n := fromConvertible(h.Get2(`from`, UNDEF))
				a := h.Get2(`abs`, nil)
				if a != nil && a.(*BooleanValue).Bool() {
					n = n.Abs()
				}
				return n
			})
		},

		func(d Dispatch) {
			d.Param(`Convertible`)
			d.OptionalParam(`Boolean`)
			d.Function(func(c EvalContext, args []PValue) PValue {
				n := fromConvertible(args[0])
				if len(args) > 1 && args[1].(*BooleanValue).Bool() {
					n = n.Abs()
				}
				return n
			})
		},
	)

	NewGoConstructor2(`SemVer`,
		func(t LocalTypes) {
			t.Type(`PositiveInteger`, `Integer[0,default]`)
			t.Type(`SemVerQualifier`, `Pattern[/\A(?<part>[0-9A-Za-z-]+)(?:\.\g<part>)*\Z/]`)
			t.Type(`SemVerString`, `String[1]`)
			t.Type(`SemVerHash`, `Struct[major=>PositiveInteger,minor=>PositiveInteger,patch=>PositiveInteger,Optional[prerelease]=>SemVerQualifier,Optional[build]=>SemVerQualifier]`)
		},

		func(d Dispatch) {
			d.Param(`SemVerString`)
			d.Function(func(c EvalContext, args []PValue) PValue {
				v, err := semver.ParseVersion(args[0].String())
				if err != nil {
					panic(NewIllegalArgument(`SemVer`, 0, err.Error()))
				}
				return WrapSemVer(v)
			})
		},

		func(d Dispatch) {
			d.Param(`PositiveInteger`)
			d.Param(`PositiveInteger`)
			d.Param(`PositiveInteger`)
			d.OptionalParam(`SemVerQualifier`)
			d.OptionalParam(`SemVerQualifier`)
			d.Function(func(c EvalContext, args []PValue) PValue {
				argc := len(args)
				major := args[0].(*IntegerValue).Int()
				minor := args[1].(*IntegerValue).Int()
				patch := args[2].(*IntegerValue).Int()
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
					panic(NewArgumentsError(`SemVer`, err.Error()))
				}
				return WrapSemVer(v)
			})
		},

		func(d Dispatch) {
			d.Param(`SemVerHash`)
			d.Function(func(c EvalContext, args []PValue) PValue {
				hash := args[0].(*HashValue)
				major := hash.Get2(`major`, ZERO).(*IntegerValue).Int()
				minor := hash.Get2(`minor`, ZERO).(*IntegerValue).Int()
				patch := hash.Get2(`patch`, ZERO).(*IntegerValue).Int()
				preRelease := ``
				build := ``
				ev := hash.Get2(`prerelease`, nil)
				if ev != nil {
					preRelease = ev.String()
				}
				ev = hash.Get2(`build`, nil)
				if ev != nil {
					build = ev.String()
				}
				v, err := semver.NewVersion3(int(major), int(minor), int(patch), preRelease, build)
				if err != nil {
					panic(NewArgumentsError(`SemVer`, err.Error()))
				}
				return WrapSemVer(v)
			})
		},
	)

	NewGoConstructor2(`SemVerRange`,
		func(t LocalTypes) {
			t.Type(`SemVerRangeString`, `String[1]`)
			t.Type(`SemVerRangeHash`, `Struct[min=>Variant[Default,SemVer],Optional[max]=>Variant[Default,SemVer],Optional[exclude_max]=>Boolean]`)
		},

		func(d Dispatch) {
			d.Param(`SemVerRangeString`)
			d.Function(func(c EvalContext, args []PValue) PValue {
				v, err := semver.ParseVersionRange(args[0].String())
				if err != nil {
					panic(NewIllegalArgument(`SemVerRange`, 0, err.Error()))
				}
				return WrapSemVerRange(v)
			})
		},

		func(d Dispatch) {
			d.Param(`Variant[Default,SemVer]`)
			d.Param(`Variant[Default,SemVer]`)
			d.OptionalParam(`Boolean`)
			d.Function(func(c EvalContext, args []PValue) PValue {
				var start *semver.Version
				if _, ok := args[0].(*DefaultValue); ok {
					start = semver.MIN
				} else {
					start = args[0].(*SemVerValue).Version()
				}
				var end *semver.Version
				if _, ok := args[1].(*DefaultValue); ok {
					end = semver.MAX
				} else {
					end = args[1].(*SemVerValue).Version()
				}
				excludeEnd := false
				if len(args) > 2 {
					excludeEnd = args[2].(*BooleanValue).Bool()
				}
				return WrapSemVerRange(semver.FromVersions(start, false, end, excludeEnd))
			})
		},

		func(d Dispatch) {
			d.Param(`SemVerRangeHash`)
			d.Function(func(c EvalContext, args []PValue) PValue {
				hash := args[0].(*HashValue)
				start := hash.Get2(`min`, nil).(*SemVerValue).Version()

				var end *semver.Version
				ev := hash.Get2(`max`, nil)
				if ev == nil {
					end = semver.MAX
				} else {
					end = ev.(*SemVerValue).Version()
				}

				excludeEnd := false
				ev = hash.Get2(`excludeMax`, nil)
				if ev != nil {
					excludeEnd = ev.(*BooleanValue).Bool()
				}
				return WrapSemVerRange(semver.FromVersions(start, false, end, excludeEnd))
			})
		},
	)

	NewGoConstructor2(`String`,
		func(t LocalTypes) {
			t.Type2(`Format`, NewPatternType([]*RegexpType{NewRegexpTypeR(FORMAT_PATTERN)}))
			t.Type(`ContainerFormat`, `Struct[{
          Optional[format]         => Format,
          Optional[separator]      => String,
          Optional[separator2]     => String,
          Optional[string_formats] => Hash[Type, Format]
        }]`)
			t.Type(`TypeMap`, `Hash[Type, Variant[Format, ContainerFormat]]`)
			t.Type(`Formats`, `Variant[Default, String[1], TypeMap]`)
		},

		func(d Dispatch) {
			d.Param(`Any`)
			d.OptionalParam(`Formats`)
			d.Function(func(c EvalContext, args []PValue) PValue {
				fmt := NONE
				if len(args) > 1 {
					var err error
					fmt, err = NewFormatContext3(args[0], args[1])
					if err != nil {
						panic(NewIllegalArgument(`String`, 1, err.Error()))
					}
				}

				// Convert errors on first argument to argument errors
				defer func() {
					if r := recover(); r != nil {
						if ge, ok := r.(GenericError); ok {
							panic(NewIllegalArgument(`String`, 0, ge.Error()))
						}
						panic(r)
					}
				}()
				return WrapString(ToString2(args[0], fmt))
			})
		},
	)

	NewGoConstructor(`Unit`,
		func(d Dispatch) {
			d.Param(`Any`)
			d.Function(func(c EvalContext, args []PValue) PValue {
				return args[0]
			})
		},
	)

	NewGoConstructor(`Type`,
		func(d Dispatch) {
			d.Param(`String`)
			d.Function(func(c EvalContext, args []PValue) PValue {
				return c.ParseType(args[0])
			})
		},
	)
}

func fromConvertible(c PValue) NumericValue {
	switch c.(type) {
	case *UndefValue:
		panic(`undefined_value`)
	case NumericValue:
		return c.(NumericValue)
	case *TimestampValue:
		return WrapFloat(c.(*TimestampValue).Float())
	case *TimespanValue:
		return WrapFloat(c.(*TimespanValue).Float())
	case *BooleanValue:
		b := c.(*BooleanValue).Bool()
		if b {
			return WrapInteger(1)
		}
		return WrapInteger(0)
	case *StringValue:
		s := c.String()
		if i, err := strconv.ParseInt(s, 0, 64); err == nil {
			return WrapInteger(i)
		}
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return WrapFloat(f)
		}
		if len(s) > 2 && s[0] == '0' && (s[1] == 'b' || s[1] == 'B') {
			if i, err := strconv.ParseInt(s[2:], 2, 64); err == nil {
				return WrapInteger(i)
			}
		}
	}
	panic(NewArgumentsError(`Numeric`, fmt.Sprintf(`Value of type %s cannot be converted to an Number`, c.Type().String())))
}
