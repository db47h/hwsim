// Copyright 2018 Denis Bernard <db047h@gmail.com>
// Licensed under the MIT license. See license text in the LICENSE file.

package hwsim

import (
	"github.com/pkg/errors"
)

// Updater is the interface that custom components built using reflection must implement.
// See MakePart.
//
type Updater interface {
	Update(clk bool)
}

// A UpdaterFn is a single update function that implements Updater.
//
type UpdaterFn func(clk bool)

// Update implements Updater.
//
func (u UpdaterFn) Update(clk bool) {
	u(clk)
}

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
type MountFn func(s *Socket) Updater

// A PartSpec wraps a part specification (its blueprint).
//
// Custom parts are implemented by creating a PartSpec:
//
//	notSpec := &hwsim.PartSpec{
//		Name: "Not",
//		In: hwsim.In("in"),
//		Out: hwsim.Out("out"),
//		Mount: func (s *hwsim.Socket) []hwsim.Component {
//			in, out := s.Pin("in"), s.Pin("out")
//			return []hwsim.Component{
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
	// Part name.
	Name string
	// Input pin names. Must be distinct pin names.
	// Use the IO() function to expand an input description like
	// "a, b, bus[2]" to []string{"a", "b", "bus[0]", "bus[1]"}
	// See IO() for more details.
	Inputs []string
	// Output pin name. Must be distinct pin names.
	// Use the IO() function to expand an output description string.
	Outputs []string
	// Pinout maps the input and output pin names (public interface) of a part
	// to internal (private) names. If nil, the In and Out values will be used
	// and mapped one to one.
	// In a MountFn, only private pin names must be used when calling the Socket
	// methods.
	// Most custom part implementations should ignore this field and set it to
	// nil.
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

// A NewPartFn is a function that takes a connection configuration and returns a
// new Part. See ParseConnections for the syntax of the connection configuration
// string.
//
type NewPartFn func(c string) Part

// A Part wraps a part specification together with its connections within a host
// chip.
//
type Part struct {
	*PartSpec
	Conns []Connection
}

// A Wrapper is a parts that wraps together several other parts and has no
// additional function. When a Circtuit is built, Wrappers are unwrapped
// and discarded ─ their Update function is never called.
//
type Wrapper interface {
	Updater
	Unwrap() []Updater
}

// Circuit is a runnable circuit simulation.
//
type Circuit struct {
	wires   []*Pin
	tickers []Ticker
	ticks   uint64
	clk     bool
}

// NewCircuit builds a new circuit based on the given parts.
//
// workers is the number of goroutines used to update the state of the Circuit
// each step of the simulation. If less or equal to 0, the value of GOMAXPROCS
// will be used.
//
// stepsPerCycle indicates how many simulation steps to run per clock cycle
// (the Clk signal, not wall clock). The exact value to use depends on the
// complexity of the chips used (a built-in NAND takes one step to update its
// output). While this value could be computed, this feature is not implemented
// yet.
//
// Callers must make sure to call Dispose() once the circuit is no longer needed
// in order to release allocated resources.
//
func NewCircuit(parts ...Part) (*Circuit, error) {
	if len(parts) == 0 {
		return nil, errors.New("empty part list")
	}

	wrap, err := Chip("CIRCUIT", "", "", parts...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create chip wrapper")
	}

	c := new(Circuit)
	c.wires = make([]*Pin, cstCount)

	inputFn := func(f func() bool) *Pin {
		p := new(Pin)
		up := UpdaterFn(func(clk bool) { p.Send(clk, f()) })
		p.SetSource(up)
		return p
	}

	c.wires[cstFalse] = inputFn(func() bool { return false })
	c.wires[cstTrue] = inputFn(func() bool { return true })
	c.wires[cstClk] = inputFn(func() bool { return c.clk })

	unwrap(&c.tickers, wrap("").Mount(newSocket(c)))

	for i := range c.wires {
		if c.wires[i].src == nil {
			panic(errors.Errorf("nil src for wire %p (%d)", c.wires[i], i))
		}
	}

	return c, nil
}

func unwrap(ul *[]Ticker, u Updater) {
	if uw, ok := u.(Wrapper); ok {
		for _, u := range uw.Unwrap() {
			unwrap(ul, u)
		}
	} else if t, ok := u.(Ticker); ok {
		*ul = append(*ul, t)
	}
}

// Dispose releases all resources allocated for a circuit and stops
// worker goroutines.
//
func (c *Circuit) Dispose() {
}

// alloc allocates a pin.
//
func (c *Circuit) allocPin() *Pin {
	p := new(Pin)
	c.wires = append(c.wires, p)
	return p
}

// Ticks returns the value of the step counter.
//
func (c *Circuit) Ticks() uint64 {
	return c.ticks
}

// Tick runs the simulation until the beginning of the next half clock cycle.
//
func (c *Circuit) Tick() {
	if !c.clk {
		c.clk = true
		c.update()
	}
}

// Tock runs the simulation until the beginning of the next clock cycle.
// Once Tock returns, the output of clocked components should have stabilized.
//
func (c *Circuit) Tock() {
	if c.clk {
		c.clk = false
		c.update()
	}
}

func (c *Circuit) update() {
	c.ticks++
	for _, u := range c.tickers {
		u.Update(c.clk)
	}
	for _, w := range c.wires {
		w.clk = c.clk
	}
}

// TickTock runs the simulation for a whole clock cycle.
//
func (c *Circuit) TickTock() {
	c.Tick()
	c.Tock()
}

// Ticker is a marker interface implemented by updaters that have side effects
// outside of a circuit or that somehow drives the circuit. All sequential
// components must implement Ticker.
//
type Ticker interface {
	Updater
	Tick()
}

// A TickerFn is a single update function that implements Ticker.
//
type TickerFn func(clk bool)

// Update implements Updater.
//
func (f TickerFn) Update(clk bool) { f(clk) }

// Tick implements Ticker.
//
func (f TickerFn) Tick() {}
