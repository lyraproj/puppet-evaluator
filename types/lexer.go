package types

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/lyraproj/puppet-evaluator/utils"
	"unicode"
	"unicode/utf8"
)

type tokenType int

const (
	end = iota
	name
	identifier
	integer
	float
	regexpLiteral
	stringLiteral
	leftBracket
	rightBracket
	leftCurlyBrace
	rightCurlyBrace
	leftParen
	rightParen
	comma
	dot
	equal
	rocket
)

func (t tokenType) String() (s string) {
	switch t {
	case end:
		s = "end"
	case name:
		s = "name"
	case identifier:
		s = "identifier"
	case integer:
		s = "integer"
	case float:
		s = "float"
	case regexpLiteral:
		s = "regexp"
	case stringLiteral:
		s = "string"
	case leftBracket:
		s = "leftBracket"
	case rightBracket:
		s = "rightBracket"
	case leftCurlyBrace:
		s = "leftCurlyBrace"
	case rightCurlyBrace:
		s = "rightCurlyBrace"
	case leftParen:
		s = "leftParen"
	case rightParen:
		s = "rightParen"
	case comma:
		s = "comma"
	case dot:
		s = "dot"
	case rocket:
		s = "rocket"
	default:
		s = "*UNKNOWN TOKEN*"
	}
	return
}

const (
	stNothing = iota
	stSqString
	stDqString
	stRqString
	stSqEscape
	stDqEscape
	stRqEscape
	stHexFirst
	stIntegerFirst
	stInteger
	stSign
	stFraction
	stExponentFirst
	stExponentSign
	stExponent
	stEqStart
	stLineComment
	stTypeNameStart
	stTypeName
	stTypeNameSepStart
	stIdentifierStart
	stIdentifier
	stIdentifierSepStart
)

type token struct {
	s string
	i tokenType
}

func (t token) String() string {
	return fmt.Sprintf("%s: '%s'", t.i.String(), t.s)
}

