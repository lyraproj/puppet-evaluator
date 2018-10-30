package types

import (
	"bytes"
	"encoding/base64"
	"io"
	"unicode/utf8"

	"fmt"
	"github.com/puppetlabs/go-evaluator/errors"
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-issues/issue"
	"reflect"
	"io/ioutil"
	"os"
)

var binaryType_DEFAULT = &BinaryType{}

var Binary_Type eval.ObjectType

func init() {
	Binary_Type = newObjectType(`Pcore::BinaryType`, `Pcore::AnyType{}`, func(ctx eval.Context, args []eval.Value) eval.Value {
		return DefaultBinaryType()
	})

	newGoConstructor2(`Binary`,
		func(t eval.LocalTypes) {
			t.Type(`ByteInteger`, `Integer[0,255]`)
			t.Type(`Encoding`, `Enum['%b', '%u', '%B', '%s', '%r']`)
			t.Type(`StringHash`, `Struct[value => String, format => Optional[Encoding]]`)
			t.Type(`ArrayHash`, `Struct[value => Array[ByteInteger]]`)
		},

		func(d eval.Dispatch) {
			d.Param(`String`)
			d.OptionalParam(`Encoding`)
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				str := args[0].String()
				f := `%B`
				if len(args) > 1 {
					f = args[1].String()
				}
				return BinaryFromString(str, f)
			})
		},

		func(d eval.Dispatch) {
			d.Param(`Array[ByteInteger]`)
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				return BinaryFromArray(args[0].(eval.List))
			})
		},

		func(d eval.Dispatch) {
			d.Param(`StringHash`)
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				hv := args[0].(eval.OrderedMap)
				return BinaryFromString(hv.Get5(`value`, eval.UNDEF).String(), hv.Get5(`format`, eval.UNDEF).String())
			})
		},

		func(d eval.Dispatch) {
			d.Param(`ArrayHash`)
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				return BinaryFromArray(args[0].(eval.List))
			})
		},
	)
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

func (t *BinaryType) IsAssignable(o eval.Type, g eval.Guard) bool {
	_, ok := o.(*BinaryType)
	return ok
}

func (t *BinaryType) IsInstance(o eval.Value, g eval.Guard) bool {
	_, ok := o.(*BinaryValue)
	return ok
}

func (t *BinaryType) MetaType() eval.ObjectType {
	return Binary_Type
}

func (t *BinaryType) Name() string {
	return `Binary`
}

func (t *BinaryType) ReflectType() (reflect.Type, bool) {
	return reflect.TypeOf([]byte{}), true
}

func (t *BinaryType) String() string {
	return `Binary`
}

func (t *BinaryType) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *BinaryType) Type() eval.Type {
	return &TypeType{t}
}

func WrapBinary(val []byte) *BinaryValue {
	return &BinaryValue{val}
}

// BinaryFromFile opens file appointed by the given path for reading and returns
// its contents as a Binary. The function will panic with an issue.Reported unless
// the operation succeeds.
func BinaryFromFile(c eval.Context, path string) *BinaryValue {
	if bf, ok := BinaryFromFile2(c, path); ok {
		return bf
	}
	panic(eval.Error(eval.EVAL_FILE_NOT_FOUND, issue.H{`path`: path}))
}

// BinaryFromFile2 opens file appointed by the given path for reading and returns
// its contents as a Binary together with a boolean indicating if the file was
// found or not.
//
// The function will only return false if the given file does not exist. It will panic
// with an issue.Reported on all other errors.
func BinaryFromFile2(c eval.Context, path string) (*BinaryValue, bool) {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		stat, serr := os.Stat(path)
		if serr != nil {
			if os.IsNotExist(serr) {
				return nil, false
			}
			if os.IsPermission(serr) {
				panic(eval.Error(eval.EVAL_FILE_READ_DENIED, issue.H{`path`: path}))
			}
		} else {
			if stat.IsDir() {
				panic(eval.Error(eval.EVAL_IS_DIRECTORY, issue.H{`path`: path}))
			}
		}
		panic(eval.Error(eval.EVAL_FAILURE, issue.H{`message`: err.Error()}))
	}
	return WrapBinary(bytes), true
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

func BinaryFromArray(array eval.List) *BinaryValue {
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

func (bv *BinaryValue) Reflect(c eval.Context) reflect.Value {
	return reflect.ValueOf(bv.bytes)
}

func (bv *BinaryValue) ReflectTo(c eval.Context, value reflect.Value) {
	assertKind(value, reflect.Slice)
	switch value.Type().Elem().Kind() {
	case reflect.Int8, reflect.Uint8:
		value.SetBytes(bv.bytes)
	case reflect.Interface:
		value.Set(reflect.ValueOf(bv.bytes))
	default:
		panic(eval.Error(eval.EVAL_ATTEMPT_TO_SET_WRONG_KIND, issue.H{`expected`: `[]byte`, `actual`: fmt.Sprintf(`[]%s`, value.Kind())}))
	}
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

func (bv *BinaryValue) Type() eval.Type {
	return DefaultBinaryType()
}

func (bv *BinaryValue) Bytes() []byte {
	return bv.bytes
}
