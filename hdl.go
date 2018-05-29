package hdl

import (
	"sync"

	"github.com/pkg/errors"
)

// TODO:
//
//	- check how map[x]y arguments are re-used/saved by the callees.
//	- refactor names like pinout and othe wire-y related things to reflect what they truly are.
//	- handle buses. Chip i/o pin spec should accept thins like a[8] (i.e. an 8 pin bus), while wiring specs
//	  should accept things like: W{"my4biBus": "input[0..3]"}
//	- Add more built-in things.

// CHIP Xor {
// 	IN a, b;
// 	OUT out;
// 	PARTS:
// 	Not(in=a, out=nota);
// 	Not(in=b, out=notb);
// 	And(a=a, b=notb, out=w1);
// 	And(a=nota, b=b, out=w2);
// 	Or(a=w1, b=w2, out=out);
// }

// CHIP Mux16 {
// 	IN a[16], a[16], sel;
// 	OUT out[16];
// 	BUILTIN Mux; // Reference to builtIn/Mux.class, that
// 	// implements both the Mux.hdl and the
// 	// Mux16.hdl built-in chips.
// }

// An Updater is an updatable component in a circuit
// TODO: rename that !
//
type Updater func(c *Circuit)

// W is a set of wires, connecting a part's I/O pins (the map key) to pins in its container.
//
type W map[string]string

// Copy returns a copy of w.
//
func (w W) Copy() W {
	t := make(W, len(w))
	for k, v := range w {
		t[k] = v
	}
	return t
}

// Wire returns a copy of w where keys for the in/out pins are guaranteed to be
// present (unconnected pins are set to False). If w contains pin names
// not present in either in or out, it will return an error.
//
// This function should be called first in any function behaving like a NewPartFunc.
//
func (w W) Wire(in, out []string) (W, error) {
	w = w.Copy()
	wires := make(W, len(w))
	for _, name := range in {
		if outer, ok := w[name]; ok {
			wires[name] = outer
			delete(w, name)
		} else {
			wires[name] = False
		}
	}
	for _, name := range out {
		if outer, ok := w[name]; ok {
			wires[name] = outer
			delete(w, name)
		} else {
			wires[name] = False
		}
	}
	// check unknown pins
	for name := range w {
		return nil, errors.New("unknown pin \"" + name + "\"")
	}
	return wires, nil
}

// A BuildFn creates a new instance of a part as an Updater slice.
// The provided pins map the part's internal pin names to pin numbers in a circuit.
//
// TODO: document if pins is modified by the callee
//
type BuildFn func(pins map[string]int, c *Circuit) []Updater

// A PartSpec represents a part specification.
//
type PartSpec struct {
	Name string   // Part name
	In   []string // Input pin names
	Out  []string // Output pin names

	// TODO: review all implementations for proper error messages.
	//
	Build BuildFn // Build function.
}

// Wire returns a wired part based on the given spec and wiring.
//
func (p *PartSpec) Wire(w W) Part {
	w, err := w.Wire(p.In, p.Out)
	if err != nil {
		panic(err)
	}
	return &part{p, w}
}

// A Part wraps a part specification together with its wiring
// in a container part.
//
type Part interface {
	Spec() *PartSpec
	Wires() W
}

type part struct {
	p *PartSpec
	w W
}

func (p *part) Spec() *PartSpec {
	return p.p
}

func (p *part) Wires() W {
	return p.w
}

// A NewPartFunc is a function that takes a set of wires and returns a new Part.
//
type NewPartFunc func(pins W) Part

type chip struct {
	PartSpec
	parts []Part
}

func (c *chip) build(pins map[string]int, cc *Circuit) []Updater {
	var updaters []Updater
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
		pup := p.Spec().Build(ppins, cc)
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
// wire the new part into others:
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
	// check that no outputs are connected together.
	outs := make(map[string]int)
	// add our inputs
	for _, i := range inputs {
		outs[i] = 1
	}
	// for each part, add its outputs
	for _, p := range parts {
		w := p.Wires()
		for _, o := range p.Spec().Out {
			n := w[o]
			if n == False {
				// nil or unconnected output, ignore.
				continue
			}
			if n == True {
				return nil, errors.New(p.Spec().Name + " pin " + o + " connected to constant True input")
			}
			if _, ok := outs[n]; ok {
				return nil, errors.New("pin " + n + " connected to more than one output")
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
		for _, o := range p.Spec().In {
			n := w[o]
			if n == True {
				continue
			}
			if cnt, ok := outs[n]; ok {
				outs[n] = cnt + 1
				continue
			}
			return nil, errors.New(p.Spec().Name + " pin " + o + ":" + n + " not connected to any output")
		}
	}
	// log.Print(outs)
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
	c.Build = c.build
	return c.Wire, nil
}

// Constant input pin names.
//
var (
	True  = "true"
	False = "false"
	GND   = "false"
)

const (
	cstFalse = iota
	cstTrue
	cstCount
)

func cstPins() map[string]int {
	return map[string]int{False: cstFalse, True: cstTrue}
}

// Circuit is a runable circuit simulation.
//
type Circuit struct {
	s0    []bool // wire states frame #0
	s1    []bool // wire states frame #1
	parts []Updater
	count int // wire count
}

// NewCircuit returns a new circuit based on the given chips.
//
func NewCircuit(ps []Part) (*Circuit, error) {
	// new circuit with room for constant value pins.
	cc := &Circuit{count: cstCount}
	wrap, err := Chip("CIRCUIT", nil, nil, ps)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create chip wrapper")
	}
	ups := wrap(nil).Spec().Build(cstPins(), cc)
	cc.parts = ups
	cc.s0 = make([]bool, cc.count)
	cc.s1 = make([]bool, cc.count)
	return cc, nil
}

// Alloc allocates a pin and returns its number.
//
func (c *Circuit) Alloc() int {
	cnt := c.count
	c.count++
	return cnt
}

// Get returns the state of a given input/output.
//
func (c *Circuit) Get(n int) bool {
	return c.s0[n]
}

// Set sets the state of a given input/output.
//
func (c *Circuit) Set(n int, s bool) {
	c.s1[n] = s
}

func min(a, b int) int {
	if a <= b {
		return a
	}
	return b
}

// Update advances a chip simulation by one step.
//
func (c *Circuit) Update(workers int) {
	// set constant pins
	c.s0[cstFalse] = false
	c.s0[cstTrue] = true

	if workers <= 0 {
		for _, u := range c.parts {
			u(c)
		}
		c.s0, c.s1 = c.s1, c.s0
		return
	}

	var wg sync.WaitGroup
	p := c.parts
	l := len(p) / workers
	if l*workers < len(p) {
		l++
	}
	for len(p) > 0 {
		wg.Add(1)
		l = min(l, len(p))
		go func(parts []Updater) {
			for _, u := range parts {
				u(c)
			}
			wg.Done()
		}(p[:l])
		p = p[l:]
	}
	wg.Wait()
	c.s0, c.s1 = c.s1, c.s0
}
