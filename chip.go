// Copyright 2018 Denis Bernard <db047h@gmail.com>
// Licensed under the MIT license. See license text in the LICENSE file.

package hwsim

import (
	"sort"
	"strconv"

	"github.com/pkg/errors"
)

type chip struct {
	PartSpec             // PartSpec for this chip
	parts    []*PartSpec // sub parts
	w        wiring
}

func (c *chip) mount(s *Socket) []Updater {
	var updaters []Updater
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
				// log.Printf("%s:%s pin %s (%s) on wire %s =  %d", c.Name, p.Name, k, subK, n, sub.m[subK])
			} else {
				// Chip() makes sure that unknown pins can only be inputs.
				if subK == Clk {
					sub.m[subK] = cstClk
				} else {
					sub.m[subK] = cstFalse
				}
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
//	xor, err := hwsim.Chip(
//		"XOR",
//		hwsim.In("a, b"),
//		hwsim.Out("out"),
//		hwsim.Parts{
//			hwlib.Nand("a=a, b=b, out=nandAB")),
//			hwlib.Nand("a=a, b=nandAB, out=w0")),
//			hwlib.Nand("a=b, b=nandAB, out=w1")),
//			hwlib.Nand("a=w0, b=w1, out=out")),
//		})
//
// The created chip can be composed with other parts to create other chips
// simply by calling the returned NewPartFn with a connection configuration:
//
//	xnor, err := Chip(
//		"XNOR",
//		hwsim.In("a, b"),
//		hwsim.Out("out"),
//		hwsim.Parts{
//			// reuse the xor chip created above
//			xor("a=a, b=b, out=xorAB"}),
//			hwlib.Not("in=xorAB, out=out"}),
//		})
//
func Chip(name string, inputs string, outputs string, parts ...Part) (NewPartFn, error) {
	// build wiring
	ins, err := ParseIOSpec(inputs)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse input spec")
	}
	outs, err := ParseIOSpec(outputs)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse output spec")
	}

	wr := newWiring(ins, outs)
	spcs := make([]*PartSpec, len(parts))

	isOut := func(p *PartSpec, name string) bool {
		n := sort.SearchStrings(p.Outputs, name)
		return n < len(p.Outputs) && p.Outputs[n] == name
	}

	for pnum := range parts {
		p := &parts[pnum]
		spcs[pnum] = p.PartSpec
		conns := p.Conns
		sort.Strings(p.Outputs)

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
					if err := wr.connect(i, typeOutput, tmpName(pnum, k), o, typeUnknown); err != nil {
						return nil, errors.Wrap(err, pinName(spcs, i)+":"+pinName(spcs, o))
					}
				}
			} else {
				if len(c.CP) > 1 {
					return nil, errors.New(p.Name + " input pin " + k + "connected to more than one output")
				}
				i, o := pin{-1, c.CP[0]}, pin{pnum, k}
				if err := wr.connect(i, typeUnknown, c.CP[0], o, typeInput); err != nil {
					return nil, errors.Wrap(err, pinName(spcs, i)+":"+pinName(spcs, o))
				}
			}
		}

		// add omitted part outputs: pins not used within this chip but that must be assigned pin #s
		// and rename wires that connect to merged outputs in a part (for fan-out to work with chip nesting).
		// m maps a part's internal output pin names to this chip's wire names.
		m := make(map[string]string)
		for _, k := range p.Outputs {
			o := pin{pnum, k}
			n, ok := wr[o]
			if !ok {
				wr[o] = &node{name: tmpName(pnum, k), pin: o, typ: typeOutput}
				continue
			}
			// rename
			ik := p.Pinout[k]
			if ik == "" {
				continue
			}
			if wn := m[ik]; wn == "" {
				m[ik] = n.name
			} else {
				// output ik connected to wire wn, rename n
				n.setName(wn)
			}
		}
	}

	if err := wr.prune(); err != nil {
		return nil, err
	}

	pinout := make(map[string]string)
	// map all input and output pins, even if not used.
	for _, i := range ins {
		pinout[i] = wr.wireName(pin{-1, i})
	}
	for _, o := range outs {
		pinout[o] = wr.wireName(pin{-1, o})
	}

	c := &chip{
		PartSpec{
			Name:    name,
			Inputs:  ins,
			Outputs: outs,
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

func tmpName(pnum int, k string) string {
	return "__" + strconv.Itoa(pnum) + "_" + k
}
