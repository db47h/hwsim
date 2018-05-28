package hdl

import (
	"errors"
	"sync"
)

// TODO:
//
//	- check how map[x]y arguments are re-used/saved by the callees.
//	- find a way to check the wiring of a chip. eg. If one part is Not(W{"in": "a", "out": "unused"}).
//	  Chip() should be able to find out that the internal pin "unused" is indeed unused and report it.
//	  Unused pins should be omitted and automatically grounded.
//	- refactor names like pinout and othe wire-y related things to reflect what they truly are.
//	- handle buses. Chip i/o pin spec should accept thins like a[8] (i.e. an 8 pin bus), while wiring specs
//	  should accept things like: W{"my4biBus": "input[0..3]"}

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
// The provided pins maps the part's internal pin names to pin numbers in a circuit.
//
type BuildFn func(pins map[string]int, c *Circuit) ([]Updater, error)

// A PartSpec represents a part specification.
//
type PartSpec struct {
	In  []string
	Out []string

	// TODO: review all implementations for proper error messages.
	//
	Build BuildFn // Build function.
}

// Wire returns a wired part based on the given spec and wiring.
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

// A NewPartFunc is a function that takes a number of named pins and returns a new Chip.
//
type NewPartFunc func(pins W) Part

type chip struct {
	PartSpec
	parts []Part
}

func (c *chip) build(pins map[string]int, cc *Circuit) ([]Updater, error) {
	var updaters []Updater
	if len(pins) < cstCount {
		panic("invalid pin map")
	}
	// collect parts
	for _, p := range c.parts {
		ppins := cstPins()
		for in, ex := range p.Wires() {
			var n int
			var ok bool
			if n, ok = pins[ex]; !ok {
				n = cc.Alloc()
				pins[ex] = n
			}
			ppins[in] = n
		}
		pup, err := p.Spec().Build(ppins, cc)
		if err != nil {
			return nil, err
		}
		updaters = append(updaters, pup...)
	}
	return updaters, nil
}

// Chip combines existing components into a new component.
//
// An Xor gate could be created like this:
//
//	xor := Chip(
//		[]string{"a", "b"},
//		[]string{"out"},
//		[]hdl.Part{
//			hdl.Not("a", "nota"),
//			hdl.Not("b", "notb"),
//			hdl.And("a", "notb", "w1"),
//			hdl.And("b", "nota", "w2"),
//			hdl.Or("w1", "w2", "out")
//		})
//
// The returned function can be used to wire the new component into others:
//
//	xnor := hdl.Chip(
//		[]string{"a", "b"},
//		[]string{"out"},
//		[]hdl.Part{
//			xor("a", "b", "xorAB"),
//			hdl.Not("xorAB", "out"),
//		})
//
func Chip(inputs []string, outputs []string, parts []Part) NewPartFunc {
	// TODO check in/out pins here
	c := &chip{
		PartSpec{
			In:  inputs,
			Out: outputs,
		},
		parts,
	}
	c.Build = c.build
	return c.Wire
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
	wrap := Chip(nil, nil, ps)(nil)
	ups, err := wrap.Spec().Build(cstPins(), cc)
	if err != nil {
		return nil, err
	}
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
