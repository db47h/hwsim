// Copyright 2018 Denis Bernard <db047h@gmail.com>
// Licensed under the MIT license. See license text in the LICENSE file.

package hwsim

import "strconv"

// Constant Wire names. These wires can only be connected to the input pins of a part.
//
// Those are reserved names and should not be used as input or output names in
// custom chips.
//
var (
	False       = "false" // always false input
	True        = "true"  // alwyas true input
	Clk         = "clk"   // clock signal. False during Tick, true during Tock.
	cstPinNames = [...]string{"false", "true", "clk"}
)

const (
	cstFalse = iota
	cstTrue
	cstClk
	cstCount
)

// A Connection represents a connection between the pin PP of a part and
// the pins CP in its host chip.
//
type Connection struct {
	PP string
	CP []string
}

// A Wire connects pins together. A Wire may have only one source pin and multiple
// destination pins. i.e. Only one component can send a signal on a Wire.
//
type Wire struct {
	src   Updater
	clk   bool
	recv  bool
	value bool
}

// SetSource sets the given Updater as the wire's source.
//
func (c *Wire) SetSource(u Updater) {
	c.src = u
}

// Send sends a signal a time clk.
//
func (c *Wire) Send(clk bool, value bool) {
	if c.clk != clk {
		c.clk, c.value = clk, value
	}
}

// Recv recieves a signal at time clk.
// It may trigger an update of the source component.
//
func (c *Wire) Recv(clk bool) bool {
	if c.clk != clk {
		if c.recv {
			panic("wiring loop detected")
		} else {
			c.recv = true
			c.src.Update(clk) // should trigger a send
			c.recv = false
		}
	}
	return c.value
}

// A Bus is a set of Wires.
//
type Bus []*Wire

// Recv returns the int64 value of the bus. Wire 0 is the LSB.
//
func (b Bus) Recv(clk bool) int64 {
	var out int64
	for bit, p := range b {
		if p.Recv(clk) {
			out |= 1 << uint(bit)
		}
	}
	return out
}

// Send sets the int64 value of the bus. Pin Wire is the LSB.
//
func (b Bus) Send(clk bool, v int64) {
	for bit, p := range b {
		p.Send(clk, v&(1<<uint(bit)) != 0)
	}
}

// pinName returns the pin name for the n-th bit of the named bus.
//
func pinName(name string, bit int) string {
	return name + "[" + strconv.Itoa(bit) + "]"
}

// A Socket maps a part's internal pin names to Wires in a circuit.
// See PartSpec.Pinout.
//
type Socket struct {
	m map[string]*Wire
	c *Circuit
}

func newSocket(c *Circuit) *Socket {
	return &Socket{
		m: map[string]*Wire{
			False: c.wires[cstFalse],
			True:  c.wires[cstTrue],
			Clk:   c.wires[cstClk],
		},
		c: c,
	}
}

// Wire returns the Wire connected to the given pin name.
//
func (s *Socket) Wire(pinName string) *Wire {
	return s.m[pinName]
}

// wireOrNew returns the Wire connected to the given pin name.
// If no such Wire exists, a new one is assigned.
//
func (s *Socket) wireOrNew(name string) *Wire {
	p, ok := s.m[name]
	if !ok {
		p = s.c.allocPin()
		s.m[name] = p
	}
	return p
}

// Bus returns the Bus connected to the given bus name.
//
func (s *Socket) Bus(name string, size int) Bus {
	out := make([]*Wire, size)
	for i := range out {
		out[i] = s.m[pinName(name, i)]
	}
	return out
}
