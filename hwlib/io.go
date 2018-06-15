// Copyright 2018 Denis Bernard <db047h@gmail.com>
// Licensed under the MIT license. See license text in the LICENSE file.

package hwlib

import (
	"strconv"

	"github.com/db47h/hwsim"
)

// Int64 returns the pins as an int64. Pin 0 is lsb.
//
func Int64(c *hwsim.Circuit, pins []int) int64 {
	var out int64
	for bit := range pins {
		if c.Get(pins[bit]) {
			out |= 1 << uint(bit)
		}
	}
	return out
}

// SetInt64 sets the pins to the given int64 value.
//
func SetInt64(c *hwsim.Circuit, pins []int, v int64) {
	for bit := range pins {
		c.Set(pins[bit], v&(1<<uint(bit)) != 0)
	}
}

// Input creates a function based input.
//
//	Outputs: out
//	Function: out = f()
//
func Input(f func() bool) hwsim.NewPartFn {
	p := &hwsim.PartSpec{
		Name:    "Input",
		Inputs:  nil,
		Outputs: []string{pOut},
		Mount: func(s *hwsim.Socket) []hwsim.Component {
			pin := s.Pin(pOut)
			return []hwsim.Component{
				func(c *hwsim.Circuit) {
					c.Set(pin, f())
				},
			}
		},
	}
	return p.NewPart
}

// Output creates an output or probe. The fn function is
// called with the named pin state on every circuit update.
//
//	Inputs: in
//	Function: f(in)
//
func Output(f func(bool)) hwsim.NewPartFn {
	p := &hwsim.PartSpec{
		Name:    "Output",
		Inputs:  []string{pIn},
		Outputs: nil,
		Mount: func(s *hwsim.Socket) []hwsim.Component {
			in := s.Pin(pIn)
			return []hwsim.Component{
				func(c *hwsim.Circuit) { f(c.Get(in)) },
			}
		},
	}
	return p.NewPart
}

// InputN creates an input bus of the given bits size.
//
func InputN(bits int, f func() int64) hwsim.NewPartFn {
	return (&hwsim.PartSpec{
		Name:    "INPUT" + strconv.Itoa(bits),
		Inputs:  nil,
		Outputs: bus(bits, pOut),
		Mount: func(s *hwsim.Socket) []hwsim.Component {
			pins := s.Bus(pOut, bits)
			return []hwsim.Component{func(c *hwsim.Circuit) {
				in := f()
				for bit := 0; bit < len(pins); bit++ {
					c.Set(pins[bit], in&(1<<uint(bit)) != 0)
				}
			}}
		}}).NewPart
}

// OutputN creates an output bus of the given bits size.
//
func OutputN(bits int, f func(int64)) hwsim.NewPartFn {
	return (&hwsim.PartSpec{
		Name:    "OUTPUTBUS" + strconv.Itoa(bits),
		Inputs:  bus(bits, pIn),
		Outputs: nil,
		Mount: func(s *hwsim.Socket) []hwsim.Component {
			pins := s.Bus(pIn, bits)
			return []hwsim.Component{func(c *hwsim.Circuit) {
				f(Int64(c, pins))
			}}
		}}).NewPart
}
