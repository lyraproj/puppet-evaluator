package utils

import (
	"unicode/utf8"
)

type StringReader struct {
	p int
	l int
	c int
	s string
}

func NewStringReader(s string) *StringReader {
	return &StringReader{p: 0, l: 1, c: 0, s: s}
}

func (r *StringReader) Next() rune {
	if r.p >= len(r.s) {
		if r.p == len(r.s) {
			r.p++
			r.c++
		}
		return 0
	}
	c := rune(r.s[r.p])
	if c < utf8.RuneSelf {
		r.p++
		if c == '\n' {
			r.l++
			r.c = 1
		}
		r.c++
	} else {
		var size int
		c, size = utf8.DecodeRuneInString(r.s[r.p:])
		if c != utf8.RuneError {
			r.p += size
			r.c++
		}
	}
	return c
}

func (r *StringReader) Column() int {
	return r.c
}

func (r *StringReader) Line() int {
	return r.l
}

func (r *StringReader) Pos() int {
	return r.p
}

func (r *StringReader) Rewind() {
	r.p = 0
	r.l = 1
	r.c = 1
}
