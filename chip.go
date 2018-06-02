package hdl

import (
	"strconv"

	"github.com/pkg/errors"
)

type chip struct {
	PartSpec
	specs
	w map[pin]string
}

func (c *chip) mount(s *Socket) []Component {
	var updaters []Component

	for i, p := range c.specs {
		// make a sub-socket
		sub := newSocket(s.c)
		for k, subK := range p.Pinout {
			if subK == "" {
				continue
			}
			// subK is the pin name in the part's namespace
			if n := c.w[pin{i, k}]; n != "" {
				sub.m[subK] = s.PinOrNew(n)
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
	spcs := make(specs, len(parts))

	for pnum, p := range parts {
		sp := p.Spec()
		spcs[pnum] = sp
		ex := p.wires()
		// check wiring
		for k := range ex {
			if _, ok := sp.Pinout[k]; !ok {
				return nil, errors.New("invalid pin name " + k + " for part " + sp.Name)
			}
		}
		for _, k := range sp.In {
			if vs, ok := ex[k]; ok {
				if len(vs) > 1 {
					return nil, errors.New(sp.Name + " input pin " + k + ": invalid pin mapping")
				}
				i, o := pin{-1, vs[0]}, pin{pnum, k}
				if err := wr.add(i, typeUnknown, o, typeInput); err != nil {
					return nil, errors.Wrap(err, spcs.pinName(i)+":"+spcs.pinName(o))
				}
			}
		}
		for _, k := range sp.Out {
			for _, v := range ex[k] {
				i, o := pin{pnum, k}, pin{-1, v}
				if err := wr.add(i, typeOutput, o, typeUnknown); err != nil {
					return nil, errors.Wrap(err, spcs.pinName(i)+":"+spcs.pinName(o))
				}
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

type specs []*PartSpec

func (sp specs) pinName(p pin) string {
	if p.p < 0 {
		return p.name
	}
	return sp[p.p].Name + "." + p.name
}

func checkWiring(wr wiring, root *node, specs specs) (map[pin]string, error) {
	pins := make(map[pin]string, len(wr))
	again := true
	for again {
		again = false
		for _, n := range wr {
			if n.isInput() {
				continue
			} else {
				// remove intermediary pins
				for i := 0; i < len(n.outs); {
					next := n.outs[i]
					if next.isInput() || len(next.outs) == 0 {
						i++
						continue
					}
					again = true
					for _, o := range next.outs {
						o.org = n
					}
					n.outs = append(n.outs, next.outs...)
					next.outs = nil
					// remove orphaned internal chip pins that are not outputs
					if next.pin.p < 0 && !next.isOutput() {
						// delete next
						n.outs[i] = n.outs[len(n.outs)-1]
						n.outs = n.outs[:len(n.outs)-1]
						delete(wr, next.pin)
					}
				}
			}
		}
	}

	// try to set-up pin mappings for sub-parts.
	// mount needs to know quickly the pin number given a part's pin.
	// we need to assign each element of ws an internal pin name:
	//	- an input name
	//	- an output name
	//	- a temp name
	//	and propagate to others.

	i := 0
	for _, n := range wr {
		if len(n.outs) == 0 {
			if n.org == nil || n.org == root {
				// probably an ignored output
				delete(wr, n.pin)
				continue
			}
			if n.pin.p < 0 && !n.isOutput() {
				return nil, errors.New("pin " + specs.pinName(n.pin) + " not connected to any input")
			}
		} else if n.org == nil && !n.isOutput() {
			return nil, errors.New("pin " + specs.pinName(n.pin) + " not connected to any output")
		}

		if n.name == "" {
			t := n
			for prev := t.org; prev != nil && prev != root; t, prev = prev, t.org {
			}
			if t.org == nil {
				t.setName("__" + strconv.Itoa(i))
			} else {
				t.setName(t.pin.name)
			}
			i++
		}
		pins[n.pin] = n.name
	}
	return pins, nil
}
