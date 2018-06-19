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

func (c *chip) mount(s *Socket) Updater {
	impl := &chipImpl{
		ups: make([]Updater, len(c.parts)),
	}
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
				// log.Printf("%s:%s pin %s (%s) on wire %s = %p", c.Name, p.Name, k, subK, n, sub.m[subK])
			} else if sub.m[subK] == nil {
				// Chip() makes sure that unknown pins can only be inputs.
				sub.m[subK] = s.Pin(False)
				// log.Printf("%s:%s pin %s (%s) on wire ??? = FALSE", c.Name, p.Name, k, subK)
			}
		}
		up := p.Mount(sub)
		impl.ups[i] = up
		if _, ok := up.(Wrapper); ok {
			continue
		}
		for _, k := range p.Outputs {
			subK := p.Pinout[k]
			if subK == "" {
				continue
			}
			if w := sub.m[subK]; w != nil {
				w.SetSource(up)
			}
		}
	}
	return impl
}

type chipImpl struct {
	ups []Updater
}

func (c *chipImpl) Update(clk bool) {
	for _, u := range c.ups {
		u.Update(clk)
	}
}

func (c *chipImpl) Unwrap() []Updater {
	return c.ups
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

// a pin is used by Chip() and identified by the part it belongs to and its name in that part's interface
type pin struct {
	p    int
	name string
}

const (
	typeUnknown = iota
	typeInput
	typeOutput
)

// node is a node in a wire (see wiring).
type node struct {
	name string  // wire name (propagates to outs)
	pin  pin     // pin connecting this node
	src  *node   // power source
	outs []*node // nodes powered by this node
	typ  int
}

func (n *node) isChipInput() bool {
	return n.pin.p < 0 && n.typ == typeInput
}
func (n *node) isPartInput() bool {
	return n.pin.p >= 0 && n.typ == typeInput
}
func (n *node) isOutput() bool {
	return n.typ == typeOutput
}
func (n *node) isChipOutput() bool {
	return n.typ == typeOutput && n.pin.p < 0
}

func (n *node) root() *node {
	for src := n.src; src != nil; n, src = src, n.src {
	}
	return n
}

func (n *node) setName(name string) {
	n.name = name
	for _, o := range n.outs {
		o.setName(name)
	}
}

// checkType asserts that the requested type change is a no-op.
// input -> unknown = input
// output -> unknown = output
// unk -> unk == unk
// anything else is illegal.
func (n *node) checkType(typ int) error {
	if typ != typeUnknown && n.typ != typ {
		return errors.New("cannot change pin type")
	}
	return nil
}

// wiring represents the wiring within a chip. It is organised as a set of node trees.
// Each node tree represents a wire.
// Pins are each assigned a node, and nodes are connected together to form a wire.
// Wires can have only one pin powering them (the root node, with n.src == nil),
// and branch out in sub-wires.
//
// There are six pin types:
//
//	             SRC? DST?
//	Chip Input    Y    N
//	Chip Output   Y    Y
//	Part Input    N    Y
//	Part Output   Y    N
//	Ephemeral     Y    Y
//
// plus constant inputs true, false and clk that are wired in as
// chip inputs.
//
type wiring map[pin]*node

func newWiring(ins, outs []string) wiring {
	wr := make(wiring, len(ins)+len(outs)+1)

	// add constant pins
	for _, pn := range cstPinNames {
		p := pin{-1, pn}
		wr[p] = &node{name: pn, pin: p, typ: typeInput}
	}

	for _, in := range ins {
		p := pin{-1, in}
		n := &node{name: in, pin: p, typ: typeInput}
		wr[p] = n
	}

	for _, out := range outs {
		p := pin{-1, out}
		n := &node{name: False, pin: p, typ: typeOutput}
		wr[p] = n
	}
	return wr
}

// connect wires pins src and dst (src being the pin powering the wire).
// sIName is the part's internal pin name for the source pin
func (wr wiring) connect(src pin, sType int, sIName string, dst pin, dType int) error {
	if dst.p < 0 {
		switch dst.name {
		case Clk:
			return errors.New("output pin connected to clock signal")
		case False:
			return errors.New("output pin connected to constant false input")
		case True:
			return errors.New("output pin connected to constant true input")
		}
	}

	ws := wr[src]
	if ws == nil {
		ws = &node{name: sIName, pin: src, typ: sType}
		wr[src] = ws
	} else if err := ws.checkType(sType); err != nil {
		return err

	}
	if ws.isPartInput() {
		return errors.New("part input pin used as output pin")
	}

	wd := wr[dst]
	if wd == nil {
		wd = &node{pin: dst, typ: dType}
		wr[dst] = wd
	} else if err := wd.checkType(dType); err != nil {
		return err
	}
	switch {
	case wd.isOutput() && wd.pin.p >= 0:
		return errors.New("part output pin used as output")
	case wd.isChipInput():
		return errors.New("chip input pin used as output")
	case wd.src == nil:
		wd.src = ws
	default:
		return errors.New("output pin already used as output")
	}
	wd.setName(ws.name)
	ws.outs = append(ws.outs, wd)
	return nil
}

func (wr wiring) wireName(p pin) string {
	if n := wr[p]; n != nil {
		return n.name
	}
	return ""
}

// prune should be called after adding all connections.
// It removes unconnected chip pins and ephemeral pins by establishing direct
// connections between parts and I/O pins and assigns names to individual wires.
//
func (wr wiring) prune() error {
	for k, n := range wr {
		// remove input pins with no outs
		if n.isChipInput() && len(n.outs) == 0 {
			delete(wr, k)
			continue
		}
		// remove unconnected output pins
		if n.isChipOutput() && n.src == nil && len(n.outs) == 0 {
			delete(wr, k)
			continue
		}

		// Error on ephemeral pins with no source or dest.
		// error on output pins with dests within the chip but no src.
		if (n.typ == typeUnknown || n.isChipOutput()) && n.src == nil {
			return errors.New("pin " + n.pin.name + " not connected to any output")
		}
		if n.typ == typeUnknown && len(n.outs) == 0 {
			return errors.New("pin " + n.pin.name + " not connected to any input")
		}

		// remove temporary pins.
		// input pins can safely be ignored since len(n.outs) is 0 for them.
		// Inspect every "next" output pin in the wire chain.
		for i := 0; i < len(n.outs); {
			next := n.outs[i]
			if len(next.outs) == 0 {
				i++
				continue
			}
			// there is a wire chain: n -> next -> next.outs
			// merge it into n.outs = n.outs + next.outs
			for _, o := range next.outs {
				o.src = n
				o.setName(n.name)
			}
			n.outs = append(n.outs, next.outs...)
			next.outs = nil
			// remove ephemeral internal chip pins
			if next.pin.p < 0 && next.typ == typeUnknown {
				// delete next
				n.outs[i] = n.outs[len(n.outs)-1]
				n.outs = n.outs[:len(n.outs)-1]
				delete(wr, next.pin)
				continue
			}
			i++
		}
	}
	return nil
}
