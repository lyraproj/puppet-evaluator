package types

import (
	"bytes"
	"io"

	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/semver"
	"github.com/puppetlabs/go-evaluator/utils"
	"github.com/puppetlabs/go-evaluator/errors"
)

type (
	SemVerRangeType struct{}

	SemVerRangeValue semver.VersionRange
)

var semVerRangeType_DEFAULT = &SemVerRangeType{}

var SemVerRange_Type eval.ObjectType

func init() {
	SemVerRange_Type = newObjectType(`Pcore::SemVerRangeType`, `Pcore::AnyType {}`, func(ctx eval.Context, args []eval.PValue) eval.PValue {
		return DefaultSemVerRangeType()
	})

	newGoConstructor2(`SemVerRange`,
		func(t eval.LocalTypes) {
			t.Type(`SemVerRangeString`, `String[1]`)
			t.Type(`SemVerRangeHash`, `Struct[min=>Variant[Default,SemVer],Optional[max]=>Variant[Default,SemVer],Optional[exclude_max]=>Boolean]`)
		},

		func(d eval.Dispatch) {
			d.Param(`SemVerRangeString`)
			d.Function(func(c eval.Context, args []eval.PValue) eval.PValue {
				v, err := semver.ParseVersionRange(args[0].String())
				if err != nil {
					panic(errors.NewIllegalArgument(`SemVerRange`, 0, err.Error()))
				}
				return WrapSemVerRange(v)
			})
		},

		func(d eval.Dispatch) {
			d.Param(`Variant[Default,SemVer]`)
			d.Param(`Variant[Default,SemVer]`)
			d.OptionalParam(`Boolean`)
			d.Function(func(c eval.Context, args []eval.PValue) eval.PValue {
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

		func(d eval.Dispatch) {
			d.Param(`SemVerRangeHash`)
			d.Function(func(c eval.Context, args []eval.PValue) eval.PValue {
				hash := args[0].(*HashValue)
				start := hash.Get5(`min`, nil).(*SemVerValue).Version()

				var end *semver.Version
				ev := hash.Get5(`max`, nil)
				if ev == nil {
					end = semver.MAX
				} else {
					end = ev.(*SemVerValue).Version()
				}

				excludeEnd := false
				ev = hash.Get5(`excludeMax`, nil)
				if ev != nil {
					excludeEnd = ev.(*BooleanValue).Bool()
				}
				return WrapSemVerRange(semver.FromVersions(start, false, end, excludeEnd))
			})
		},
	)
}

func DefaultSemVerRangeType() *SemVerRangeType {
	return semVerRangeType_DEFAULT
}

func (t *SemVerRangeType) Accept(v eval.Visitor, g eval.Guard) {
	v(t)
}

func (t *SemVerRangeType) Equals(o interface{}, g eval.Guard) bool {
	_, ok := o.(*SemVerRangeType)
	return ok
}

func (t *SemVerRangeType) MetaType() eval.ObjectType {
	return SemVerRange_Type
}

func (t *SemVerRangeType) Name() string {
	return `SemVerRange`
}

func (t *SemVerRangeType) String() string {
	return `SemVerRange`
}

func (t *SemVerRangeType) IsAssignable(o eval.PType, g eval.Guard) bool {
	_, ok := o.(*SemVerRangeType)
	return ok
}

func (t *SemVerRangeType) IsInstance(c eval.Context, o eval.PValue, g eval.Guard) bool {
	_, ok := o.(*SemVerRangeValue)
	return ok
}

func (t *SemVerRangeType) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *SemVerRangeType) Type() eval.PType {
	return &TypeType{t}
}

func WrapSemVerRange(val *semver.VersionRange) *SemVerRangeValue {
	return (*SemVerRangeValue)(val)
}

func (bv *SemVerRangeValue) VersionRange() *semver.VersionRange {
	return (*semver.VersionRange)(bv)
}

func (bv *SemVerRangeValue) Equals(o interface{}, g eval.Guard) bool {
	if ov, ok := o.(*SemVerRangeValue); ok {
		return (*semver.VersionRange)(bv).Equals((*semver.VersionRange)(ov))
	}
	return false
}

func (bv *SemVerRangeValue) SerializationString() string {
	return bv.String()
}

func (bv *SemVerRangeValue) String() string {
	return eval.ToString2(bv, NONE)
}

func (bv *SemVerRangeValue) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	f := eval.GetFormat(s.FormatMap(), bv.Type())
	vr := (*semver.VersionRange)(bv)
	switch f.FormatChar() {
	case 'p':
		if f.IsAlt() {
			utils.PuppetQuote(b, vr.NormalizedString())
		} else {
			utils.PuppetQuote(b, vr.String())
		}
	case 's':
		if f.IsAlt() {
			vr.ToNormalizedString(b)
		} else {
			vr.ToString(b)
		}
	default:
		panic(s.UnsupportedFormat(bv.Type(), `ps`, f))
	}
}

func (bv *SemVerRangeValue) ToKey(b *bytes.Buffer) {
	b.WriteByte(1)
	b.WriteByte(HK_VERSION_RANGE)
	(*semver.VersionRange)(bv).ToString(b)
}

func (bv *SemVerRangeValue) Type() eval.PType {
	return DefaultSemVerRangeType()
}
