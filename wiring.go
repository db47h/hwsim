package hdl

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// W is a set of wires, connecting a part's I/O pins (the map key) to pins in its container.
//
type W map[string]string

// wire builds a wire map by expanding bus ranges.
//
func (w W) expand(in, out []string) (map[string][]string, error) {
	r := make(map[string][]string)
	for k, v := range w {
		ks, err := expandRange(k)
		if err != nil {
			return nil, errors.Wrap(err, "expand key "+k)
		}
		vs, err := expandRange(v)
		if err != nil {
			return nil, errors.Wrap(err, "expand value "+v)
		}
		switch {
		case len(ks) == len(vs):
			// many to many
			for i := range ks {
				r[ks[i]] = []string{vs[i]}
			}
		case len(ks) == 1:
			// one to nany
			r[k] = vs
		case len(vs) == 1:
			// many to one
			for _, k := range ks {
				r[k] = vs
			}
		default:
			return nil, errors.New("pin count mismatch in pin mapping: " + k + ":" + v)
		}
	}
	return r, nil
}

func expandRange(name string) ([]string, error) {
	i := strings.IndexRune(name, '[')
	if i < 0 {
		return []string{name}, nil
	}
	bus := name[:i]
	n := name[i+1:]
	i = strings.Index(n, "..")
	if i < 0 {
		return []string{name}, nil
	}
	start, err := strconv.Atoi(n[:i])
	if err != nil {
		return nil, err
	}
	n = n[i+2:]
	i = strings.IndexRune(n, ']')
	if i < 0 {
		return nil, errors.New("no terminamting ] in bus range")
	}
	end, err := strconv.Atoi(n[:i])
	if err != nil {
		return nil, err
	}
	r := make([]string, 0, end-start+1)
	for i := start; i <= end; i++ {
		r = append(r, BusPinName(bus, i))
	}
	return r, nil
}

// a pin is identified by the part it belongs to and its name in that part's interface
type pin struct {
	p    Part
	name string
}

func (p *pin) String() string {
	if p.p == nil {
		return p.name
	}
	return fmt.Sprintf("%s.%s", p.p.Spec().Name, p.name)
}

func (p *pin) eq(prt Part, n string) bool {
	return p.p == prt && p.name == n
}

const (
	typeUnknown = iota
	typeInput
	typeOutput
)

type node struct {
	name string // chip internal pin name
	pin  pin
	outs []*node
	org  *node // pin feeding that node
	typ  int
}

func (n *node) String() string {
	var outs strings.Builder

	if n.name != "" {
		outs.WriteString(n.name)
		outs.WriteString(": ")
	}

	outs.WriteString(n.pin.String())
	if n.isInput() {
		outs.WriteString(" (IN)")
	} else if n.isOutput() {
		outs.WriteString(" (OUT)")
	}
	outs.WriteString("->")

	outs.WriteByte('[')
	for i, on := range n.outs {
		if i > 0 {
			outs.WriteString(", ")
		}
		outs.WriteString(on.pin.String())
	}
	outs.WriteByte(']')

	if n.org != nil {
		outs.WriteString(" (org ")
		outs.WriteString(n.org.pin.String())
		outs.WriteByte(')')
	}
	return outs.String()
}

func (n *node) isInput() bool {
	return n.typ == typeInput
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

type wiring map[pin]*node

func newWiring(ins, outs []string) (wr wiring, inputRoot *node) {
	wr = make(wiring, len(ins)+len(outs)+1)
	// inputRoot serves as a parent marker for chip inputs.
	inputRoot = &node{pin: pin{nil, "__INPUT__"}, outs: make([]*node, 0, len(ins)), typ: typeInput}

	for _, in := range ins {
		p := pin{nil, in}
		n := &node{pin: p, org: inputRoot, typ: typeUnknown}
		wr[p] = n
		inputRoot.outs = append(inputRoot.outs, n)
	}
	// add true and false as chip inputs
	p := pin{nil, True}
	wr[p] = &node{pin: p, org: inputRoot, typ: typeUnknown}
	p = pin{nil, False}
	wr[p] = &node{pin: p, org: inputRoot, typ: typeUnknown}

	for _, out := range outs {
		p := pin{nil, out}
		n := &node{pin: p, org: nil, typ: typeOutput}
		wr[p] = n
	}
	return wr, inputRoot
}

func (wr wiring) add(ip Part, iname string, iType int, op Part, oname string, oType int) error {
	in := pin{ip, iname}
	if op == nil {
		switch oname {
		case False:
			return nil
		case True:
			return errors.New("output pin " + in.String() + " connected to constant \"true\" input")
		}
	}
	out := pin{op, oname}
	wi := wr[in]
	if wi == nil {
		wi = &node{pin: in, typ: iType}
		wr[in] = wi
	}
	wo := wr[out]
	switch {
	case wo == nil:
		wo = &node{pin: out, org: wi, typ: oType}
		wr[out] = wo
	case wo.org == nil:
		wo.org = wi
	default:
		return errors.New("pin " + in.String() + ":" + out.String() + ": output pin already used by " + wo.org.pin.String() + ":" + wo.pin.String())
	}
	wi.outs = append(wi.outs, wo)
	return nil
}

// check wiring:
func (wr wiring) check(root *node) error {
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
					if next.pin.p == nil && !next.isOutput() {
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
			if n.pin.p == nil && !n.isOutput() {
				return errors.New("pin " + n.pin.String() + " not connected to any input")
			}
		} else if n.org == nil && !n.isOutput() {
			return errors.New("pin " + n.pin.String() + " not connected to any output")
		}

		if n.name == "" {
			t := n
			for prev := t.org; prev != nil && prev != root; t, prev = prev, t.org {
			}
			if t.org == nil {
				t.setName("__internal__" + strconv.Itoa(i))
			} else {
				t.setName(t.pin.name)
			}
			i++
		}
	}
	return nil
}
