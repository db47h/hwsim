package hwsim

import (
	"strconv"

	"github.com/pkg/errors"
)

type chip struct {
	PartSpec             // PartSpec for this chip
	parts    []*PartSpec // sub parts
	// wires maps pins used in a chip to the internal wire name which may be the
	// name of any input/output of the chip or dynamically allocated (__0, __1, etc.)
	wires map[pin]string
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
			if n := c.wires[pin{i, k}]; n != "" {
				sub.m[subK] = s.PinOrNew(n)
				//				log.Printf("%s: wire for %s.%s (%s):%s = %d", c.Name, p.Name, k, subK, n, sub.m[subK])
			} else {
				// wire unknown pins to False.
				// Chip() makes sure that unknown pins can only be inputs.
				//				log.Printf("%s: wire for %s.%s (%s):%s = %d", c.Name, p.Name, k, subK, "???", sub.m[subK])
				sub.m[subK] = cstFalse
			}
		}
		updaters = append(updaters, p.Mount(sub)...)
	}
	return updaters
}

// Chip composes existing parts into a new part packaged into a chip.
// The pin names specified as inputs and outputs will be the inputs
// and outputs of the chip.
//
// An Xor gate could be created like this:
//
//	xor, err := Chip(
//		"XOR",
//		In{"a", "b"},
//		Out{"out"},
//		Parts{
//			Nand(W{"a": "a", "b": "b", "out": "nandAB"}),
//			Nand(W{"a": "a", "b": "nandAB", "out": "w0"}),
//			Nand(W{"a": "b", "b": "nandAB", "out": "w1"}),
//			Nand(W{"a": "w0", "b": "w1", "out": "out"}),
//		})
//
// The returned value is a function of type NewPartFunc that can be used to
// compose the new part with others into other chips:
//
//	xnor, err := Chip(
//		"XNOR",
//		In{"a", "b"},
//		Out{"out"},
//		Parts{
//			xor(W{"a": "a", "b": "b", "xorAB"}),
//			Not(W{"in": "xorAB", "out": "out"}),
//		})
//
func Chip(name string, inputs In, outputs Out, parts Parts) (NewPartFn, error) {
	inputs = ExpandBus(inputs...)
	outputs = ExpandBus(outputs...)

	// build wiring
	wr, root := newWiring(inputs, outputs)
	spcs := make([]*PartSpec, len(parts))

	for pnum, p := range parts {
		sp := p.Spec()
		spcs[pnum] = sp
		ex := p.wires()

		// check that all keys match one of the part's input or output pins
		for k := range ex {
			if _, ok := sp.Pinout[k]; !ok {
				return nil, errors.New("invalid pin name " + k + " for part " + sp.Name)
			}
		}
		// add inputs
		for _, k := range sp.In {
			if vs, ok := ex[k]; ok {
				if len(vs) > 1 {
					return nil, errors.New(sp.Name + " input pin " + k + "connected to more than one output")
				}
				i, o := pin{-1, vs[0]}, pin{pnum, k}
				if err := wr.add(i, typeUnknown, o, typeInput); err != nil {
					return nil, errors.Wrap(err, pinName(spcs, i)+":"+pinName(spcs, o))
				}
			}
		}
		// add outputs
		for _, k := range sp.Out {
			if vs, ok := ex[k]; ok {
				for _, v := range vs {
					i, o := pin{pnum, k}, pin{-1, v}
					if err := wr.add(i, typeOutput, o, typeUnknown); err != nil {
						return nil, errors.Wrap(err, pinName(spcs, i)+":"+pinName(spcs, o))
					}
				}
			} else {
				p := pin{pnum, k}
				wr[p] = &node{pin: p, typ: typeOutput}
			}
		}
	}

	pins, err := checkWiring(wr, root, spcs)
	if err != nil {
		return nil, err
	}

	pinout := make(W)
	// map all input and output pins, even if not used.
	// mount will ignore pins with an empty value.
	for _, i := range inputs {
		pinout[i] = pins[pin{-1, i}]
	}
	for _, o := range outputs {
		pinout[o] = pins[pin{-1, o}]
	}

	c := &chip{
		PartSpec{
			Name:   name,
			In:     inputs,
			Out:    outputs,
			Pinout: pinout,
		},
		spcs,
		pins,
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

func checkWiring(wr wiring, root *node, spcs []*PartSpec) (map[pin]string, error) {
	pins := make(map[pin]string, len(wr))
	wireNum := 0
	for _, n := range wr {
		// Error on non-output pins with no inbound connection.
		if !n.isOutput() && n.org == nil {
			return nil, errors.New("pin " + pinName(spcs, n.pin) + " not connected to any output")
		}

		// remove temporary pins.
		// input pins can safely be ignored since len(n.outs) is 0 for them.
		// Inspect every "next" output pin in the wire chain.
		for i := 0; i < len(n.outs); {
			next := n.outs[i]
			if len(next.outs) == 0 {
				if next.pin.p < 0 && !next.isOutput() {
					return nil, errors.New("pin " + pinName(spcs, next.pin) + " not connected to any input")
				}
				i++
				continue
			}
			// there is a wire chain: n -> next -> next.outs
			// merge it into n.outs = n.outs + next.outs
			for _, o := range next.outs {
				o.org = n
			}
			n.outs = append(n.outs, next.outs...)
			next.outs = nil
			// now decide what to do with next
			// remove orphaned internal chip pins that are not outputs
			if next.pin.p < 0 && !next.isOutput() {
				// delete next
				n.outs[i] = n.outs[len(n.outs)-1]
				n.outs = n.outs[:len(n.outs)-1]
				delete(wr, next.pin)
			}
		}

		// assign a wire name to the pin tree
		if n.name == "" {
			t := n
			for prev := t.org; prev != nil && prev != root; t, prev = prev, t.org {
			}
			if t.org == nil {
				t.setName("__" + strconv.Itoa(wireNum))
			} else {
				// chip input pin, use its name.
				// TODO: this could be removed and only use internal pin names
				// but there is an issue with True/False/Clk
				t.setName(t.pin.name)
			}
			wireNum++
		}
		pins[n.pin] = n.name
	}

	return pins, nil
}
