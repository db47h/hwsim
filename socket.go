// Copyright 2018 Denis Bernard <db047h@gmail.com>
// Licensed under the MIT license. See license text in the LICENSE file.

package hwsim

import "strconv"

// busPinName returns the pin name for the n-th bit of the named bus.
//
func busPinName(name string, bit int) string {
	return name + "[" + strconv.Itoa(bit) + "]"
}

// A Socket maps a part's internal pin names to pin numbers in a circuit.
// See PartSpec.Pinout.
//
type Socket struct {
	m map[string]*Pin
	c *Circuit
}

func newSocket(c *Circuit) *Socket {
	return &Socket{
		m: map[string]*Pin{
			False: c.wires[cstFalse],
			True:  c.wires[cstTrue],
			Clk:   c.wires[cstClk],
		},
		c: c,
	}
}

// Pin returns the pin number assigned to the given pin name.
//
func (s *Socket) Pin(name string) *Pin {
	return s.m[name]
}

// pinOrNew returns the pin number assigned to the given pin name.
// If no such pin exists a new one is assigned.
//
func (s *Socket) pinOrNew(name string) *Pin {
	p, ok := s.m[name]
	if !ok {
		p = s.c.allocPin()
		s.m[name] = p
	}
	return p
}

type Bus []*Pin

func (b Bus) GetInt64(clk bool) int64 {
	var out int64
	for bit, p := range b {
		if p.Recv(clk) {
			out |= 1 << uint(bit)
		}
	}
	return out
}

func (b Bus) SetInt64(clk bool, v int64) {
	for bit, p := range b {
		p.Send(clk, v&(1<<uint(bit)) != 0)
	}
}

func (b Bus) Connect(u Updater) {
	for _, p := range b {
		p.Connect(u)
	}
}

// Bus returns the pin numbers assigned to the given bus name.
//
func (s *Socket) Bus(name string, size int) Bus {
	out := make([]*Pin, size)
	for i := range out {
		out[i] = s.m[busPinName(name, i)]
	}
	return out
}
