package hwsim

import (
	"runtime"
	"sync"

	"github.com/pkg/errors"
)

// A Component is a component in a circuit that can Get and Set states.
//
type Component func(c *Circuit)

// A MountFn mounts a part into socket s. MountFn's should query
// the socket for assigned pin numbers and return closures around
// these pin numbers.
//
// For example, a Not gate can be defined like this:
//
//	not := &PartSpec{
//		Name: "Not",
//		In: In("in"),
//		Out: Out("out"),
//		Mount: func (s *Socket) []Component {
//			in, out := s.Pin("in"), s.Pin("out")
//			return []Component{
//				func (c *Circuit) { c.Set(out, !c.Get(in)) }
//			}
//		}}
//
type MountFn func(s *Socket) []Component

// A PartSpec wraps a part specification (its blueprint).
//
// Custom parts are implemented by creating a PartSpec:
//
//	notSpec := &PartSpec{
//		Name: "Not",
//		In: In("in"),
//		Out: Out("out"),
//		Mount: func (s *Socket) []Component {
//			in, out := s.Pin("in"), s.Pin("out")
//			return []Component{
//				func (c *Circuit) { c.Set(out, !c.Get(in)) }
//			}
//		}}
//
// Then get a NewPartFn for that PartSpec:
//
//	var notGate = notSpec.NewPart
//
// or:
//
//	func Not(c string) Part { return notSpec.NewPart(c) }
//
// Which can the be used as a NewPartFn when building other chips:
//
//	c, _ := Chip("dummy", In("a, b"), Out("c, d"), Parts{
//		notGate("in: a, out: c"),
//		Not("in: b, out: d"),
//	})
//
type PartSpec struct {
	Name    string // Part name
	Inputs         // Input pins
	Outputs        // Output pins

	// Pinout maps the input and output pin names (public interface) of a part
	// to internal (private) names. If nil, the In and Out values will be used
	// and mapped one to one.
	// In a MountFn, only private pin names must be used when calling Socket
	// methods.
	Pinout map[string]string

	// Mount function (see MountFn).
	Mount MountFn
}

// NewPart is a NewPartFn that wraps p with the given connections into a Part.
//
func (p *PartSpec) NewPart(connections string) Part {
	ex, err := ParseConnections(connections)
	if err != nil {
		panic(err)
	}
	if p.Pinout == nil {
		p.Pinout = make(map[string]string)
		for _, i := range p.Inputs {
			p.Pinout[i] = i
		}
		for _, o := range p.Outputs {
			p.Pinout[o] = o
		}
	}
	return Part{p, ex}
}

// Inputs is a slice of input pin names.
//
type Inputs []string

// Outputs is a slice of output pin names.
//
type Outputs []string

// In parses an input pin specification string and returns individual input pin names.
//
func In(inputs string) Inputs {
	r, err := parseIOspec(inputs)
	if err != nil {
		panic(errors.Wrap(err, "failed to parse inputs"))
	}
	return r
}

// Out parses an output pin specification string and returns individual output pin names.
//
func Out(outputs string) Outputs {
	r, err := parseIOspec(outputs)
	if err != nil {
		panic(errors.Wrap(err, "failed to parse outputs"))
	}
	return r
}

// A NewPartFn is a function that takes a connection configuration and returns a
// new Part. See ParseConnections for the syntax of the connection configuration
// string.
//
type NewPartFn func(c string) Part

// Parts is a convenience wrapper for []Part.
//
type Parts []Part

// A Part wraps a part specification together with its connections within a host
// chip.
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
// workers is the number of goroutines used to update the state of the Circuit
// each steap of the simulation. If less or equal to 0, the return value of
// runtime.NumCPU() will be used.
//
// stepsPerCycle indicates how many simulation steps to run per clock cycle
// (the Clk signal, not wall clock). The exact value to use depends on the
// complexity of the chips used (a built-in NAND takes one step to update its
// output). While this value could be computed, this feature is not implemented
// yet.
//
func NewCircuit(workers int, stepsPerCycle uint, ps Parts) (*Circuit, error) {
	if len(ps) == 0 {
		return nil, errors.New("empty part list")
	}

	if stepsPerCycle < 2 {
		stepsPerCycle = 2
	}
	stepsPerCycle--
	stepsPerCycle |= stepsPerCycle >> 1
	stepsPerCycle |= stepsPerCycle >> 2
	stepsPerCycle |= stepsPerCycle >> 4
	stepsPerCycle |= stepsPerCycle >> 8
	stepsPerCycle |= stepsPerCycle >> 16
	stepsPerCycle |= stepsPerCycle >> 32
	stepsPerCycle++

	// new circuit with room for constant value pins.
	cc := &Circuit{count: cstCount, tpc: stepsPerCycle}
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

// Get returns the state of pin n. The value of n should be obtained in a
// MountFn by a call to one of the Socket methods.
//
func (c *Circuit) Get(n int) bool {
	return c.s0[n]
}

// Set sets the state s of pin n. The value of n should be obtained in a
// MountFn by a call to one of the Socket methods.
//
func (c *Circuit) Set(n int, s bool) {
	c.s1[n] = s
}

// Toggle toggles the state of pin n. The value of n should be obtained in a
// MountFn by a call to one of the Socket methods.
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

// Step advances the simulation by one step.
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
