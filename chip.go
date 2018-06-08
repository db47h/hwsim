package hwsim

import (
	"sort"

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
				if pin, ok := sub.m[subK]; ok {
					s.m[n] = pin // part's internal pin already allocated, propagate to our wire
				} else {
					sub.m[subK] = s.pinOrNew(n)
				}
				// log.Printf("%s:%s pin %s (%s) on wire %s =  %d", c.Name, p.Name, k, subK, n, sub.m[subK])
			} else {
				// wire unknown pins to False.
				// Chip() makes sure that unknown pins can only be inputs.
				sub.m[subK] = cstFalse
				// log.Printf("%s:%s pin %s (%s) on wire ??? = FALSE", c.Name, p.Name, k, subK)
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
func Chip(name string, inputs Inputs, outputs Outputs, parts Parts) (NewPartFn, error) {
	// build wiring
	wr := newWiring(inputs, outputs)
	spcs := make([]*PartSpec, len(parts))

	isOut := func(p *PartSpec, name string) bool {
		n := sort.SearchStrings(p.Outputs, name)
		return n < len(p.Outputs) && p.Outputs[n] == name
	}

	for pnum := range parts {
		p := &parts[pnum]
		spcs[pnum] = p.PartSpec
		conns := p.Connections
		sort.Strings(p.Outputs)
		//		log.Printf("part: %s, conns: %v", p.Name, conns)

		// Add the part's pins tho the wiring
		for i := range conns {
			c := &conns[i]
			k := c.PP

			// check that the pin matches one of the part's input or output pins
			if _, ok := p.Pinout[k]; !ok {
				return nil, errors.New("invalid pin name " + k + " for part " + p.Name)
			}

			if isOut(p.PartSpec, k) {
				for _, v := range c.CP {
					i, o := pin{pnum, k}, pin{-1, v}
					if err := wr.connect(i, typeOutput, o, typeUnknown); err != nil {
						return nil, errors.Wrap(err, pinName(spcs, i)+":"+pinName(spcs, o))
					}
				}
			} else {
				if len(c.CP) > 1 {
					return nil, errors.New(p.Name + " input pin " + k + "connected to more than one output")
				}
				i, o := pin{-1, c.CP[0]}, pin{pnum, k}
				if err := wr.connect(i, typeUnknown, o, typeInput); err != nil {
					return nil, errors.Wrap(err, pinName(spcs, i)+":"+pinName(spcs, o))
				}
			}
		}

		// add omitted outputs
		for _, k := range p.Outputs {
			p := pin{pnum, k}
			if _, ok := wr[p]; !ok {
				wr[p] = &node{pin: p, typ: typeOutput}
			}
		}
	}

	if err := wr.pruneEphemeral(); err != nil {
		return nil, err
	}

	// need to give the same names to wires that connect to a part whose outputs are merged
	for pnum := range parts {
		p := &parts[pnum]
		m := make(map[string]string)
		for _, k := range p.Outputs {
			n := wr[pin{pnum, k}]
			if n == nil || n.name == "" {
				continue
			}
			ik := p.Pinout[k]
			if wn := m[ik]; wn != "" {
				// output ik connected to wire wn, rename n
				n.root().setName(wn)
			} else {
				m[ik] = n.name
			}
		}
	}

	pinout := make(map[string]string)
	// map all input and output pins, even if not used.
	// mount will ignore pins with an empty value.
	for _, i := range inputs {
		pinout[i] = wr.wireName(pin{-1, i})
		// if n := wr[pin{-1, i}]; n != nil {
		// 	pinout[i] = n.name
		// }
	}
	for _, o := range outputs {
		pinout[o] = wr.wireName(pin{-1, o})
		// if n := wr[pin{-1, o}]; n != nil {
		// 	pinout[o] = n.name
		// }
	}

	c := &chip{
		PartSpec{
			Name:    name,
			Inputs:  inputs,
			Outputs: outputs,
			Pinout:  pinout,
		},
		spcs,
		wr,
	}
	c.PartSpec.Mount = c.mount
	return c.PartSpec.NewPart, nil
}

func pinName(sp []*PartSpec, p pin) string {
	if p.p < 0 {
		return p.name
	}
	return sp[p.p].Name + "." + p.name
}
