package types

import (
	"bytes"
	"io"

	"github.com/puppetlabs/go-evaluator/errors"
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/semver"
)

type (
	SemVerType struct {
		vRange *semver.VersionRange
	}

	SemVerValue SemVerType
)

var semVerType_DEFAULT = &SemVerType{semver.MATCH_ALL}

var SemVer_Type eval.ObjectType

func init() {
	SemVer_Type = newObjectType(`Pcore::SemVerType`,
		`Pcore::ScalarType {
	attributes => {
		ranges => {
      type => Array[Variant[SemVerRange,String[1]]],
      value => []
    }
	}
}`, func(ctx eval.EvalContext, args []eval.PValue) eval.PValue {
			return NewSemVerType2(args...)
		})
}

func DefaultSemVerType() *SemVerType {
	return semVerType_DEFAULT
}

func NewSemVerType(vr *semver.VersionRange) *SemVerType {
	if vr.Equals(semver.MATCH_ALL) {
		return DefaultSemVerType()
	}
	return &SemVerType{vr}
}

func NewSemVerType2(limits ...eval.PValue) *SemVerType {
	return NewSemVerType3(WrapArray(limits))
}

func NewSemVerType3(limits eval.IndexedValue) *SemVerType {
	argc := limits.Len()
	if argc == 0 {
		return DefaultSemVerType()
	}

	if argc == 1 {
		if ranges, ok := limits.At(0).(eval.IndexedValue); ok {
			return NewSemVerType3(ranges)
		}
	}

	var finalRange *semver.VersionRange
	limits.EachWithIndex(func(arg eval.PValue, idx int) {
		var rng *semver.VersionRange
		str, ok := arg.(*StringValue)
		if ok {
			var err error
			rng, err = semver.ParseVersionRange(str.String())
			if err != nil {
				panic(errors.NewIllegalArgument(`SemVer[]`, idx, err.Error()))
			}
		} else {
			rv, ok := arg.(*SemVerRangeValue)
			if !ok {
				panic(NewIllegalArgumentType2(`SemVer[]`, idx, `Variant[String,SemVerRange]`, arg))
			}
			rng = rv.VersionRange()
		}
		if finalRange == nil {
			finalRange = rng
		} else {
			finalRange = finalRange.Merge(rng)
		}
	})
	return NewSemVerType(finalRange)
}

func (t *SemVerType) Accept(v eval.Visitor, g eval.Guard) {
	v(t)
}

func (t *SemVerType) Default() eval.PType {
	return semVerType_DEFAULT
}

func (t *SemVerType) Equals(o interface{}, g eval.Guard) bool {
	_, ok := o.(*SemVerType)
	return ok
}

func (t *SemVerType) Get(key string) (eval.PValue, bool) {
	switch key {
	case `ranges`:
		return WrapArray(t.Parameters()), true
	default:
		return nil, false
	}
}

func (t *SemVerType) MetaType() eval.ObjectType {
	return SemVer_Type
}

func (t *SemVerType) Name() string {
	return `SemVer`
}

func (t *SemVerType) String() string {
	return eval.ToString2(t, NONE)
}

func (t *SemVerType) IsAssignable(o eval.PType, g eval.Guard) bool {
	if vt, ok := o.(*SemVerType); ok {
		return vt.vRange.IsAsRestrictiveAs(t.vRange)
	}
	return false
}

func (t *SemVerType) IsInstance(o eval.PValue, g eval.Guard) bool {
	if v, ok := o.(*SemVerValue); ok {
		return t.vRange.Includes(v.Version())
	}
	return false
}

func (t *SemVerType) Parameters() []eval.PValue {
	if t.vRange.Equals(semver.MATCH_ALL) {
		return eval.EMPTY_VALUES
	}
	return []eval.PValue{WrapString(t.vRange.String())}
}

func (t *SemVerType) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *SemVerType) Type() eval.PType {
	return &TypeType{t}
}

func WrapSemVer(val *semver.Version) *SemVerValue {
	return (*SemVerValue)(NewSemVerType(semver.ExactVersionRange(val)))
}

func (v *SemVerValue) Version() *semver.Version {
	return v.vRange.StartVersion()
}

func (v *SemVerValue) Equals(o interface{}, g eval.Guard) bool {
	if ov, ok := o.(*SemVerValue); ok {
		return v.Version().Equals(ov.Version())
	}
	return false
}

func (v *SemVerValue) SerializationString() string {
	return v.String()
}

func (v *SemVerValue) String() string {
	return v.Version().String()
}

func (v *SemVerValue) ToKey(b *bytes.Buffer) {
	b.WriteByte(1)
	b.WriteByte(HK_VERSION)
	v.Version().ToString(b)
}

func (v *SemVerValue) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	v.Version().ToString(b)
}

func (v *SemVerValue) Type() eval.PType {
	return (*SemVerType)(v)
}
