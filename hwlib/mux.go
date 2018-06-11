// Copyright 2018 Denis Bernard <db047h@gmail.com>
// Licensed under the MIT license. See license text in the LICENSE file.

package hwlib

import (
	"strconv"

	"github.com/db47h/hwsim"
)

// Mux returns a multiplexer.
//
//	Inputs: a, b, sel
//	Outputs: out
//	Function: if sel == 0 { out = a } else { out = b }
//
func Mux(w string) hwsim.Part { return mux.NewPart(w) }

var mux = hwsim.PartSpec{
	Name:    "MUX",
	Inputs:  hwsim.Inputs{pA, pB, pSel},
	Outputs: hwsim.Outputs{pOut},
	Mount: func(s *hwsim.Socket) []hwsim.Component {
		a, b, sel, out := s.Pin(pA), s.Pin(pB), s.Pin(pSel), s.Pin(pOut)
		return []hwsim.Component{func(c *hwsim.Circuit) {
			if c.Get(sel) {
				c.Set(out, c.Get(b))
			} else {
				c.Set(out, c.Get(a))
			}
		}}
	},
}

// DMux returns a demultiplexer.
//
//	Inputs: in, sel
//	Outputs: a, b
//	Function: if sel == 0 { a = in; b = 0 } else { a = 0; b = in }
//
func DMux(w string) hwsim.Part { return dmux.NewPart(w) }

var dmux = hwsim.PartSpec{
	Name:    "DMUX",
	Inputs:  hwsim.Inputs{pIn, pSel},
	Outputs: hwsim.Outputs{pA, pB},
	Mount: func(s *hwsim.Socket) []hwsim.Component {
		in, sel, a, b := s.Pin(pIn), s.Pin(pSel), s.Pin(pA), s.Pin(pB)
		return []hwsim.Component{func(c *hwsim.Circuit) {
			if c.Get(sel) {
				c.Set(a, false)
				c.Set(b, c.Get(in))
			} else {
				c.Set(a, c.Get(in))
				c.Set(b, false)
			}
		}}
	},
}

// SpecMuxN returns a PartSpec for an n-bits Mux
//
//	Inputs: a[bits], b[bits], sel
//	Outputs: out[bits]
//	Function: for i := range out { if sel == 0 { out[i] = a[i] } else { out[i] = b[i] } }
//
func SpecMuxN(bits int) *hwsim.PartSpec {
	return &hwsim.PartSpec{
		Name:    "Mux" + strconv.Itoa(bits),
		Inputs:  append(bus(bits, pA, pB), pSel),
		Outputs: bus(bits, pOut),
		Mount: func(s *hwsim.Socket) []hwsim.Component {
			a, b, sel := s.Bus(pA, bits), s.Bus(pB, bits), s.Pin(pSel)
			o := s.Bus(pOut, bits)
			return []hwsim.Component{
				func(c *hwsim.Circuit) {
					if c.Get(sel) {
						for i := range o {
							c.Set(o[i], c.Get(b[i]))
						}
					} else {
						for i := range o {
							c.Set(o[i], c.Get(a[i]))
						}
					}
				}}
		}}

}

var (
	mux16 = SpecMuxN(16)
)

// Mux16 returns a 16-bits Mux
//
//	Inputs: a[16], b[16], sel
//	Outputs: out[16]
//	Function: for i := range out { if sel == 0 { out[i] = a[i] } else { out[i] = b[i] } }
//
func Mux16(c string) hwsim.Part {
	return mux16.NewPart(c)
}
