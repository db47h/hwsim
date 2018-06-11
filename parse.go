// Copyright 2018 Denis Bernard <db047h@gmail.com>
// Licensed under the MIT license. See license text in the LICENSE file.

package hwsim

import (
	"strconv"

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
		i, err := p.Next(false, false)
		if err != nil {
			return nil, err
		}
		if i == nil {
			return out, nil
		}
		switch p := i.(type) {
		case *hdl.PinIndex:
			for i := 0; i < p.Index; i++ {
				out = append(out, p.Name+"["+strconv.Itoa(i)+"]")
			}
		case *hdl.Pin:
			out = append(out, p.Name)
		default:
			panic("BUG: unexpected PinExpr type")
		}
	}
}

// ParseConnections parses a connection configuration like "partPinX=chipPinY, ..."
// into a []Connections{{PP: "partPinX", CP: []string{"chipPinX"}}, ...}.
//
//	Wire       = Assignment { [ space ] "," [ space ] Assignment } .
//	Assignment = Pin "=" Pin .
//	Pin        = identifier [ "[" Index | Range "]" ] .
//	Index      = integer .
//	Range      = integer ".." integer .
//	identifier = letter { letter | digit } .
//	integer    = { digit } .
//	letter     = "A" ... "Z" | "a" ... "z" | "_" .
//	digit      = "0" ... "9" .
//
func ParseConnections(c string) (conns Connections, err error) {
	p := &hdl.Parser{Input: c}

	for {
		e, err := p.Next(true, true)
		if err != nil {
			return nil, err
		}
		if e == nil {
			return conns, nil
		}
		// p.Next(true, ???) should only return PinAssignments. Failure to do so would be a bug, so just let it panic.
		m := e.(*hdl.PinAssignment)
		ks := expandRange(m.LHS)
		vs := expandRange(m.RHS)
		switch {
		case len(ks) == len(vs):
			// many to many
			for i := range ks {
				conns = append(conns, Connection{ks[i], []string{vs[i]}})
			}
		case len(ks) == 1:
			// one to nany
			conns = append(conns, Connection{ks[0], vs})
		case len(vs) == 1:
			// many to one
			for _, k := range ks {
				conns = append(conns, Connection{k, vs})
			}
		default:
			return nil, errors.New("pin count mismatch in pin mapping: " + m.String())
		}
	}
}

func expandRange(v hdl.PinExpr) []string {
	switch p := v.(type) {
	case *hdl.PinRange:
		var l []string
		for i := p.Start; i <= p.End; i++ {
			l = append(l, busPinName(p.Name, i))
		}
		return l
	case *hdl.PinIndex:
		return []string{busPinName(p.Name, p.Index)}
	case *hdl.Pin:
		return []string{p.Name}
	default:
		panic("BUG: unexpected PinExpr type")
	}
}
