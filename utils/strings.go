package utils

import (
	"bytes"
	"fmt"
	. "io"
	"regexp"

	"unicode/utf8"

	"strings"

	"github.com/puppetlabs/go-parser/parser"
)

func AllStrings(strings []string, predicate func(str string) bool) bool {
	for _, v := range strings {
		if !predicate(v) {
			return false
		}
	}
	return true
}

// Returns true if strings contains str
func ContainsString(strings []string, str string) bool {
	if str != `` {
		for _, v := range strings {
			if v == str {
				return true
			}
		}
	}
	return false
}

// Returns true if strings contains all entries in other
func ContainsAllStrings(strings []string, other []string) bool {
	for _, str := range other {
		if !ContainsString(strings, str) {
			return false
		}
	}
	return true
}

// Returns true if the string represents a base 10 integer
func IsDecimalInteger(s string) bool {
	if len(s) > 0 {
		for _, c := range s {
			if c < '0' || c > '9' {
				return false
			}
		}
		return true
	}
	return false
}

// Returns true if at least one of the regexps matches str
func MatchesString(regexps []*regexp.Regexp, str string) bool {
	if str != `` {
		for _, v := range regexps {
			if v.MatchString(str) {
				return true
			}
		}
	}
	return false
}

// Returns true if all strings are matched by at least one of the regexps
func MatchesAllStrings(regexps []*regexp.Regexp, strings []string) bool {
	for _, str := range strings {
		if !MatchesString(regexps, str) {
			return false
		}
	}
	return true
}

// Creates a new slice where all duplicate strings in the given slice have been removed. Order is retained
func Unique(strings []string) []string {
	top := len(strings)
	if top < 2 {
		return strings
	}
	exists := make(map[string]bool, top)
	result := make([]string, 0, top)

	for _, v := range strings {
		if !exists[v] {
			exists[v] = true
			result = append(result, v)
		}
	}
	return result
}

func CapitalizeSegment(segment string) string {
	b := bytes.NewBufferString(``)
	capitalizeSegment(b, segment)
	return b.String()
}

func capitalizeSegment(b Writer, segment string) {
	_, s := utf8.DecodeRuneInString(segment)
	if s > 0 {
		if s == len(segment) {
			WriteString(b, strings.ToUpper(segment))
		} else {
			WriteString(b, strings.ToUpper(segment[:s]))
			WriteString(b, strings.ToLower(segment[s:]))
		}
	}
}

var COLON_SPLIT = regexp.MustCompile(`::`)

func CapitalizeSegments(segment string) string {
	segments := COLON_SPLIT.Split(segment, -1)
	top := len(segments)
	if top > 0 {
		b := bytes.NewBufferString(``)
		capitalizeSegment(b, segments[0])
		for idx := 1; idx < top; idx++ {
			WriteString(b, `::`)
			capitalizeSegment(b, segments[idx])
		}
		return b.String()
	}
	return ``
}

func RegexpQuote(b Writer, str string) {
	WriteByte(b, '/')
	for _, c := range str {
		switch c {
		case '\t':
			WriteString(b, `\t`)
		case '\n':
			WriteString(b, `\n`)
		case '\r':
			WriteString(b, `\r`)
		case '/':
			WriteString(b, `\/`)
		case '\\':
			WriteString(b, `\\`)
		default:
			if c < 0x20 {
				fmt.Fprintf(b, `\u{%X}`, c)
			} else {
				WriteRune(b, c)
			}
		}
	}
	WriteByte(b, '/')
}

func PuppetQuote(w Writer, str string) {
	r := parser.NewStringReader(str)
	b, ok := w.(*bytes.Buffer)
	if !ok {
		b = bytes.NewBufferString(``)
		defer func() {
			w.Write(b.Bytes())
		}()
	}

	WriteByte(b, '\'')
	escaped := false
	for c, start := r.Next(); c != 0; c, _ = r.Next() {
		if c < 0x20 {
			r.SetPos(start)
			b.Reset()
			puppetDoubleQuote(r, b)
			return
		}

		if escaped {
			WriteByte(b, '\\')
			WriteRune(b, c)
			escaped = false
			continue
		}

		switch c {
		case '\'':
			WriteString(b, `\'`)
		case '\\':
			escaped = true
		default:
			WriteRune(b, c)
		}
	}
	if escaped {
		WriteByte(b, '\\')
	}
	WriteByte(b, '\'')
}

func puppetDoubleQuote(r parser.StringReader, b Writer) {
	WriteByte(b, '"')
	for c, _ := r.Next(); c != 0; c, _ = r.Next() {
		switch c {
		case '\t':
			WriteString(b, `\t`)
		case '\n':
			WriteString(b, `\n`)
		case '\r':
			WriteString(b, `\r`)
		case '"':
			WriteString(b, `\"`)
		case '\\':
			WriteString(b, `\\`)
		case '$':
			WriteString(b, `\$`)
		default:
			if c < 0x20 {
				fmt.Fprintf(b, `\u{%X}`, c)
			} else {
				WriteRune(b, c)
			}
		}
	}
	WriteByte(b, '"')
}

func WriteByte(b Writer, v byte) {
	b.Write([]byte{v})
}

func WriteRune(b Writer, v rune) {
	if v < utf8.RuneSelf {
		WriteByte(b, byte(v))
	} else {
		buf := make([]byte, utf8.UTFMax)
		n := utf8.EncodeRune(buf, v)
		b.Write(buf[:n])
	}
}
