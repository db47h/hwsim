// Copyright 2018 Denis Bernard <db047h@gmail.com>
// Licensed under the MIT license. See license text in the LICENSE file.

// Package hwlib provides a library of reusable parts for hwsim.
//
// Copyright 2018 Denis Bernard <db047h@gmail.com>
//
// This package is licensed under the MIT license. See license text in the LICENSE file.
//
package hwlib

import (
	"strconv"

	"github.com/db47h/hwsim"
)

// common pin names
const (
	pA   = "a"
	pB   = "b"
	pIn  = "in"
	pSel = "sel"
	pOut = "out"
)

// make a bus name
func bus(name string, bits int) string {
	return name + "[" + strconv.Itoa(bits) + "]"
}

var notGate = hwsim.PartSpec{Name: "NOR", Inputs: hwsim.Inputs{pIn}, Outputs: hwsim.Outputs{pOut},
	Mount: func(s *hwsim.Socket) []hwsim.Component {
		in, out := s.Pin(pIn), s.Pin(pOut)
		return []hwsim.Component{
			func(c *hwsim.Circuit) { c.Set(out, !c.Get(in)) },
		}
	},
}

// Not returns a NOT gate.
//
//	Inputs: in
//	Outputs: out
//	Function: out = !in
//
func Not(w string) hwsim.Part {
	return notGate.NewPart(w)
}

// other gates
type gate func(a, b bool) bool

func (g gate) mount(s *hwsim.Socket) []hwsim.Component {
	a, b, out := s.Pin(pA), s.Pin(pB), s.Pin(pOut)
	return []hwsim.Component{
		func(c *hwsim.Circuit) { c.Set(out, g(c.Get(a), c.Get(b))) },
	}
}

func newGate(name string, fn func(a, b bool) bool) *hwsim.PartSpec {
	return &hwsim.PartSpec{
		Name:    name,
		Inputs:  gateIn,
		Outputs: gateOut,
		Mount:   gate(fn).mount,
	}
}

var (
	gateIn  = hwsim.Inputs{pA, pB}
	gateOut = hwsim.Outputs{pOut}

	and  = newGate("AND", func(a, b bool) bool { return a && b })
	nand = newGate("NAND", func(a, b bool) bool { return !(a && b) })
	or   = newGate("OR", func(a, b bool) bool { return a || b })
	nor  = newGate("NOR", func(a, b bool) bool { return !(a || b) })
	xor  = newGate("XOR", func(a, b bool) bool { return a && !b || !a && b })
	xnor = newGate("XNOR", func(a, b bool) bool { return a && b || !a && !b })
)

// And returns a AND gate.
//
//	Inputs: a, b
//	Outputs: out
//	Function: out = a && b
//
func And(w string) hwsim.Part { return and.NewPart(w) }

// Nand returns a NAND gate.
//
//	Inputs: a, b
//	Outputs: out
//	Function: out = !(a && b)
//
func Nand(w string) hwsim.Part { return nand.NewPart(w) }

// Or returns a OR gate.
//
//	Inputs: a, b
//	Outputs: out
//	Function: out = a || b
//
func Or(w string) hwsim.Part { return or.NewPart(w) }

// Nor returns a NOR gate.
//
//	Inputs: a, b
//	Outputs: out
//	Function: out = !(a || b)
//
func Nor(w string) hwsim.Part { return nor.NewPart(w) }

// Xor returns a XOR gate.
//
//	Inputs: a, b
//	Outputs: out
//	Function: out = (a && !b) || (!a && b)
//
func Xor(w string) hwsim.Part { return xor.NewPart(w) }

// Xnor returns a XNOR gate.
//
//	Inputs: a, b
//	Outputs: out
//	Function: out = a && b || !a && !b
//
func Xnor(w string) hwsim.Part { return xnor.NewPart(w) }

