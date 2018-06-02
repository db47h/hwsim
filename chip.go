package hdl

import (
	"github.com/pkg/errors"
)

type chip struct {
	PartSpec
	parts []Part
	w     wiring
}

func (c *chip) mount(s *Socket) []Component {
	var updaters []Component

	for _, p := range c.parts {
		// make a sub-socket
		sub := newSocket(s.c)
		for k, subK := range p.Spec().Pinout {
			// subK is the pin name in the part's namespace
			if n := c.w[pin{p, k}]; n != nil {
				sub.m[subK] = s.PinOrNew(n.name)
			}
		}
		updaters = append(updaters, p.Spec().Mount(sub)...)
	}
	return updaters
}

// Chip composes existing parts into a new part packaged into a chip.
// The pin names specified as inputs and outputs will be the inputs
// and outputs of the chip.
//
// An Xor gate could be created like this:
//
//	xor := Chip(
//		[]string{"a", "b"},
//		[]string{"out"},
//		[]hdl.Part{
//			hdl.Nand(hdl.W{"a": "a", "b": "b", "out": "nandAB"}),
//			hdl.Nand(hdl.W{"a": "a", "b": "nandAB", "out": "w0"}),
//			hdl.Nand(hdl.W{"a": "b", "b": "nandAB", "out": "w1"}),
//			hdl.Nand(hdl.W{"a": "w0", "b": "w1", "out": "out"}),
//		})
//
// The returned value is a function of type NewPartFunc that can be used to
// compose the new part with others into other chips:
//
//	xnor := hdl.Chip(
//		[]string{"a", "b"},
//		[]string{"out"},
//		[]hdl.Part{
//			xor(hdl.W{"a": "a", "b": "b", "xorAB"}),
//			hdl.Not(hdl.W{"in": "xorAB", "out": "out"}),
//		})
//
func Chip(name string, inputs []string, outputs []string, parts []Part) (NewPartFunc, error) {
	inputs = ExpandBus(inputs...)
	outputs = ExpandBus(outputs...)

	// build wiring
	wr, root := newWiring(inputs, outputs)

	for _, p := range parts {
		sp := p.Spec()
		ex := p.wires()
		for _, k := range sp.In {
			if vs, ok := ex[k]; ok {
				if len(vs) > 1 {
					return nil, errors.New(sp.Name + " input pin " + k + ": invalid pin mapping")
				}
				if err := wr.add(nil, vs[0], typeUnknown, p, k, typeInput); err != nil {
					return nil, err
				}
			}
		}
		for _, k := range sp.Out {
			for _, v := range ex[k] {
				if err := wr.add(p, k, typeOutput, nil, v, typeUnknown); err != nil {
					return nil, err
				}
			}
		}
	}

	if err := wr.check(root); err != nil {
		return nil, err
	}

	pinout := make(W)
	for _, i := range inputs {
		if n := wr[pin{nil, i}]; n != nil {
			pinout[i] = n.name
		}
	}
	for _, o := range outputs {
		if n := wr[pin{nil, o}]; n != nil {
			pinout[o] = n.name
		}
	}

	c := &chip{
		PartSpec{
			Name:   name,
			In:     inputs,
			Out:    outputs,
			Pinout: pinout,
		},
		parts,
		wr,
	}
	c.PartSpec.Mount = c.mount
	return c.Wire, nil
}
