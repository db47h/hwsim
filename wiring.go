package hwsim

import (
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// W is a set of wires, connecting a part's I/O pins (the map key) to pins in its container.
//
type W map[string]string

// wire builds a wire map by expanding bus ranges.
//
func (w W) expand() (map[string][]string, error) {
	r := make(map[string][]string)
	for k, v := range w {
		if k == "" || v == "" {
			return nil, errors.New("invalid pin mapping " + k + ":" + v)
		}
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
	if bus == "" {
		return nil, errors.New("empty bus name")
	}
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
	p    int
	name string
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

func newWiring(ins In, outs Out) (wr wiring, inputRoot *node) {
	wr = make(wiring, len(ins)+len(outs)+1)
	// inputRoot serves as a parent marker for chip inputs.
	inputRoot = &node{pin: pin{-1, "__INPUT__"}, outs: make([]*node, len(ins)), typ: typeInput}

	// add true and false as chip inputs
	p := pin{-1, True}
	wr[p] = &node{pin: p, org: inputRoot, typ: typeUnknown}
	p = pin{-1, False}
	wr[p] = &node{pin: p, org: inputRoot, typ: typeUnknown}

	for i, in := range ins {
		p := pin{-1, in}
		n := &node{pin: p, org: inputRoot, typ: typeUnknown}
		wr[p] = n
		inputRoot.outs[i] = n
	}

	for _, out := range outs {
		p := pin{-1, out}
		n := &node{pin: p, org: nil, typ: typeOutput}
		wr[p] = n
	}
	return wr, inputRoot
}

func (wr wiring) add(in pin, iType int, out pin, oType int) error {
	if out.p < 0 {
		switch out.name {
		case False:
			return nil
		case Clk:
			return errors.New("output pin connected to clock signal")
		case True:
			return errors.New("output pin connected to constant \"true\" input")
		}
	}
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
		return errors.New("output pin already used as output or is one of the chip's input pin")
	}
	wi.outs = append(wi.outs, wo)
	return nil
}
