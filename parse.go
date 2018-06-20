// Copyright 2018 Denis Bernard <db047h@gmail.com>
// Licensed under the MIT license. See license text in the LICENSE file.

package hwsim

import (
	"github.com/db47h/hwsim/internal/hdl"
	"github.com/pkg/errors"
)

// IO is a wrapper around ParseIOSpec that panics if an error is returned.
//
func IO(spec string) []string {
	r, err := ParseIOSpec(spec)
	if err != nil {
		panic(errors.Wrap(err, "failed to parse IO description string"))
	}
	return r
}

// ParseIOSpec parses an input or output pin specification string and returns a slice of individual pin names
// suitable for use as the Input or Output field of a PartSpec.
//
// The input format is:
//
//	InputDecl  = PinDecl { "," PinDecl } .
//	PinDecl    = PinIdentifier | BusIdentifier .
//	BusId      = identifier "[" size | Range "]" .
//	PinId      = identifier .
//	Range      = integer ".." integer .
//	identifier = letter { letter | digit } .
//	size       = { digit } .
//	letter     = "A" ... "Z" | "a" ... "z" | "_" .
//	digit      = "0" ... "9" .
//
// Buses are declared by simply specifying their size. For example,
// the I/O specification string "a, b, bus[4]" will be expanded to:
//
//	[]string{"a", "b", "bus[0]", "bus[1]", "bus[2]", "bus[3]"}
//
// Ranges can also be used to force a specific range of bus indices:
//
//	ParseIOSpec("p, g, c[1..4]")
//
// will expand to:
//
//	[]string{"p", "g", "c[1]", "c[2]", "c[3]", "c[4]"}
//
func ParseIOSpec(names string) ([]string, error) {
	var out []string
	p := &hdl.Parser{Input: names}
	for {
		i, err := p.Next(false, true)
		if err != nil {
			return nil, err
		}
		if i == nil {
			return out, nil
		}
		switch p := i.(type) {
		case *hdl.PinRange:
			for i := p.Start; i <= p.End; i++ {
				out = append(out, pinName(p.Name, i))
			}
		case *hdl.PinIndex:
			for i := 0; i < p.Index; i++ {
				out = append(out, pinName(p.Name, i))
			}
		case *hdl.Pin:
			out = append(out, p.Name)
		default:
			panic("BUG: unexpected PinExpr type")
		}
	}
}

// ParseConnections parses a connection configuration like "partPinX=chipPinY, ..."
// into a []Connection{{PP: "partPinX", CP: []string{"chipPinX"}}, ...}.
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
func ParseConnections(c string) (conns []Connection, err error) {
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
			l = append(l, pinName(p.Name, i))
		}
		return l
	case *hdl.PinIndex:
		return []string{pinName(p.Name, p.Index)}
	case *hdl.Pin:
		return []string{p.Name}
	default:
		panic("BUG: unexpected PinExpr type")
	}
}
