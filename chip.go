package hwsim

import (
	"github.com/pkg/errors"
)

type chip struct {
	PartSpec             // PartSpec for this chip
	parts    []*PartSpec // sub parts
	w        wiring
}

func (c *chip) mount(s *Socket) []Component {
	var updaters []Component

	for i, p := range c.parts {
		// make a sub-socket
		sub := newSocket(s.c)
		// k is the exported pin name (always an input or output name)
		// subK is the pin name in the part's namespace
		for k, subK := range p.Pinout {
			if subK == "" {
				continue
			}
			if n := c.w.wireName(pin{i, k}); n != "" {
				sub.m[subK] = s.pinOrNew(n)
			} else {
				// wire unknown pins to False.
				// Chip() makes sure that unknown pins can only be inputs.
				sub.m[subK] = cstFalse
			}
		}
		updaters = append(updaters, p.Mount(sub)...)
	}
	return updaters
}

// Chip composes existing parts into a new chip.
//
// The pin names specified as inputs and outputs will be the inputs
// and outputs of the chip (the chip interface).
//
// A XOR gate could be created like this:
//
//	xor, err := Chip(
//		"XOR",
//		In("a, b"),
//		Out("out"),
//		Parts{
//			Nand("a=a, b=b, out=nandAB")),
//			Nand("a=a, b=nandAB, out=w0")),
//			Nand("a=b, b=nandAB, out=w1")),
//			Nand("a=w0, b=w1, out=out")),
//		})
//
// The created chip can be composed with other parts to create other chips
// simply by calling the returned NewPartFn with a connection configuration:
//
//	xnor, err := Chip(
//		"XNOR",
//		In("a, b"),
//		Out("out"),
//		Parts{
//			// reuse the xor chip created above
//			xor("a=a, b=b, out=xorAB"}),
//			Not("in=xorAB, out=out"}),
//		})
//
func Chip(name string, inputs In, outputs Out, parts Parts) (NewPartFn, error) {
	inputs = ExpandBus(inputs...)
	outputs = ExpandBus(outputs...)

	// build wiring
	wr := newWiring(inputs, outputs)
	spcs := make([]*PartSpec, len(parts))

	for pnum := range parts {
		p := &parts[pnum]
		spcs[pnum] = p.PartSpec
		conns := p.Connections

		// check that all keys match one of the part's input or output pins
		for k := range conns {
			if _, ok := p.Pinout[k]; !ok {
				return nil, errors.New("invalid pin name " + k + " for part " + p.Name)
			}
		}
		// add inputs
		for _, k := range p.In {
			if vs, ok := conns[k]; ok {
				if len(vs) > 1 {
					return nil, errors.New(p.Name + " input pin " + k + "connected to more than one output")
				}
				i, o := pin{-1, vs[0]}, pin{pnum, k}
				if err := wr.connect(i, typeUnknown, o, typeInput); err != nil {
					return nil, errors.Wrap(err, pinName(spcs, i)+":"+pinName(spcs, o))
				}
			}
		}
		// add outputs
		for _, k := range p.Out {
			if vs, ok := conns[k]; ok {
				for _, v := range vs {
					i, o := pin{pnum, k}, pin{-1, v}
					if err := wr.connect(i, typeOutput, o, typeUnknown); err != nil {
						return nil, errors.Wrap(err, pinName(spcs, i)+":"+pinName(spcs, o))
					}
				}
			} else {
				p := pin{pnum, k}
				wr[p] = &node{pin: p, typ: typeOutput}
			}
		}
	}

	if err := wr.pruneEphemeral(); err != nil {
		return nil, err
	}

	pinout := make(map[string]string)
	// map all input and output pins, even if not used.
	// mount will ignore pins with an empty value.
	for _, i := range inputs {
		pinout[i] = wr.wireName(pin{-1, i})
	}
	for _, o := range outputs {
		pinout[o] = wr.wireName(pin{-1, o})
	}

	c := &chip{
		PartSpec{
			Name:   name,
			In:     inputs,
			Out:    outputs,
			Pinout: pinout,
		},
		spcs,
		wr,
	}
	c.PartSpec.Mount = c.mount
	return c.PartSpec.Wire, nil
}

func pinName(sp []*PartSpec, p pin) string {
	if p.p < 0 {
		return p.name
	}
	return sp[p.p].Name + "." + p.name
}
