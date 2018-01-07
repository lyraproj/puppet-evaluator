package types

import (
	"bytes"
	. "io"

	. "github.com/puppetlabs/go-evaluator/errors"
	. "github.com/puppetlabs/go-evaluator/evaluator"
	. "github.com/puppetlabs/go-evaluator/semver"
)

type (
	SemVerType struct {
		vRange *VersionRange
	}

	SemVerValue SemVerType
)

var semVerType_DEFAULT = &SemVerType{MATCH_ALL}

func DefaultSemVerType() *SemVerType {
	return semVerType_DEFAULT
}

func NewSemVerType(vr *VersionRange) *SemVerType {
	if vr.Equals(MATCH_ALL) {
		return DefaultSemVerType()
	}
	return &SemVerType{vr}
}

func NewSemVerType2(limits ...PValue) *SemVerType {
	argc := len(limits)
	if argc == 0 {
		return DefaultSemVerType()
	}

	var finalRange *VersionRange
	for idx, arg := range limits {
		var rng *VersionRange
		str, ok := arg.(*StringValue)
		if ok {
			var err error
			rng, err = ParseVersionRange(str.String())
			if err != nil {
				panic(NewIllegalArgument(`SemVer[]`, idx, err.Error()))
			}
		} else {
			rv, ok := arg.(*SemVerRangeValue)
			if !ok {
				panic(NewIllegalArgumentType2(`SemVer[]`, idx, `Variant[String,SemVerRange]`, limits[0]))
			}
			rng = rv.VersionRange()
		}
		if finalRange == nil {
			finalRange = rng
		} else {
			finalRange = finalRange.Merge(rng)
		}
	}
	return NewSemVerType(finalRange)
}

func (t *SemVerType) Default() PType {
	return semVerType_DEFAULT
}

func (t *SemVerType) Equals(o interface{}, g Guard) bool {
	_, ok := o.(*SemVerType)
	return ok
}

func (t *SemVerType) Name() string {
	return `SemVer`
}

func (t *SemVerType) String() string {
	return ToString2(t, NONE)
}

func (t *SemVerType) IsAssignable(o PType, g Guard) bool {
	if vt, ok := o.(*SemVerType); ok {
		return vt.vRange.IsAsRestrictiveAs(t.vRange)
	}
	return false
}

func (t *SemVerType) IsInstance(o PValue, g Guard) bool {
	if v, ok := o.(*SemVerValue); ok {
		return t.vRange.Includes(v.Version())
	}
	return false
}

func (t *SemVerType) Parameters() []PValue {
	if t.vRange.Equals(MATCH_ALL) {
		return EMPTY_VALUES
	}
	return []PValue{WrapString(t.vRange.String())}
}

func (t *SemVerType) ToString(b Writer, s FormatContext, g RDetect) {
	TypeToString(t, b, s, g)
}

func (t *SemVerType) Type() PType {
	return &TypeType{t}
}

func WrapSemVer(val *Version) *SemVerValue {
	return (*SemVerValue)(NewSemVerType(ExactVersionRange(val)))
}

func (v *SemVerValue) Version() *Version {
	return v.vRange.StartVersion()
}

func (v *SemVerValue) Equals(o interface{}, g Guard) bool {
	if ov, ok := o.(*SemVerValue); ok {
		return v.Version().Equals(ov.Version())
	}
	return false
}

func (v *SemVerValue) String() string {
	return v.Version().String()
}

func (v *SemVerValue) ToKey(b *bytes.Buffer) {
	b.WriteByte(1)
	b.WriteByte(HK_VERSION)
	v.Version().ToString(b)
}

func (v *SemVerValue) ToString(b Writer, s FormatContext, g RDetect) {
	v.Version().ToString(b)
}

func (v *SemVerValue) Type() PType {
	return (*SemVerType)(v)
}
