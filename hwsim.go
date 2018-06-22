// Copyright 2018 Denis Bernard <db047h@gmail.com>
// Licensed under the MIT license. See license text in the LICENSE file.

package hwsim

import (
	"github.com/pkg/errors"
)

// Updater is the interface for components in a circuit.
//
// Clocked components must also implement Ticker.
//
type Updater interface {
	// Update is called every time an Updater's output pins must be updated.
	// The clk value is the current state of the clock signal (true during a
	// tick, false during a tock). Non-clocked components should ignore this
	// signal and just pass it along in the Recv and Send calls to their
	// connected wires.
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

// A MountFn mounts a part into socket s. MountFn's should query the socket
// to get Wires connected to a part's pins and return closures around these
// Wires.
//
// For example, a Not gate can be defined like this:
//
//	notSpec := &hwsim.PartSpec{
//		Name: "Not",
//		In: hwsim.IO("in"),
//		Out: hwsim.IO("out"),
//		Mount: func (s *hwsim.Socket) hwsim.Updater {
//			in, out := s.Wire("in"), s.Wire("out")
//			return hwsim.UpdaterFn(
//				func (clk bool) { out.Send(clk, !in.Recv(clk)) }
//			)
//		}}
//
type MountFn func(s *Socket) Updater

// A PartSpec represents a part specification (its blueprint).
//
// Custom parts are implemented by creating a PartSpec:
//
//	notSpec := &hwsim.PartSpec{
//		Name: "Not",
//		In: hwsim.IO("in"),
//		Out: hwsim.IO("out"),
//		Mount: func (s *hwsim.Socket) hwsim.Updater {
//			in, out := s.Wire("in"), s.Wire("out")
//			return hwsim.UpdaterFn(
//				func (clk bool) { out.Send(clk, !in.Recv(clk)) }
//			)
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
//	c, _ := Chip("dummy", In("a, b"), Out("notA, d"),
//		notGate("in: a, out: notA"),
//		// ...
//	)
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
	// In a MountFn, only internal pin names must be used when calling the Socket
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

// A Wrapper is a part that wraps together several other parts and has no
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
	wires   []*Wire
	tickers []Ticker
	size    int // # of updaters
	ticks   uint64
	clk     bool
}

// NewCircuit builds a new circuit simulation based on the given parts.
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
	c.wires = make([]*Wire, cstCount)

	inputFn := func(f func() bool) *Wire {
		p := new(Wire)
		up := UpdaterFn(func(clk bool) { p.Send(clk, f()) })
		p.SetSource(up)
		return p
	}

	c.wires[cstFalse] = inputFn(func() bool { return false })
	c.wires[cstTrue] = inputFn(func() bool { return true })
	c.wires[cstClk] = inputFn(func() bool { return c.clk })

	c.unwrap(wrap("").Mount(newSocket(c)))

	for i := range c.wires {
		if c.wires[i].src == nil {
			panic(errors.Errorf("nil src for wire %p (%d)", c.wires[i], i))
		}
	}

	return c, nil
}

func (c *Circuit) unwrap(u Updater) {
	if uw, ok := u.(Wrapper); ok {
		for _, u := range uw.Unwrap() {
			c.unwrap(u)
		}
	} else {
		c.size++
		if t, ok := u.(Ticker); ok {
			c.tickers = append(c.tickers, t)
		}
	}
}

// alloc allocates a pin.
//
func (c *Circuit) allocPin() *Wire {
	p := new(Wire)
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

// ComponentCount returns the number of components in the circuit.
//
func (c *Circuit) ComponentCount() int {
	return c.size
}

// WireCount returns the number of components in the circuit.
//
func (c *Circuit) WireCount() int {
	return len(c.wires)
}

// Ticker is a marker interface implemented by Updaters that have side effects
// outside of a circuit or that somehow drive the circuit. All sequential
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
