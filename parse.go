package hwsim

import (
	"strconv"
	"strings"

	"github.com/db47h/hwsim/internal/hdl"
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
	p := &hdl.Parser{Input: names}
	for {
		i, err := p.Next(false)
		if err != nil {
			return nil, err
		}
		if i == nil {
			return out, nil
		}
		switch p := i.(type) {
		case hdl.PinRange:
			return nil, errors.Errorf("in %q at pos %d: pin ranges forbidden in input/output declaration", names, p.Pos)
		case hdl.Pin:
			out = append(out, p.Name)
		case hdl.PinIndex:
			for i := 0; i < p.Index; i++ {
				out = append(out, p.Name+"["+strconv.Itoa(i)+"]")
			}
		}
	}
}

// ParseConnections parses a connection configuration like "partPinX=chipPinY, ..."
// into a Connections{"partPinX": []string{"chipPinX"}}.
//
//	Wire       = Assignment { [ space ] "," [ space ] Assignment } .
//	Assignment = pin "=" pin .
//	Pin        = identifier [ "[" Range | index "]" ] .
//	Range      = index ".." index .
//	identifier = letter { letter | digit } .
//	index      = { digit } .
//	letter     = "A" .. "Z" | "a" .. "z" | "_" .
//	digit      = "0" .. "9" .
//
func ParseConnections(c string) (conns Connections, err error) {
	// just split the input string, syntax check is done somewhere else
	mappings := strings.FieldsFunc(c, func(r rune) bool { return r == ',' })
	conns = make(Connections)

	for _, m := range mappings {
		var ks, vs []string
		m = strings.TrimSpace(m)
		i := strings.IndexRune(m, '=')
		if i < 0 {
			return nil, errors.New(m + ": not a valid pin mapping (missing =)")
		}
		k, v := strings.TrimSpace(m[:i]), strings.TrimSpace(m[i+1:])
		// parse and expand
		if k == "" || v == "" {
			return nil, errors.New("invalid pin mapping " + k + ":" + v)
		}
		if ks, err = expandRange(k); err != nil {
			return nil, errors.Wrap(err, "expand key "+k)
		}
		if vs, err = expandRange(v); err != nil {
			return nil, errors.Wrap(err, "expand value "+v)
		}
		switch {
		case len(ks) == len(vs):
			// many to many
			for i := range ks {
				conns[ks[i]] = []string{vs[i]}
			}
		case len(ks) == 1:
			// one to nany
			conns[k] = vs
		case len(vs) == 1:
			// many to one
			for _, k := range ks {
				conns[k] = vs
			}
		default:
			return nil, errors.New("pin count mismatch in pin mapping: " + k + ":" + v)
		}
	}
	return conns, nil
}

func expandRange(name string) ([]string, error) {
	i := strings.IndexRune(name, '[')
	if i < 0 {
		return []string{name}, nil
	}
	bus := name[:i]
	if bus == "" {
		return nil, errors.New("empty bus name")
	}
	n := name[i+1:]
	i = strings.Index(n, "..")
	if i < 0 {
		return []string{name}, nil
	}
	start, err := strconv.Atoi(n[:i])
	if err != nil {
		return nil, err
	}
	n = n[i+2:]
	i = strings.IndexRune(n, ']')
	if i < 0 {
		return nil, errors.New("no terminamting ] in bus range")
	}
	end, err := strconv.Atoi(n[:i])
	if err != nil {
		return nil, err
	}
	r := make([]string, 0, end-start+1)
	for i := start; i <= end; i++ {
		r = append(r, busPinName(bus, i))
	}
	return r, nil
}
