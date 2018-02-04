package types

import (
	"bytes"
	"io"

	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/semver"
	"github.com/puppetlabs/go-evaluator/utils"
)

type (
	SemVerRangeType struct{}

	SemVerRangeValue semver.VersionRange
)

var semVerRangeType_DEFAULT = &SemVerRangeType{}

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

func (t *SemVerRangeType) IsInstance(o eval.PValue, g eval.Guard) bool {
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
