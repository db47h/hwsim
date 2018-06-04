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
	Name string // Part name
	In          // Input pin names
	Out         // Output pin names
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

// MakePart returns a NewPartFunc for the given PartSpec. It is a utility
// wrapper around PartSpec.Wire.
//
func MakePart(p *PartSpec) NewPartFn {
	return p.Wire
}

// Parts is a convenience wrapper for []Part.
//
type Parts []Part

// In is a slice of input pin names.
//
type In []string

// Out is a slice of output pin names.
//
type Out []string

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

// A NewPartFn is a function that takes a set of wires and returns a new Part.
//
type NewPartFn func(wires W) Part

// Circuit is a runable circuit simulation.
//
type Circuit struct {
	s0    []bool // wire states frame #0
	s1    []bool // wire states frame #1
	cs    []Component
	count int // wire count
	tpc   int // ticks per clock cycle

	wc []chan struct{}
	wg sync.WaitGroup
}

// NewCircuit builds a new circuit based on the given parts.
//
func NewCircuit(workers int, ticksPerCycle int, ps Parts) (*Circuit, error) {
	if len(ps) == 0 {
		return nil, errors.New("empty part list")
	}

	if ticksPerCycle > 0 {
		ticksPerCycle--
		ticksPerCycle |= ticksPerCycle >> 1
		ticksPerCycle |= ticksPerCycle >> 2
		ticksPerCycle |= ticksPerCycle >> 4
		ticksPerCycle |= ticksPerCycle >> 8
		ticksPerCycle |= ticksPerCycle >> 16
		ticksPerCycle |= ticksPerCycle >> 32
		ticksPerCycle++
	} else {
		ticksPerCycle = 256
	}

	// new circuit with room for constant value pins.
	cc := &Circuit{count: cstCount, tpc: ticksPerCycle}
	wrap, err := Chip("CIRCUIT", nil, nil, ps)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create chip wrapper")
	}
	ups := wrap(nil).Spec().Mount(newSocket(cc))
	ups = append(ups, clock(cc))
	cc.cs = ups
	cc.s0 = make([]bool, cc.count)
	cc.s1 = make([]bool, cc.count)

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

// clock returns a clock for this circuit
//
func clock(c *Circuit) Component {
	ticks := 0
	hafTicks := c.tpc/2 - 1
	return func(c *Circuit) {
		ticks++
		if ticks&hafTicks == 0 {
			c.Toggle(cstClk)
			ticks = 0
		} else {
			c.Set(cstClk, c.Get(cstClk))
		}
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
	// set constant pins
	c.s0[cstFalse] = false
	c.s0[cstTrue] = true

	c.wg.Add(len(c.wc))
	for _, wc := range c.wc {
		wc <- struct{}{}
	}
	c.wg.Wait()
	c.s0, c.s1 = c.s1, c.s0
}

// Tick runs the simulation until the end of a true clock signal.
//
func (c *Circuit) Tick() {
	// wait for end of tock
	for c.Get(cstClk) == false {
		c.Step()
	}
	for c.Get(cstClk) == true {
		c.Step()
	}
}

// Tock runs the simulation until the end of a false clock signal.
// At this point, the output of clocked components should have stabilized.
//
func (c *Circuit) Tock() {
	// wait for end of tick
	for c.Get(cstClk) == true {
		c.Step()
	}
	for c.Get(cstClk) == false {
		c.Step()
	}
}

// TickTock runs the simulation for a whole clock cycle.
//
func (c *Circuit) TickTock() {
	// wait for end of tick
	for c.Get(cstClk) == false {
		c.Step()
	}
	for c.Get(cstClk) == true {
		c.Step()
	}
	for c.Get(cstClk) == false {
		c.Step()
	}
}
