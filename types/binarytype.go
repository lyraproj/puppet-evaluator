package types

import (
	"bytes"
	"encoding/base64"
	"io"
	"unicode/utf8"

	"github.com/puppetlabs/go-evaluator/errors"
	"github.com/puppetlabs/go-evaluator/eval"
)

var binaryType_DEFAULT = &BinaryType{}

var Binary_Type eval.ObjectType

func init() {
	Binary_Type = newObjectType(`Pcore::BinaryType`, `Pcore::AnyType{}`, func(ctx eval.EvalContext, args []eval.PValue) eval.PValue {
		return DefaultBinaryType()
	})
}

type (
	BinaryType struct{}

	// BinaryValue keeps only the value because the type is known and not parameterized
	BinaryValue struct {
		bytes []byte
	}
)

func DefaultBinaryType() *BinaryType {
	return binaryType_DEFAULT
}

func (t *BinaryType) Accept(v eval.Visitor, g eval.Guard) {
	v(t)
}

func (t *BinaryType) Equals(o interface{}, g eval.Guard) bool {
	_, ok := o.(*BinaryType)
	return ok
}

func (t *BinaryType) IsAssignable(o eval.PType, g eval.Guard) bool {
	_, ok := o.(*BinaryType)
	return ok
}

func (t *BinaryType) IsInstance(o eval.PValue, g eval.Guard) bool {
	_, ok := o.(*BinaryValue)
	return ok
}

func (t *BinaryType) MetaType() eval.ObjectType {
	return Binary_Type
}

func (t *BinaryType) Name() string {
	return `Binary`
}

func (t *BinaryType) String() string {
	return `Binary`
}

func (t *BinaryType) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *BinaryType) Type() eval.PType {
	return &TypeType{t}
}

func WrapBinary(val []byte) *BinaryValue {
	return &BinaryValue{val}
}

func BinaryFromString(str string, f string) *BinaryValue {
	var bytes []byte
	var err error

	switch f {
	case `%b`:
		bytes, err = base64.StdEncoding.DecodeString(str)
	case `%u`:
		bytes, err = base64.URLEncoding.DecodeString(str)
	case `%B`:
		bytes, err = base64.StdEncoding.Strict().DecodeString(str)
	case `%s`:
		if !utf8.ValidString(str) {
			panic(errors.NewIllegalArgument(`BinaryFromString`, 0, `The given string is not valid utf8. Cannot create a Binary UTF-8 representation`))
		}
		bytes = []byte(str)
	case `%r`:
		bytes = []byte(str)
	default:
		panic(errors.NewIllegalArgument(`BinaryFromString`, 1, `unsupported format specifier`))
	}
	if err == nil {
		return WrapBinary(bytes)
	}
	panic(errors.NewIllegalArgument(`BinaryFromString`, 0, err.Error()))
}

func BinaryFromArray(array eval.IndexedValue) *BinaryValue {
	top := array.Len()
	result := make([]byte, top)
	for idx := 0; idx < top; idx++ {
		if v, ok := toInt(array.At(idx)); ok && 0 <= v && v <= 255 {
			result[idx] = byte(v)
			continue
		}
		panic(errors.NewIllegalArgument(`Binary`, 0, `The given array is not all integers between 0 and 255`))
	}
	return WrapBinary(result)
}

func (bv *BinaryValue) Equals(o interface{}, g eval.Guard) bool {
	if ov, ok := o.(*BinaryValue); ok {
		return bytes.Equal(bv.bytes, ov.bytes)
	}
	return false
}

func (bv *BinaryValue) SerializationString() string {
	return base64.StdEncoding.Strict().EncodeToString(bv.bytes)
}

func (bv *BinaryValue) String() string {
	return eval.ToString2(bv, NONE)
}

func (bv *BinaryValue) ToKey(b *bytes.Buffer) {
	b.WriteByte(0)
	b.WriteByte(HK_BINARY)
	b.Write(bv.bytes)
}

func (bv *BinaryValue) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	f := eval.GetFormat(s.FormatMap(), bv.Type())
	var str string
	switch f.FormatChar() {
	case 's':
		if !utf8.Valid(bv.bytes) {
			panic(errors.GenericError(`binary data is not valid UTF-8`))
		}
		str = string(bv.bytes)
	case 'p':
		str = `Binary('` + base64.StdEncoding.EncodeToString(bv.bytes) + `')`
	case 'b':
		str = base64.StdEncoding.EncodeToString(bv.bytes) + "\n"
	case 'B':
		str = base64.StdEncoding.Strict().EncodeToString(bv.bytes)
	case 'u':
		str = base64.URLEncoding.EncodeToString(bv.bytes)
	case 't':
		str = `Binary`
	case 'T':
		str = `BINARY`
	default:
		panic(s.UnsupportedFormat(bv.Type(), `bButTsp`, f))
	}
	f.ApplyStringFlags(b, str, f.IsAlt())
}

func (bv *BinaryValue) Type() eval.PType {
	return DefaultBinaryType()
}

func (bv *BinaryValue) Bytes() []byte {
	return bv.bytes
}
