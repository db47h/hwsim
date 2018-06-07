package hwsim

import (
	"strconv"

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
// The map value is a slice because any output pin of a part can
// be connected to more than one other pin within the chip.
//
type Connections map[string][]string

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

func (n *node) setName(name string) {
	n.name = name
	for _, o := range n.outs {
		o.setName(name)
	}
}

// setType possibly changes the type of a node.
// it may result in turning n into an input or output type,
// in which case the node must be checked for correctnes.
// add() already does these checks.
func (n *node) setType(typ int) error {
	if typ != typeUnknown && n.typ != typ {
		if n.typ == typeUnknown {
			n.typ = typ
		} else if n.typ == typeInput {
			return errors.New("cannot turn input pin into an output pin")
		} else {
			return errors.New("cannot turn output pin into an input pin")
		}
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
		wr[p] = &node{pin: p, typ: typeInput}
	}

	for _, in := range ins {
		p := pin{-1, in}
		n := &node{pin: p, typ: typeInput}
		wr[p] = n
	}

	for _, out := range outs {
		p := pin{-1, out}
		n := &node{pin: p, typ: typeOutput}
		wr[p] = n
	}
	return wr
}

// connect wires pins src and dst (src being the pin powering the wire).
func (wr wiring) connect(src pin, sType int, dst pin, dType int) error {
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
		ws = &node{pin: src, typ: sType}
		wr[src] = ws
	} else if err := ws.setType(sType); err != nil {
		return err
	}
	if ws.isPartInput() {
		return errors.New("part input pin used as output pin")
	}

	wd := wr[dst]
	if wd == nil {
		wd = &node{pin: dst, typ: dType}
		wr[dst] = wd
	} else if err := wd.setType(dType); err != nil {
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

	ws.outs = append(ws.outs, wd)
	return nil
}

func (wr wiring) wireName(p pin) string {
	if n := wr[p]; n != nil {
		return n.name
	}
	return ""
}

// pruneEphemeral should be called after adding all connections.
// It removes ephemeral pins by establishing direct connections
// between parts and I/O pins and assigns names to individual wires.
//
func (wr wiring) pruneEphemeral() error {
	wireNum := 0
	for k, n := range wr {
		// Error on ephemeral pins with no source or dest.
		if n.typ == typeUnknown {
			if n.src == nil {
				return errors.New("pin " + n.pin.name + " not connected to any output")
			}
			if len(n.outs) == 0 {
				return errors.New("pin " + n.pin.name + " not connected to any input")
			}
		}
		// remove input pins with no outs
		if n.isChipInput() && len(n.outs) == 0 {
			delete(wr, k)
			continue
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
			}
			n.outs = append(n.outs, next.outs...)
			next.outs = nil
			// remove ephemeral internal chip pins
			if next.pin.p < 0 && next.typ == typeUnknown {
				// delete next
				n.outs[i] = n.outs[len(n.outs)-1]
				n.outs = n.outs[:len(n.outs)-1]
				delete(wr, next.pin)
			} else {
				i++
			}
		}

		// assign a wire name to the pin tree
		if n.name == "" {
			t := n
			for prev := t.src; prev != nil; t, prev = prev, t.src {
			}
			if t.isChipInput() {
				t.setName(t.pin.name)
			} else {
				t.setName("__" + strconv.Itoa(wireNum))
			}
			wireNum++
		}
	}

	return nil
}
