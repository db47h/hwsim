package hdl

import "github.com/pkg/errors"

type chip struct {
	PartSpec
	parts []Part
}

func (c *chip) mount(cc *Circuit, pins Socket) []Component {
	var updaters []Component
	if len(pins) < cstCount {
		panic("invalid pin map")
	}
	// collect parts
	for _, p := range c.parts {
		// build the part's external pin map
		ppins := cstPins()
		for ppin, cpin := range p.Wires() {
			var n int
			var ok bool
			// chip pin name unknown, allocate it
			if n, ok = pins[cpin]; !ok {
				n = cc.Alloc()
				pins[cpin] = n
			}
			// map the part's pin name to the same number
			// thus establishing the connection.
			ppins[ppin] = n
		}
		pup := p.Spec().Mount(cc, ppins)
		updaters = append(updaters, pup...)
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
	// check that no outputs are connected together.
	// outs is a map of chip name to # of inputs ising it
	outs := make(Socket)
	// add our inputs as outputs within the chip
	for _, i := range inputs {
		// set to one because when the returned NewPartFn will be called,
		// unconnected I/O wires will be automatically connected to False
		outs[i] = 1
	}
	// for each part, add its outputs
	for _, p := range parts {
		w := p.Wires()
		sp := p.Spec()
		for _, o := range sp.Out {
			n := w[o]
			if n == False {
				// nil or unconnected output, ignore.
				continue
			}
			if n == True {
				return nil, errors.New(sp.Name + " pin " + o + " connected to constant True input")
			}
			if _, ok := outs[n]; ok {
				return nil, errors.New(sp.Name + " pin " + o + ":" + n + ": pin already used as output by another part or is an input pin of the chip")
			}
			outs[n] = 0
		}
	}

	// Check that each output is used as an input somewhere.
	// Start by assuming that the chip's outputs are connected.
	for _, o := range outputs {
		outs[o] = 1
	}
	// add False and True as a valid, connected outputs
	outs[False] = 1
	outs[True] = 1
	for _, p := range parts {
		w := p.Wires()
		sp := p.Spec()
		for _, o := range sp.In {
			n := w[o]
			if n == True {
				continue
			}
			if cnt, ok := outs[n]; ok {
				outs[n] = cnt + 1
				continue
			}
			return nil, errors.New(sp.Name + " pin " + o + ":" + n + " not connected to any output")
		}
	}
	for k, v := range outs {
		if v == 0 {
			return nil, errors.New("pin " + k + " not connected to any input")
		}
	}

	c := &chip{
		PartSpec{
			Name: name,
			In:   inputs,
			Out:  outputs,
		},
		parts,
	}
	c.Mount = c.mount
	return c.Wire, nil
}
