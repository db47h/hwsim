package hwsim

import (
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/pkg/errors"
)

// A Component is a component in a circuit that can Get and Set states.
//
type Component func(c *Circuit)

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
			out = append(out, busPinName(n, i))
		}
	}
	return out
}

// A MountFn mounts a part into the given socket.
//
type MountFn func(s *Socket) []Component

// A PartSpec represents a part specification.
//
type PartSpec struct {
	Name string // Part name
	In          // Input pin names
	Out         // Output pin names
	// Pinout maps the input and output pin names (public) of a part to internal (private) names
	// for them. If nil, the In/Out values will be used.
	// In a MountFn, only private pin names must be used when calling Socket methods.
	Pinout map[string]string

	Mount MountFn // Mount function.
}

// Wire is a NewPartFunc that wraps p with the given connections into a PartWiring.
//
func (p *PartSpec) Wire(connections string) Part {
	ex, err := ParseConnections(connections)
	if err != nil {
		panic(err)
	}
	if p.Pinout == nil {
		p.Pinout = make(map[string]string)
		for _, i := range p.In {
			p.Pinout[i] = i
		}
		for _, o := range p.Out {
			p.Pinout[o] = o
		}
	}
	return Part{p, ex}
}

// MakePart returns a NewPartFunc for the given PartSpec. It is a utility
// wrapper around PartSpec.Wire.
//
func MakePart(p *PartSpec) NewPartFn {
	return p.Wire
}

// In is a slice of input pin names.
//
type In []string

// Out is a slice of output pin names.
//
type Out []string

// A NewPartFn is a function that takes a wiring configuration and returns a new PartWiring.
// See ParseWiring for the syntax of the wiring configuration.
//
type NewPartFn func(wiring string) Part

// Parts is a convenience wrapper for []Part.
//
type Parts []Part

// A Part wraps a part specification together with its connections
// within a host chip.
//
type Part struct {
	*PartSpec
	Connections
}

// Circuit is a runable circuit simulation.
//
type Circuit struct {
	s0    []bool // wire states frame #0
	s1    []bool // wire states frame #1
	cs    []Component
	count int  // wire count
	tpc   uint // ticks per clock cycle
	tick  uint

	wc []chan struct{}
	wg sync.WaitGroup
}

// NewCircuit builds a new circuit based on the given parts.
//
func NewCircuit(workers int, ticksPerCycle uint, ps Parts) (*Circuit, error) {
	if len(ps) == 0 {
		return nil, errors.New("empty part list")
	}

	if ticksPerCycle < 2 {
		ticksPerCycle = 2
	}
	ticksPerCycle--
	ticksPerCycle |= ticksPerCycle >> 1
	ticksPerCycle |= ticksPerCycle >> 2
	ticksPerCycle |= ticksPerCycle >> 4
	ticksPerCycle |= ticksPerCycle >> 8
	ticksPerCycle |= ticksPerCycle >> 16
	ticksPerCycle |= ticksPerCycle >> 32
	ticksPerCycle++

	// new circuit with room for constant value pins.
	cc := &Circuit{count: cstCount, tpc: ticksPerCycle}
	wrap, err := Chip("CIRCUIT", nil, nil, ps)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create chip wrapper")
	}
	ups := wrap("").Mount(newSocket(cc))
	cc.cs = ups
	cc.s0 = make([]bool, cc.count)
	cc.s1 = make([]bool, cc.count)
	// init constant pins
	cc.s0[cstClk] = true
	cc.s0[cstFalse] = false
	cc.s0[cstTrue] = true
	cc.s1[cstFalse] = false
	cc.s1[cstTrue] = true

	// workers
	if workers == 0 {
		workers = runtime.NumCPU()
	}
	if workers == 0 {
		workers = 1
	}

	// # of updaters per worker
	size := len(ups) / workers
	if size*workers < len(ups) {
		size++
	}
	for len(ups) > 0 {
		size = min(size, len(ups))
		wc := make(chan struct{}, 1)
		cc.wc = append(cc.wc, wc)
		go worker(cc, ups[:size], wc)
		ups = ups[size:]
	}

	return cc, nil
}

// Dispose releases all resources for a circuit.
//
func (c *Circuit) Dispose() {
	c.wg.Add(len(c.wc))
	for _, wc := range c.wc {
		close(wc)
	}
	c.wg.Wait()
}

func worker(c *Circuit, cs []Component, wc <-chan struct{}) {
	for {
		_, ok := <-wc
		if !ok {
			c.wg.Done()
			return
		}
		for _, f := range cs {
			f(c)
		}
		c.wg.Done()
	}
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

// Toggle toggles the state of a given pin.
//
func (c *Circuit) Toggle(n int) {
	c.s1[n] = !c.s0[n]
}

func min(a, b int) int {
	if a <= b {
		return a
	}
	return b
}

// Step advances a simulation by one step.
//
func (c *Circuit) Step() {
	c.wg.Add(len(c.wc))
	for _, wc := range c.wc {
		wc <- struct{}{}
	}

	if c.s0[cstFalse] || !c.s0[cstTrue] {
		panic("true or false constants have been overwritten")
	}

	// update clock signal
	tick := c.tick + 1
	if tick&(c.tpc-1) == 0 {
		c.Set(cstClk, true)
	} else if tick&(c.tpc/2-1) == 0 {
		c.Set(cstClk, false)
	} else {
		c.Set(cstClk, c.Get(cstClk))
	}

	c.wg.Wait()
	c.tick = tick
	c.s0, c.s1 = c.s1, c.s0
}

// Tick runs the simulation until the beginning of the next half clock cycle.
//
func (c *Circuit) Tick() {
	for c.Get(cstClk) {
		c.Step()
	}
}

// Tock runs the simulation until the beginning of the next clock cycle.
// Once Tock returns, the output of clocked components should have stabilized.
//
func (c *Circuit) Tock() {
	for !c.Get(cstClk) {
		c.Step()
	}
}

// TickTock runs the simulation for a whole clock cycle.
//
func (c *Circuit) TickTock() {
	c.Tick()
	c.Tock()
}
