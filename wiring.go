package hwsim

import (
	"github.com/pkg/errors"
)

// Constant input pin names. These pins can only be connected to the input pins of a chip.
//
// They are reserved names and should not be used as input or output names in
// custom chips.
//
var (
	False       = "false" // always false input
	True        = "true"  // alwyas true input
	Clk         = "clk"   // clock signal. True during Tick, False during Tock.
	cstPinNames = [...]string{"false", "true", "clk"}
)

const (
	cstFalse = iota
	cstTrue
	cstClk
	cstCount
)

// Connections represents the connections between the pins of a part (the map
// keys) to other pins in its host chip (the map values).
//
type Connections []Connection

// A Connection represents a connection between the pin PP of a part and
// the pins CP in its host chip.
//
type Connection struct {
	PP string
	CP []string
}

// a pin is identified by the part it belongs to and its name in that part's interface
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

func newWiring(ins Inputs, outs Outputs) wiring {
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