func scan(sr *utils.StringReader, tf func(t token) error) (err error) {
	buf := bytes.NewBufferString(``)
	state := stNothing

	badToken := func(r rune) error {
		return fmt.Errorf("unexpected character '%c'", r)
	}

	for {
		r := sr.Next()
		if r == utf8.RuneError {
			return errors.New("unicode error")
		}

		switch state {
		case stLineComment:
			if r == '\n' {
				state = stNothing
			}
			if r != 0 {
				continue
			}
		case stEqStart:
			if r == '>' {
				if err = tf(token{`=>`, rocket}); err != nil {
					return nil
				}
				state = stNothing
				continue
			}
			if err = tf(token{`=`, equal}); err != nil {
				return err
			}
			state = stNothing
		case stSqString:
			switch r {
			case '\'':
				if err = tf(token{buf.String(), stringLiteral}); err != nil {
					return err
				}
				buf.Reset()
				state = stNothing
			case '\\':
				state = stSqEscape
			case 0, '\n':
				return errors.New("unterminated string")
			default:
				buf.WriteRune(r)
			}
			continue
		case stDqString:
			switch r {
			case '"':
				if err = tf(token{buf.String(), stringLiteral}); err != nil {
					return err
				}
				buf.Reset()
				state = stNothing
			case '\\':
				state = stDqEscape
			case 0, '\n':
				return errors.New("unterminated string")
			default:
				buf.WriteRune(r)
			}
			continue
		case stRqString:
			switch r {
			case '/':
				if err = tf(token{buf.String(), regexpLiteral}); err != nil {
					return err
				}
				buf.Reset()
				state = stNothing
			case '\\':
				state = stRqEscape
			case 0, '\n':
				return errors.New("unterminated regexp")
			default:
				buf.WriteRune(r)
			}
			continue
		case stSqEscape:
			switch r {
			case 'n':
				r = '\n'
			case 'r':
				r = '\r'
			case 't':
				r = '\t'
			case 0:
				return errors.New("unterminated string")
			case '\\', '\'':
			default:
				return fmt.Errorf("illegal escape '\\%c'", r)
			}
			buf.WriteRune(r)
			state = stSqString
			continue
		case stDqEscape:
			switch r {
			case 'n':
				r = '\n'
			case 'r':
				r = '\r'
			case 't':
				r = '\t'
			case 0:
				return errors.New("unterminated string")
			case '"', '\\':
			default:
				return fmt.Errorf("illegal escape '\\%c'", r)
			}
			buf.WriteRune(r)
			state = stDqString
			continue
		case stRqEscape:
			if r == 0 {
				return errors.New("unterminated regexp")
			}
			if r != '/' {
				buf.WriteByte('\\')
			}
			buf.WriteRune(r)
			state = stRqString
			continue
		case stSign, stInteger, stIntegerFirst, stHexFirst:
			switch r {
			case '0':
				if state == stSign {
					state = stIntegerFirst
				} else {
					state = stInteger
				}
				buf.WriteRune(r)
				continue
			case 'e', 'E':
				if state != stInteger {
					return badToken(r)
				}
				buf.WriteRune(r)
				state = stExponentFirst
				continue
			case 'x', 'X':
				if state != stIntegerFirst {
					return badToken(r)
				}
				state = stHexFirst
				buf.WriteRune(r)
				continue
			case '.':
				if state == stSign || state == stHexFirst {
					return badToken(r)
				}
				buf.WriteRune(r)
				state = stFraction
				continue
			default:
				if r >= '1' && r <= '9' {
					state = stInteger
					buf.WriteRune(r)
					continue
				}
				if state == stSign || state == stHexFirst || unicode.IsLetter(r) {
					return badToken(r)
				}
				if err = tf(token{buf.String(), integer}); err != nil {
					return err
				}
				buf.Reset()
				state = stNothing
			}
		case stFraction:
			if r >= '0' && r <= '9' {
				buf.WriteRune(r)
				continue
			}
			if r == 'e' || r == 'E' {
				buf.WriteRune(r)
				state = stExponentFirst
				continue
			}
			if r == '.' || unicode.IsLetter(r) {
				return badToken(r)
			}
			if err = tf(token{buf.String(), float}); err != nil {
				return err
			}
			buf.Reset()
			state = stNothing
		case stExponent, stExponentSign, stExponentFirst:
			if r == '-' {
				if state != stExponentFirst {
					return badToken(r)
				}
				buf.WriteRune(r)
				state = stExponentSign
				continue
			}
			if r >= '0' && r <= '9' {
				state = stExponent
				buf.WriteRune(r)
				continue
			}
			if state != stExponent || r == '.' || unicode.IsLetter(r) {
				return badToken(r)
			}
			if err = tf(token{buf.String(), float}); err != nil {
				return err
			}
			buf.Reset()
			state = stNothing
		case stTypeName:
			if r == '_' || r >= '0' && r <= '9' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' {
				buf.WriteRune(r)
				continue
			}
			if r == ':' {
				state = stTypeNameSepStart
				buf.WriteRune(r)
				continue
			}
			if err = tf(token{buf.String(), name}); err != nil {
				return err
			}
			buf.Reset()
			state = stNothing
		case stTypeNameSepStart:
			if r == ':' {
				state = stTypeNameStart
				buf.WriteRune(r)
				continue
			}
			return badToken(':')
		case stTypeNameStart:
			if r >= 'A' && r <= 'Z' {
				state = stTypeName
				buf.WriteRune(r)
				continue
			}
			return badToken(r)
		case stIdentifier:
			if r == '_' || r == '-' || r >= '0' && r <= '9' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' {
				buf.WriteRune(r)
				continue
			}
			if r == ':' {
				state = stIdentifierSepStart
				buf.WriteRune(r)
				continue
			}
			if err = tf(token{buf.String(), identifier}); err != nil {
				return err
			}
			buf.Reset()
			state = stNothing
		case stIdentifierSepStart:
			if r == ':' {
				state = stIdentifierStart
				buf.WriteRune(r)
				continue
			}
			return badToken(':')
		case stIdentifierStart:
			if r == '_' || r >= 'a' && r <= 'z' {
				state = stTypeName
				buf.WriteRune(r)
				continue
			}
			return badToken(r)
		}

		if r == 0 {
			break
		}

		switch r {
		case ' ', '\t', '\n':
		case '"':
			state = stDqString
		case '\'':
			state = stSqString
		case '/':
			state = stRqString
		case '#':
			state = stLineComment
		case '{':
			if err = tf(token{string(r), leftCurlyBrace}); err != nil {
				return err
			}
		case '}':
			if err = tf(token{string(r), rightCurlyBrace}); err != nil {
				return err
			}
		case '[':
			if err = tf(token{string(r), leftBracket}); err != nil {
				return err
			}
		case ']':
			if err = tf(token{string(r), rightBracket}); err != nil {
				return err
			}
		case '(':
			if err = tf(token{string(r), leftParen}); err != nil {
				return err
			}
		case ')':
			if err = tf(token{string(r), rightParen}); err != nil {
				return err
			}
		case ',':
			if err = tf(token{string(r), comma}); err != nil {
				return err
			}
		case '.':
			if err = tf(token{string(r), dot}); err != nil {
				return err
			}
		case '=':
			state = stEqStart
		case '-', '+':
			buf.WriteRune(r)
			state = stSign
		case '0':
			buf.WriteRune(r)
			state = stIntegerFirst
		default:
			if r >= '1' && r <= '9' {
				buf.WriteRune(r)
				state = stInteger
			} else if r >= 'A' && r <= 'Z' {
				buf.WriteRune(r)
				state = stTypeName
			} else if r >= 'a' && r <= 'z' {
				buf.WriteRune(r)
				state = stIdentifier
			} else {
				return badToken(r)
			}
		}
	}
	if err = tf(token{``, end}); err != nil {
		return err
	}
	return nil
}
