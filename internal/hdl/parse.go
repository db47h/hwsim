package hdl

import (
	"strconv"
	"strings"
	"unicode"

	"github.com/db47h/hwsim/internal/lex"
	"github.com/pkg/errors"
)

// Tokens
const (
	Raw lex.Type = iota
	Ident
	BracketOpen
	BracketClose
	Comma
	Int
	Range
	Equal
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
	case r == '=':
		l.Emit(Equal, "=")
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

// Pin is a simple pin name
//
type Pin struct {
	Name string
	pos  lex.Pos
}

// PinIndex is an indexed pin p[index]
//
type PinIndex struct {
	*Pin
	Index int
}

// PinRange is a pin range p[start..end]
//
type PinRange struct {
	*Pin
	Start int
	End   int
}

// PinAssignment is a part pin to chip pin assignment. pp=pc
//
type PinAssignment struct {
	LHS PinExpr
	RHS PinExpr
}

// PinExpr is a pin declaration expression.
type PinExpr interface {
	Pos() int
	String() string
}

// Pos implements PinExpr
func (p *Pin) Pos() int       { return int(p.pos) }
func (p *Pin) String() string { return p.Name }

func (p *PinIndex) String() string { return p.Name + "[" + strconv.Itoa(p.Index) + "]" }

func (p *PinRange) String() string {
	return p.Name + "[" + strconv.Itoa(p.Start) + ".." + strconv.Itoa(p.End) + "]"
}

// Pos implements PinExpr
func (p *PinAssignment) Pos() int       { return p.LHS.Pos() }
func (p *PinAssignment) String() string { return p.LHS.String() + "=" + p.RHS.String() }

// Parser is a simplistic parser
//
type Parser struct {
	Input string
	l     lex.Interface
	i     lex.Item
	state int
}

const (
	stateInit = iota
	stateStarted
	stateDone = -1
)

// Next returns the next item in the input stream
// It only recognizes pin names followed by an index or range and separated by commas.
// conns specifies if reading a connection config strings, in which case the only
// possible return type is *PinAssignment or nil at EOF.
//
func (p *Parser) Next(conns, allowRange bool) (PinExpr, error) {
	if p.state == stateDone {
		return nil, nil
	}
	if p.l == nil {
		p.l = Lexer(p.Input)
	}

	p.i = p.l.Lex()
	if p.state == stateInit && p.i.Type == lex.EOF {
		p.state = stateDone
		return nil, nil
	}
	p.state = stateStarted

	pin, err := p.getPin(allowRange)
	if err != nil {
		p.state = stateDone
		return nil, err
	}
	if !conns {
		switch p.i.Type {
		case lex.EOF:
			p.state = stateDone
			fallthrough
		case Comma:
			return pin, nil
		default:
			return nil, parseError(p.Input, p.i.Pos, "unexpected "+p.i.String())
		}
	}

	if p.i.Type != Equal {
		return nil, parseError(p.Input, p.i.Pos, "missing '=' in connection description")
	}
	p.i = p.l.Lex()
	pin2, err := p.getPin(allowRange)
	if err != nil {
		p.state = stateDone
		return nil, err
	}
	switch p.i.Type {
	case lex.EOF:
		p.state = stateDone
		fallthrough
	case Comma:
		return &PinAssignment{pin, pin2}, nil
	}

	return nil, parseError(p.Input, p.i.Pos, "unexpected "+p.i.String())
}

func (p *Parser) getPin(allowRange bool) (PinExpr, error) {
	if p.i.Type != Ident {
		return nil, parseError(p.Input, p.i.Pos, "expected pin name")
	}
	pin := &Pin{p.i.Value.(string), p.i.Pos}
	// after ident, expect ',', '[', '=' or EOF
	p.i = p.l.Lex()
	if p.i.Type != BracketOpen {
		return pin, nil
	}
	// expect bus size
	i := p.l.Lex()
	if i.Type != Int {
		return nil, parseError(p.Input, i.Pos, "integer value expected after '['")
	}
	start := i.Value.(int)
	end := -1
	i = p.l.Lex()
	if i.Type == Range {
		if !allowRange {
			return nil, parseError(p.Input, i.Pos, "pin ranges forbidden in this context")
		}
		i = p.l.Lex()
		if i.Type != Int {
			return nil, parseError(p.Input, i.Pos, "integer value expected after '..'")
		}
		end = i.Value.(int)
		i = p.l.Lex()
	}
	if i.Type != BracketClose {
		return nil, parseError(p.Input, i.Pos, "closing ']' expected after index or range")
	}
	p.i = p.l.Lex()
	if end >= 0 {
		return &PinRange{pin, start, end}, nil
	}
	return &PinIndex{pin, start}, nil
}

func parseError(in string, pos lex.Pos, msg string) error {
	return errors.Errorf("in %q at pos %d: %s", in, pos+1, msg)
}
