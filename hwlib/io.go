// Copyright 2018 Denis Bernard <db047h@gmail.com>
// Licensed under the MIT license. See license text in the LICENSE file.

package hwlib

import (
	"strconv"

	hw "github.com/db47h/hwsim"
)

// Input creates a function based input.
//
//	Outputs: out
//	Function: out = f()
//
func Input(f func() bool) hw.NewPartFn {
	p := &hw.PartSpec{
		Name:    "Input",
		Inputs:  nil,
		Outputs: hw.Outputs{pOut},
		Mount: func(s *hw.Socket) []hw.Component {
			pin := s.Pin(pOut)
			return []hw.Component{
				func(c *hw.Circuit) {
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
func Output(f func(bool)) hw.NewPartFn {
	p := &hw.PartSpec{
		Name:    "Output",
		Inputs:  hw.Inputs{pIn},
		Outputs: nil,
		Mount: func(s *hw.Socket) []hw.Component {
			in := s.Pin(pIn)
			return []hw.Component{
				func(c *hw.Circuit) { f(c.Get(in)) },
			}
		},
	}
	return p.NewPart
}

// InputN creates an input bus of the given bits size.
//
func InputN(bits int, f func() int64) hw.NewPartFn {
	return (&hw.PartSpec{
		Name:    "INPUT" + strconv.Itoa(bits),
		Inputs:  nil,
		Outputs: hw.Out(bus(pOut, bits)),
		Mount: func(s *hw.Socket) []hw.Component {
			pins := s.Bus(pOut, bits)
			return []hw.Component{func(c *hw.Circuit) {
				in := f()
				for bit := 0; bit < len(pins); bit++ {
					c.Set(pins[bit], in&(1<<uint(bit)) != 0)
				}
			}}
		}}).NewPart
}

// OutputN creates an output bus of the given bits size.
//
func OutputN(bits int, f func(int64)) hw.NewPartFn {
	return (&hw.PartSpec{
		Name:    "OUTPUTBUS" + strconv.Itoa(bits),
		Inputs:  hw.In(bus(pIn, bits)),
		Outputs: nil,
		Mount: func(s *hw.Socket) []hw.Component {
			pins := s.Bus(pIn, bits)
			return []hw.Component{func(c *hw.Circuit) {
				var out int64
				for i := 0; i < len(pins); i++ {
					if c.Get(pins[i]) {
						out |= 1 << uint(i)
					}
				}
				f(out)
			}}
		}}).NewPart
}
