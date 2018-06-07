package hdl

import (
	"strings"
	"unicode"

	"github.com/db47h/hwsim/internal/lex"
)

// Tokens
const (
	EOF lex.Type = lex.EOF
	Raw lex.Type = iota
	Ident
	BracketOpen
	BracketClose
	Comma
	Int
	Range
)

// Lexer returns a new lexer for i/o specs and connection descriptions.
//
func Lexer(input string) lex.Interface {
	return lex.New(strings.NewReader(input), lexInit)
}

func lexInit(l *lex.Lexer) lex.StateFn {
	r := l.Next()
	switch {
	case r == lex.EOF:
		return lexEOF
	case unicode.IsSpace(r):
		l.AcceptWhile(unicode.IsSpace)
	case unicode.IsLetter(r) || r == '_':
		return lexIdent
	case r == '[':
		l.Emit(BracketOpen, "[")
	case r == ']':
		l.Emit(BracketClose, "]")
	case r == ',':
		l.Emit(Comma, ",")
	case '0' <= r && r <= '9':
		return lexNumber
	case r == '.':
		n := l.Next()
		if n == '.' {
			l.Emit(Range, "..")
			break
		}
		l.Backup()
		fallthrough
	default:
		l.Emit(Raw, r)
		return lexEOF
	}
	return nil
}

func lexNumber(l *lex.Lexer) lex.StateFn {
	var buf strings.Builder
	buf.WriteRune(l.Current())
	i := int(l.Current() - '0')
	r := l.Next()
	for '0' <= r && r <= '9' {
		i = i*10 + int(r-'0')
		buf.WriteRune(r)
		r = l.Next()
	}
	l.Backup()
	l.Emit(Int, i)
	return nil
}

func lexIdent(l *lex.Lexer) lex.StateFn {
	var buf strings.Builder
	buf.Grow(8)
	buf.WriteRune(l.Current())
	r := l.Next()
	for unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
		buf.WriteRune(r)
		r = l.Next()
	}
	l.Backup()
	l.Emit(Ident, buf.String())
	return nil
}

// lexEOF places the lexer in End-Of-File state.
// Once in this state, the lexer will only emit EOF.
//
func lexEOF(l *lex.Lexer) lex.StateFn {
	l.Emit(lex.EOF, "end of input")
	return lexEOF
}
