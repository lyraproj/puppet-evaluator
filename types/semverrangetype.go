package types

import (
	"bytes"
	. "io"

	. "github.com/puppetlabs/go-evaluator/utils"
	. "github.com/puppetlabs/go-evaluator/evaluator"
	. "github.com/puppetlabs/go-evaluator/semver"
)

type (
	SemVerRangeType struct{}

	SemVerRangeValue VersionRange
)

var semVerRangeType_DEFAULT = &SemVerRangeType{}

func DefaultSemVerRangeType() *SemVerRangeType {
	return semVerRangeType_DEFAULT
}

func (t *SemVerRangeType) Accept(v Visitor, g Guard) {
	v(t)
}

func (t *SemVerRangeType) Equals(o interface{}, g Guard) bool {
	_, ok := o.(*SemVerRangeType)
	return ok
}

func (t *SemVerRangeType) Name() string {
	return `SemVerRange`
}

func (t *SemVerRangeType) String() string {
	return `SemVerRange`
}

func (t *SemVerRangeType) IsAssignable(o PType, g Guard) bool {
	_, ok := o.(*SemVerRangeType)
	return ok
}

func (t *SemVerRangeType) IsInstance(o PValue, g Guard) bool {
	_, ok := o.(*SemVerRangeValue)
	return ok
}

func (t *SemVerRangeType) ToString(b Writer, s FormatContext, g RDetect) {
	TypeToString(t, b, s, g)
}

func (t *SemVerRangeType) Type() PType {
	return &TypeType{t}
}

func WrapSemVerRange(val *VersionRange) *SemVerRangeValue {
	return (*SemVerRangeValue)(val)
}

func (bv *SemVerRangeValue) VersionRange() *VersionRange {
	return (*VersionRange)(bv)
}

func (bv *SemVerRangeValue) Equals(o interface{}, g Guard) bool {
	if ov, ok := o.(*SemVerRangeValue); ok {
		return (*VersionRange)(bv).Equals((*VersionRange)(ov))
	}
	return false
}

func (bv *SemVerRangeValue) String() string {
	return ToString2(bv, NONE)
}

func (bv *SemVerRangeValue) ToString(b Writer, s FormatContext, g RDetect) {
	f := GetFormat(s.FormatMap(), bv.Type())
	vr := (*VersionRange)(bv)
	switch f.FormatChar() {
	case 'p':
		if f.IsAlt() {
			PuppetQuote(b, vr.NormalizedString())
		} else {
			PuppetQuote(b, vr.String())
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
	(*VersionRange)(bv).ToString(b)
}

func (bv *SemVerRangeValue) Type() PType {
	return DefaultSemVerRangeType()
}
