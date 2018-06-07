package hwsim

import (
	"github.com/db47h/hwsim/internal/hdl"
	"github.com/db47h/hwsim/internal/lex"
	"github.com/pkg/errors"
)

// parseIOspec parses the pin specification string and returns individual pin
// names in a slice, also expanding bus declarations to individual pin names.
// For example:
//
//	parseIOspec("in[2] sel") // returns []string{"in[0]", "in[1]", "sel"}
//
func parseIOspec(names string) ([]string, error) {
	var out []string

	l := hdl.Lexer(names)

	i := l.Lex()
	if i.Type == hdl.EOF {
		return nil, nil
	}
F:
	for {
		if i.Type != hdl.Ident {
			return nil, parseError(names, i.Pos, "expected pin name")
		}
		name := i.Value.(string)
		// after ident, expect comma, [ or EOF
		i = l.Lex()
		switch i.Type {
		case hdl.EOF:
			out = append(out, name)
			break F
		case hdl.Comma:
			out = append(out, name)
			i = l.Lex()
			continue
		case hdl.BracketOpen:
		default:
			return nil, parseError(names, i.Pos, "expected bus size specification or comma")
		}
		// expect bus size
		i = l.Lex()
		if i.Type != hdl.Int {
			return nil, parseError(names, i.Pos, "missing bus size")
		}
		for i, cnt := 0, i.Value.(int); i < cnt; i++ {
			out = append(out, busPinName(name, i))
		}
		i = l.Lex()
		if i.Type != hdl.BracketClose {
			return nil, parseError(names, i.Pos, "missing close bracket")
		}
		i = l.Lex()
		if i.Type == hdl.EOF {
			break
		}
		if i.Type != hdl.Comma {
			return nil, parseError(names, i.Pos, "expected comma or end of input")
		}
		i = l.Lex()
	}

	return out, nil
}

func parseError(in string, pos lex.Pos, msg string) error {
	return errors.Errorf("in %q at pos %d: %s", in, pos+1, msg)
}
