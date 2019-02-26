package parser

import (
	"bytes"
	"fmt"
	"unicode"
)

type TokenType int

const (
	End = TokenType(-(iota + 1))
	TypeName
	Identifier
	Integer
	Float
	String
	Lb
	Rb
	Lc
	Rc
	Lp
	Rp
	Comma
	Dot
	Equal
	Rocket
)

func (t TokenType) String() (s string) {
	switch t {
	case End:
		s = "End"
	case TypeName:
		s = "TypeName"
	case Identifier:
		s = "Identifier"
	case Integer:
		s = "Integer"
	case Float:
		s = "Float"
	case String:
		s = "String"
	case Lb:
		s = "Lb"
	case Rb:
		s = "Rb"
	case Lc:
		s = "Lc"
	case Rc:
		s = "Rc"
	case Lp:
		s = "Lp"
	case Rp:
		s = "Rp"
	case Comma:
		s = "Comma"
	case Dot:
		s = "Dot"
	case Rocket:
		s = "Rocket"
	default:
		s = "*UNKNOWN TOKEN*"
	}
	return
}

const (
	nothing = iota
	sqString
	dqString
	sqEscape
	dqEscape
	hexFirst
	integerFirst
	integer
	sign
	fraction
	exponentFirst
	exponentSign
	exponent
	eqStart
	lineComment
	typeNameStart
	typeName
	typeNameSepStart
	identifierStart
	identifier
	identifierSepStart
)

type Token struct {
	s string
	i TokenType
}

func (t Token) String() string {
	return fmt.Sprintf("%s: '%s'", t.i.String(), t.s)
}

