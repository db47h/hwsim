package hdl

import (
	"strconv"
	"strings"
	"sync"

	"github.com/pkg/errors"
)

// TODO:
//
//	- handle buses. Chip i/o pin spec should accept thins like a[8] (i.e. an 8 pin bus), while wiring specs
//	  should accept things like: W{"my4biBus": "input[0..3]"}
//	- check how map[x]y arguments are re-used/saved by the callees.
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

// An Component is component in a circuit that can Get and Set states.
//
type Component func(c *Circuit)

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

// BusPinName returns the pin name for the n-th bit of the named bus.
//
func BusPinName(name string, bit int) string {
	return name + "[" + strconv.Itoa(bit) + "]"
}

// Bus returns individual pin names for the specified bus name and size.
//
func Bus(name string, bits int) []string {
	out := make([]string, 0, bits)
	for i := 0; i < bits; i++ {
		out = append(out, BusPinName(name, i))
	}
	return out
}

// ExpandBus returns a copy of the pin names with buses expanded as individual
// pin names. e.g. "in[2]" will be expanded to "in[0]", "in[1]"
//
func ExpandBus(pins ...string) []string {
	out := make([]string, 0, len(pins))
	for _, n := range pins {
		i := strings.IndexRune(n, '[')
		if i < 0 {
			out = append(out, n)
			continue
		}
		t := n[i+1:]
		n = n[:i]
		i = strings.IndexRune(t, ']')
		if i < 0 {
			panic("no terminamting ] in bus specification")
		}
		l, err := strconv.Atoi(t[:i])
		if err != nil {
			panic(err)
		}
		for i := 0; i < l; i++ {
			out = append(out, BusPinName(n, i))
		}
	}
	return out
}

// A Socket maps a part's pin names to pin numbers in a circuit.
//
type Socket map[string]int

// Get returns the pin number allocated to the given pin name.
//
func (s Socket) Get(name string) int { return s[name] }

// GetBus returns the pin numbers allocated to the given bus name.
//
func (s Socket) GetBus(name string) []int {
	out := make([]int, 0)
	i := 0
	for {
		n, ok := s[BusPinName(name, i)]
		if !ok {
			break
		}
		out = append(out, n)
		i++
	}
	return out
}

// A MountFn mounts a part into the given socket and circuit.
// In effect, it creates a new instance of a part as an Updater slice.
//
type MountFn func(c *Circuit, pins Socket) []Component

// A PartSpec represents a part specification.
//
type PartSpec struct {
	Name string   // Part name
	In   []string // Input pin names
	Out  []string // Output pin names

	Mount MountFn // Mount function.
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

func cstPins() Socket {
	return Socket{False: cstFalse, True: cstTrue}
}

// Circuit is a runable circuit simulation.
//
type Circuit struct {
	s0    []bool // wire states frame #0
	s1    []bool // wire states frame #1
	cs    []Component
	count int // wire count
}

// NewCircuit builds a new circuit based on the given parts.
//
func NewCircuit(ps []Part) (*Circuit, error) {
	// new circuit with room for constant value pins.
	cc := &Circuit{count: cstCount}
	wrap, err := Chip("CIRCUIT", nil, nil, ps)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create chip wrapper")
	}
	ups := wrap(nil).Spec().Mount(cc, cstPins())
	cc.cs = ups
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

// Get returns the state of a given pin.
//
func (c *Circuit) Get(n int) bool {
	return c.s0[n]
}

// Set sets the state of a given pin.
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

// Update advances a simulation by one step.
//
func (c *Circuit) Update(workers int) {
	// set constant pins
	c.s0[cstFalse] = false
	c.s0[cstTrue] = true

	if workers <= 0 {
		for _, u := range c.cs {
			u(c)
		}
		c.s0, c.s1 = c.s1, c.s0
		return
	}

	var wg sync.WaitGroup
	p := c.cs
	l := len(p) / workers
	if l*workers < len(p) {
		l++
	}
	for len(p) > 0 {
		wg.Add(1)
		l = min(l, len(p))
		go func(cs []Component) {
			for _, f := range cs {
				f(c)
			}
			wg.Done()
		}(p[:l])
		p = p[l:]
	}
	wg.Wait()
	c.s0, c.s1 = c.s1, c.s0
}
