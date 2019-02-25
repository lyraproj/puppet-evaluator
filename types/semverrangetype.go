package types

import (
	"bytes"
	"io"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/puppet-evaluator/errors"
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/utils"
	"github.com/lyraproj/semver/semver"
	"reflect"
)

type (
	SemVerRangeType struct{}

	SemVerRangeValue struct {
		rng semver.VersionRange
	}
)

var semVerRangeType_DEFAULT = &SemVerRangeType{}

var SemVerRangeMetaType eval.ObjectType

func init() {
	SemVerRangeMetaType = newObjectType(`Pcore::SemVerRangeType`, `Pcore::AnyType {}`, func(ctx eval.Context, args []eval.Value) eval.Value {
		return DefaultSemVerRangeType()
	})

	newGoConstructor2(`SemVerRange`,
		func(t eval.LocalTypes) {
			t.Type(`SemVerRangeString`, `String[1]`)
			t.Type(`SemVerRangeHash`, `Struct[min=>Variant[Default,SemVer],Optional[max]=>Variant[Default,SemVer],Optional[exclude_max]=>Boolean]`)
		},

		func(d eval.Dispatch) {
			d.Param(`SemVerRangeString`)
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
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
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				var start semver.Version
				if _, ok := args[0].(*DefaultValue); ok {
					start = semver.Min
				} else {
					start = args[0].(*SemVerValue).Version()
				}
				var end semver.Version
				if _, ok := args[1].(*DefaultValue); ok {
					end = semver.Max
				} else {
					end = args[1].(*SemVerValue).Version()
				}
				excludeEnd := false
				if len(args) > 2 {
					excludeEnd = args[2].(booleanValue).Bool()
				}
				return WrapSemVerRange(semver.FromVersions(start, false, end, excludeEnd))
			})
		},

		func(d eval.Dispatch) {
			d.Param(`SemVerRangeHash`)
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				hash := args[0].(*HashValue)
				start := hash.Get5(`min`, nil).(*SemVerValue).Version()

				var end semver.Version
				ev := hash.Get5(`max`, nil)
				if ev == nil {
					end = semver.Max
				} else {
					end = ev.(*SemVerValue).Version()
				}

				excludeEnd := false
				ev = hash.Get5(`excludeMax`, nil)
				if ev != nil {
					excludeEnd = ev.(booleanValue).Bool()
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
	return SemVerRangeMetaType
}

func (t *SemVerRangeType) Name() string {
	return `SemVerRange`
}

func (t *SemVerRangeType) CanSerializeAsString() bool {
	return true
}

func (t *SemVerRangeType) SerializationString() string {
	return t.String()
}

func (t *SemVerRangeType) String() string {
	return `SemVerRange`
}

func (t *SemVerRangeType) IsAssignable(o eval.Type, g eval.Guard) bool {
	_, ok := o.(*SemVerRangeType)
	return ok
}

func (t *SemVerRangeType) IsInstance(o eval.Value, g eval.Guard) bool {
	_, ok := o.(*SemVerRangeValue)
	return ok
}

func (t *SemVerRangeType) ReflectType(c eval.Context) (reflect.Type, bool) {
	return reflect.TypeOf(semver.MatchAll), true
}

func (t *SemVerRangeType) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *SemVerRangeType) PType() eval.Type {
	return &TypeType{t}
}

func WrapSemVerRange(val semver.VersionRange) *SemVerRangeValue {
	return &SemVerRangeValue{val}
}

func (bv *SemVerRangeValue) VersionRange() semver.VersionRange {
	return bv.rng
}

func (bv *SemVerRangeValue) Equals(o interface{}, g eval.Guard) bool {
	if ov, ok := o.(*SemVerRangeValue); ok {
		return bv.rng.Equals(ov.rng)
	}
	return false
}

func (bv *SemVerRangeValue) Reflect(c eval.Context) reflect.Value {
	return reflect.ValueOf(bv.rng)
}

func (bv *SemVerRangeValue) ReflectTo(c eval.Context, dest reflect.Value) {
	rv := bv.Reflect(c)
	if !rv.Type().AssignableTo(dest.Type()) {
		panic(eval.Error(eval.EVAL_ATTEMPT_TO_SET_WRONG_KIND, issue.H{`expected`: rv.Type().String(), `actual`: dest.Type().String()}))
	}
	dest.Set(rv)
}

func (bv *SemVerRangeValue) CanSerializeAsString() bool {
	return true
}

func (bv *SemVerRangeValue) SerializationString() string {
	return bv.String()
}

func (bv *SemVerRangeValue) String() string {
	return eval.ToString2(bv, NONE)
}

func (bv *SemVerRangeValue) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	f := eval.GetFormat(s.FormatMap(), bv.PType())
	vr := bv.rng
	switch f.FormatChar() {
	case 'p':
		io.WriteString(b, `SemVerRange(`)
		if f.IsAlt() {
			utils.PuppetQuote(b, vr.NormalizedString())
		} else {
			utils.PuppetQuote(b, vr.String())
		}
		io.WriteString(b, `)`)
	case 's':
		if f.IsAlt() {
			vr.ToNormalizedString(b)
		} else {
			vr.ToString(b)
		}
	default:
		panic(s.UnsupportedFormat(bv.PType(), `ps`, f))
	}
}

func (bv *SemVerRangeValue) ToKey(b *bytes.Buffer) {
	b.WriteByte(1)
	b.WriteByte(HK_VERSION_RANGE)
	bv.rng.ToString(b)
}

func (bv *SemVerRangeValue) PType() eval.Type {
	return DefaultSemVerRangeType()
}