func Scan(src string, tf func(t Token)) {
	buf := bytes.NewBufferString(``)
	state := nothing
	line := 1
	pos := 0

	syntaxError := func(line, pos int, msg string) error {
		return fmt.Errorf("error at %d:%d, %s", line, pos, msg)
	}

	badToken := func(r rune) error {
		return syntaxError(line, pos, fmt.Sprintf("illegal token '%c'", r))
	}

	for _, r := range src {
		pos++

		switch state {
		case lineComment:
			if r == '\n' {
				line++
				pos = 0
				state = nothing
			}
			continue
		case eqStart:
			if r == '>' {
				tf(Token{`=>`, Rocket})
				state = nothing
				continue
			}
			tf(Token{`=`, Equal})
			state = nothing
		case sqString:
			switch r {
			case '\'':
				tf(Token{buf.String(), String})
				buf.Reset()
				state = nothing
			case '\\':
				state = sqEscape
			case '\n':
				panic(syntaxError(line, pos, "unterminated string"))
			default:
				buf.WriteRune(r)
			}
			continue
		case sqEscape:
			switch(r) {
			case 'n':
				r = '\n'
			case 'r':
				r = '\r'
			case 't':
				r = '\t'
			case '\\', '\'':
			default:
				panic(syntaxError(line, pos, fmt.Sprintf("illegal escape '\\%c'", r)))
			}
			buf.WriteRune(r)
			state = sqString
			continue
		case dqEscape:
			switch(r) {
			case 'n':
				r = '\n'
			case 'r':
				r = '\r'
			case 't':
				r = '\t'
			case '"', '\\':
			default:
				if state == sqEscape && r != '\'' || state == dqEscape && r != '"' {
					panic(syntaxError(line, pos, fmt.Sprintf("illegal escape '\\%c'", r)))
				}
			}
			buf.WriteRune(r)
			state = dqString
			continue
		case dqString:
			switch r {
			case '"':
				tf(Token{buf.String(), String})
				buf.Reset()
				state = nothing
			case '\\':
				state = dqEscape
			case '\n':
				panic(syntaxError(line, pos, "unterminated string"))
			default:
				buf.WriteRune(r)
			}
			continue
		case sign, integer, integerFirst, hexFirst:
			switch r {
			case '0':
				if state == sign {
					state = integerFirst
				} else {
					state = integer
				}
				buf.WriteRune(r)
				continue
			case 'e', 'E':
				if state != integer {
					panic(badToken(r))
				}
				buf.WriteRune(r)
				state = exponentFirst
				continue
			case 'x', 'X':
				if state != integerFirst {
					panic(badToken(r))
				}
				state = hexFirst
				buf.WriteRune(r)
				continue
			case '.':
				if state == sign || state == hexFirst {
					panic(badToken(r))
				}
				buf.WriteRune(r)
				state = fraction
				continue
			default:
				if r >= '1' && r <= '9' {
					state = integer
					buf.WriteRune(r)
					continue
				}
				if state == sign || state == hexFirst || unicode.IsLetter(r) {
					panic(badToken(r))
				}
				tf(Token{buf.String(), Integer})
				buf.Reset()
				state = nothing
			}
		case fraction:
			if r >= '0' && r <= '9' {
				buf.WriteRune(r)
				continue
			}
			if r == 'e' || r == 'E' {
				buf.WriteRune(r)
				state = exponentFirst
				continue
			}
			if r == '.' || unicode.IsLetter(r) {
				panic(badToken(r))
			}
			tf(Token{buf.String(), Float})
			buf.Reset()
			state = nothing
		case exponent, exponentSign, exponentFirst:
			if r == '-' {
				if state != exponentFirst {
					panic(badToken(r))
				}
				buf.WriteRune(r)
				state = exponentSign
				continue
			}
			if r >= '0' && r <= '9' {
				state = exponent
				buf.WriteRune(r)
				continue
			}
			if state != exponent || r == '.' || unicode.IsLetter(r) {
				panic(badToken(r))
			}
			tf(Token{buf.String(), Float})
			buf.Reset()
			state = nothing
		case typeName:
			if r == '_' || r >= '0' && r <= '9' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' {
				buf.WriteRune(r)
				continue
			}
			if r == ':' {
				state = typeNameSepStart
				buf.WriteRune(r)
				continue
			}
			tf(Token{buf.String(), TypeName})
			buf.Reset()
			state = nothing
		case typeNameSepStart:
			if r == ':' {
				state = typeNameStart
				buf.WriteRune(r)
				continue
			}
			pos--
			panic(badToken(':'))
		case typeNameStart:
			if r >= 'A' && r <= 'Z' {
				state = typeName
				buf.WriteRune(r)
				continue
			}
			panic(badToken(r))
		case identifier:
			if r == '_' || r == '-' || r >= '0' && r <= '9' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' {
				buf.WriteRune(r)
				continue
			}
			if r == ':' {
				state = identifierSepStart
				buf.WriteRune(r)
				continue
			}
			tf(Token{buf.String(), Identifier})
			buf.Reset()
			state = nothing
		case identifierSepStart:
			if r == ':' {
				state = identifierStart
				buf.WriteRune(r)
				continue
			}
			pos--
			panic(badToken(':'))
		case identifierStart:
			if r == '_' || r >= 'a' && r <= 'z' {
				state = typeName
				buf.WriteRune(r)
				continue
			}
			panic(badToken(r))
		}

		switch r {
		case ' ', '\t':
		case '\n':
			line++
			pos = 0
		case '"':
			state = dqString
		case '\'':
			state = sqString
		case '#':
			state = lineComment
		case '{':
			tf(Token{string(r), Lc})
		case '}':
			tf(Token{string(r), Rc})
		case '[':
			tf(Token{string(r), Lb})
		case ']':
			tf(Token{string(r), Rb})
		case '(':
			tf(Token{string(r), Lp})
		case ')':
			tf(Token{string(r), Rp})
		case ',':
			tf(Token{string(r), Comma})
		case '.':
			tf(Token{string(r), Dot})
		case '=':
			state = eqStart
		case '-', '+':
			buf.WriteRune(r)
			state = sign
		case '0':
			buf.WriteRune(r)
			state = integerFirst
		default:
			if r >= '1' && r <= '9' {
				buf.WriteRune(r)
				state = integer
			} else if r >= 'A' && r <= 'Z' {
				buf.WriteRune(r)
				state = typeName
			} else if r >= 'a' && r <= 'z' {
				buf.WriteRune(r)
				state = identifier
			} else {
				panic(badToken(r))
			}
		}
	}
	tf(Token{``, End})
	// Output: foo
}