// SpecNotN returns a PartSpec for a N-bits NOT gate.
//
//	Inputs: in[bits]
//	Outputs: out[bits]
//	Function: for i := range out { out[i] = !in[i] }
//
func SpecNotN(bits int) *hwsim.PartSpec {
	return &hwsim.PartSpec{
		Name:    "NOT" + strconv.Itoa(bits),
		Inputs:  hwsim.In(bus(pIn, bits)),
		Outputs: hwsim.Out(bus(pOut, bits)),
		Mount: func(s *hwsim.Socket) []hwsim.Component {
			ins := s.Bus(pIn, bits)
			outs := s.Bus(pOut, bits)
			return []hwsim.Component{func(c *hwsim.Circuit) {
				for i, pin := range ins {
					c.Set(outs[i], !c.Get(pin))
				}
			}}
		},
	}
}

var (
	not16 = SpecNotN(16)
)

// Not16 returns a 16 bits NOT gate.
//
//	Inputs: in[16]
//	Outputs: out[16]
//	Function: for i := range out { out[i] = !in[i] }
//
func Not16(w string) hwsim.Part { return not16.NewPart(w) }

type gateN struct {
	bits int
	fn   func(bool, bool) bool
}

func (g *gateN) mount(s *hwsim.Socket) []hwsim.Component {
	a, b, out := s.Bus(pA, g.bits), s.Bus(pB, g.bits), s.Bus(pOut, g.bits)
	return []hwsim.Component{
		func(c *hwsim.Circuit) {
			for i := range a {
				c.Set(out[i], g.fn(c.Get(a[i]), c.Get(b[i])))
			}
		},
	}
}

// SpecGateN returns a PartSpec for an N-bits logic gate.
//
//	Inputs: a[bits], b[bits]
//	Outouts: out[bits]
//	Function: for i := range out { out[i] = f(a[i], b[i]) }
//
func SpecGateN(name string, bits int, f func(bool, bool) bool) *hwsim.PartSpec {
	return &hwsim.PartSpec{
		Name:    name + strconv.Itoa(bits),
		Inputs:  hwsim.In(bus(pA, 16) + ", " + bus(pB, 16)),
		Outputs: hwsim.Out(bus(pOut, bits)),
		Mount:   (&gateN{bits, f}).mount,
	}
}

var (
	and16  = SpecGateN("AND", 16, func(a, b bool) bool { return a && b })
	nand16 = SpecGateN("NAND", 16, func(a, b bool) bool { return !(a && b) })
	or16   = SpecGateN("OR", 16, func(a, b bool) bool { return a || b })
	nor16  = SpecGateN("NOR", 16, func(a, b bool) bool { return !(a || b) })
)

// And16 returns a 16 bits AND gate.
//
//	Inputs: a[16], b[16]
//	Outputs: out[16]
//	Function: for i := range out { out[i] = a[i] && b[i] }
//
func And16(w string) hwsim.Part { return and16.NewPart(w) }

// Nand16 returns a 16 bits NAND gate.
//
//	Inputs: a[16], b[16]
//	Outputs: out[16]
//	Function: for i := range out { out[i] = !(a[i] && b[i]) }
//
func Nand16(w string) hwsim.Part { return nand16.NewPart(w) }

// Or16 returns a 16 bits OR gate.
//
//	Inputs: a[16], b[16]
//	Outputs: out[16]
//	Function: for i := range out { out[i] = (a[i] || b[i]) }
//
func Or16(w string) hwsim.Part { return or16.NewPart(w) }

// Nor16 returns a 16 bits NOR gate.
//
//	Inputs: a[16], b[16]
//	Outputs: out[16]
//	Function: for i := range out { out[i] = !(a[i] || b[i]) }
//
func Nor16(w string) hwsim.Part { return nor16.NewPart(w) }

// DFF returns a clocked data flip flop.
//
//	Inputs: in
//	Outputs: out
//	Function: out(t) = in(t-1) // where t is the current clock cycle.
//
func DFF(w string) hwsim.Part {
	return (&hwsim.PartSpec{
		Name:    "DFF",
		Inputs:  hwsim.Inputs{pIn},
		Outputs: hwsim.Outputs{pOut},
		Mount: func(s *hwsim.Socket) []hwsim.Component {
			in, out := s.Pin(pIn), s.Pin(pOut)
			var curOut bool
			return []hwsim.Component{
				func(c *hwsim.Circuit) {
					// raising edge?
					if c.AtTick() {
						curOut = c.Get(in)
					}
					c.Set(out, curOut)
				}}
		}}).NewPart(w)
}
