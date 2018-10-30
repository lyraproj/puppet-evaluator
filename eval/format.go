package eval

import (
	"io"
	"regexp"
)

type (
	FormatMap OrderedMap

	Format interface {
		Value
		HasStringFlags() bool
		ApplyStringFlags(b io.Writer, str string, quoted bool)
		Width() int
		Precision() int
		FormatChar() byte
		Plus() byte
		IsAlt() bool
		IsLeft() bool
		IsZeroPad() bool
		LeftDelimiter() byte
		ContainerFormats() FormatMap
		Separator(dflt string) string
		Separator2(dflt string) string
		OrigFormat() string
		ReplaceFormatChar(c byte) Format
		WithoutWidth() Format
	}

	Indentation interface {
		Breaks() bool
		Increase(indenting bool) Indentation
		IsFirst() bool
		IsIndenting() bool
		Indenting(indenting bool) Indentation
		Level() int
		Padding() string
		Subsequent() Indentation
	}

	FormatContext interface {
		Indentation() Indentation
		FormatMap() FormatMap
		Property(key string) (string, bool)
		Properties() map[string]string
		SetProperty(key, value string)
		UnsupportedFormat(t Type, supportedFormats string, actualFormat Format) error
		WithProperties(properties map[string]string) FormatContext
	}
)

var FORMAT_PATTERN = regexp.MustCompile(`\A%([\s\[+#0{<(|-]*)([1-9][0-9]*)?(?:\.([0-9]+))?([a-zA-Z])\z`)

var DEFAULT_FORMAT Format
var DEFAULT_FORMAT_CONTEXT FormatContext
var PRETTY FormatContext

var NewFormat func(format string) Format
var NewIndentation func(indenting bool, level int) Indentation
var NewFormatContext func(t Type, format Format, indentation Indentation) FormatContext
var NewFormatContext2 func(indentation Indentation, formatMap FormatMap, properties map[string]string) FormatContext
var NewFormatContext3 func(value Value, format Value) (FormatContext, error)

func GetFormat(f FormatMap, t Type) Format {
	v := f.Iterator().Find(func(ev Value) bool {
		entry := ev.(MapEntry)
		return IsAssignable(entry.Key().(Type), t)
	})
	if v != UNDEF {
		return v.(MapEntry).Value().(Format)
	}
	return DEFAULT_FORMAT
}
