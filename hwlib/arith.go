// Copyright 2018 Denis Bernard <db047h@gmail.com>
// Licensed under the MIT license. See license text in the LICENSE file.

package hwlib

import (
	"strconv"

	"github.com/db47h/hwsim"
)

var hAdder = &hwsim.PartSpec{
	Name:    "HalfAdder",
	Inputs:  []string{pA, pB},
	Outputs: []string{"s", "c"},
	Mount: func(s *hwsim.Socket) []hwsim.Component {
		a, b := s.Pin(pA), s.Pin(pB)
		sum, cout := s.Pin("s"), s.Pin("c")
		return []hwsim.Component{
			func(c *hwsim.Circuit) {
				va, vb := c.Get(a), c.Get(b)
				c.Set(sum, va && !vb || !va && vb)
				c.Set(cout, va && vb)
			}}
	}}

// HalfAdder returns a half adder.
//
//	Inputs: a, b
//	Outputs: s, c
//	Function: s = lsb(a + b)
//	          c = msb(a + b)
//
func HalfAdder(c string) hwsim.Part {
	return hAdder.NewPart(c)
}

var adder = &hwsim.PartSpec{
	Name:    "FullAdder",
	Inputs:  []string{pA, pB, "cin"},
	Outputs: []string{"s", "cout"},
	Mount: func(s *hwsim.Socket) []hwsim.Component {
		a, b, cin := s.Pin(pA), s.Pin(pB), s.Pin("cin")
		sum, cout := s.Pin("s"), s.Pin("cout")
		return []hwsim.Component{
			func(c *hwsim.Circuit) {
				va, vb, cin := c.Get(a), c.Get(b), c.Get(cin)
				s := va && !vb || !va && vb
				c.Set(sum, s && !cin || !s && cin)
				c.Set(cout, s && cin || va && vb)
			}}
	}}

// FullAdder returns a 3 bit adder.
//
//	Inputs: a, b, cin
//	Outputs: s, c
//	Function: s = lsb(a + b + cin)
//	          c = msb(a + b)
//
func FullAdder(c string) hwsim.Part {
	return adder.NewPart(c)
}

// AdderN returns a N-bits adder
//
//	Inputs: a[bits], b[bits]
//	Outputs: out[bits], c
//
func AdderN(bits int) hwsim.NewPartFn {
	adderN := &hwsim.PartSpec{
		Name:    "Adder" + strconv.Itoa(bits),
		Inputs:  bus(bits, pA, pB),
		Outputs: append(bus(bits, pOut), "c"),
		Mount: func(s *hwsim.Socket) []hwsim.Component {
			a, b := s.Bus(pA, bits), s.Bus(pB, bits)
			out, cout := s.Bus(pOut, bits), s.Pin("c")
			return []hwsim.Component{
				func(c *hwsim.Circuit) {
					cc := false
					for i, o := range out {
						va, vb := c.Get(a[i]), c.Get(b[i])
						s0 := va && !vb || !va && vb
						s := !s0 && cc || s0 && !cc
						cc = va && vb || s0 && cc
						c.Set(o, s)
					}
					c.Set(cout, cc)
				}}
		}}
	return adderN.NewPart
}
