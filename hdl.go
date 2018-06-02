package hdl

import (
	"strconv"
	"strings"
	"sync"

	"github.com/pkg/errors"
)

// A Component is a component in a circuit that can Get and Set states.
//
type Component func(c *Circuit)

// BusPinName returns the pin name for the n-th bit of the named bus.
//
func BusPinName(name string, bit int) string {
	return name + "[" + strconv.Itoa(bit) + "]"
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

// A MountFn mounts a part into the given socket.
// In effect, it creates a new instance of a part as []Component slice.
//
type MountFn func(s *Socket) []Component

// A PartSpec represents a part specification.
//
type PartSpec struct {
	Name string   // Part name
	In   []string // Input pin names
	Out  []string // Output pin names
	// Pinout maps input/output pin names to a part's internal names
	// for them. If nil, the In/Out values will be used.
	Pinout W

	Mount MountFn // Mount function.
}

// Wire is a NewPartFunc that returns a wired part based on the given spec and wiring.
//
func (p *PartSpec) Wire(w W) Part {
	ex, err := w.expand()
	if err != nil {
		panic(err)
	}
	if p.Pinout == nil {
		p.Pinout = make(W)
		for _, i := range p.In {
			p.Pinout[i] = i
		}
		for _, o := range p.Out {
			p.Pinout[o] = o
		}
	}
	return &part{p, ex}
}

// A Part wraps a part specification together with its wiring
// in a container part.
//
type Part interface {
	Spec() *PartSpec
	wires() map[string][]string
}

type part struct {
	p *PartSpec
	w map[string][]string // Initial wiring
}

func (p *part) Spec() *PartSpec {
	return p.p
}

func (p *part) wires() map[string][]string {
	return p.w
}

// A NewPartFunc is a function that takes a set of wires and returns a new Part.
//
type NewPartFunc func(pins W) Part

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
	ups := wrap(nil).Spec().Mount(newSocket(cc))
	cc.cs = ups
	cc.s0 = make([]bool, cc.count)
	cc.s1 = make([]bool, cc.count)
	return cc, nil
}

// alloc allocates a pin and returns its number.
//
func (c *Circuit) allocPin() int {
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
