package evaluator

import (
	"io"
	"regexp"
)

type (
	FormatMap KeyedValue

	Format interface {
		PValue
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
		UnsupportedFormat(t PType, supportedFormats string, actualFormat Format) error
	}
)

var FORMAT_PATTERN = regexp.MustCompile(`\A%([\s\[+#0{<(|-]*)([1-9][0-9]*)?(?:\.([0-9]+))?([a-zA-Z])\z`)

var DEFAULT_FORMAT Format
var DEFAULT_FORMAT_CONTEXT FormatContext

var NewFormat func(format string) Format
var NewIndentation func(indenting bool, level int) Indentation
var NewFormatContext func(t PType, format Format, indentation Indentation) FormatContext
var NewFormatContext2 func(indentation Indentation, formatMap FormatMap) FormatContext
var NewFormatContext3 func(value PValue, format PValue) (FormatContext, error)

func GetFormat(f FormatMap, t PType) Format {
	v := f.Entries().Iterator().Find(func(ev PValue) bool {
		entry := ev.(EntryValue)
		return IsAssignable(entry.Key().(PType), t)
	})
	if v != UNDEF {
		return v.(EntryValue).Value().(Format)
	}
	return DEFAULT_FORMAT
}
