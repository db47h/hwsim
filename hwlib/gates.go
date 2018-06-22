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
func bus(bits int, names ...string) []string {
	b := make([]string, len(names)*bits)
	for i, n := range names {
		for j := 0; j < bits; j++ {
			b[i*bits+j] = n + "[" + strconv.Itoa(j) + "]"
		}
	}
	return b
}

var notGate = hwsim.PartSpec{Name: "NOR", Inputs: []string{pIn}, Outputs: []string{pOut},
	Mount: func(s *hwsim.Socket) hwsim.Updater {
		in, out := s.Wire(pIn), s.Wire(pOut)
		return hwsim.UpdaterFn(func(clk bool) { out.Send(clk, !in.Recv(clk)) })
	}}

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

func (g gate) mount(s *hwsim.Socket) hwsim.Updater {
	a, b, out := s.Wire(pA), s.Wire(pB), s.Wire(pOut)
	return hwsim.UpdaterFn(func(clk bool) { out.Send(clk, g(a.Recv(clk), b.Recv(clk))) })
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
	gateIn  = []string{pA, pB}
	gateOut = []string{pOut}

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

func notN(bits int) *hwsim.PartSpec {
	return &hwsim.PartSpec{
		Name:    "NOT" + strconv.Itoa(bits),
		Inputs:  bus(bits, pIn),
		Outputs: bus(bits, pOut),
		Mount: func(s *hwsim.Socket) hwsim.Updater {
			ins := s.Bus(pIn, bits)
			outs := s.Bus(pOut, bits)
			return hwsim.UpdaterFn(
				func(clk bool) {
					for i, pin := range ins {
						outs[i].Send(clk, !pin.Recv(clk))
					}
				})
		}}
}

// NotN returns a N-bits NOT gate.
//
//	Inputs: in[bits]
//	Outputs: out[bits]
//	Function: for i := range out { out[i] = !in[i] }
//
func NotN(bits int) hwsim.NewPartFn {
	return notN(16).NewPart
}

var (
	not16 = notN(16)
)

type gateN struct {
	bits int
	fn   func(bool, bool) bool
}

func (g *gateN) mount(s *hwsim.Socket) hwsim.Updater {
	a, b, out := s.Bus(pA, g.bits), s.Bus(pB, g.bits), s.Bus(pOut, g.bits)
	return hwsim.UpdaterFn(
		func(clk bool) {
			for i, o := range out {
				o.Send(clk, g.fn(a[i].Recv(clk), b[i].Recv(clk)))
			}
		})
}

func newGateN(name string, bits int, f func(bool, bool) bool) *hwsim.PartSpec {
	return &hwsim.PartSpec{
		Name:    name + strconv.Itoa(bits),
		Inputs:  bus(bits, pA, pB),
		Outputs: bus(bits, pOut),
		Mount:   (&gateN{bits, f}).mount,
	}
}

// GateN returns a N-bits logic gate.
//
//	Inputs: a[bits], b[bits]
//	Outouts: out[bits]
//	Function: for i := range out { out[i] = f(a[i], b[i]) }
//
func GateN(name string, bits int, f func(bool, bool) bool) hwsim.NewPartFn {
	return newGateN(name, bits, f).NewPart
}

// AndN returns a N-bits AND gate.
//
//	Inputs: a[N], b[N]
//	Outputs: out[N]
//	Function: for i := range out { out[i] = a[i] && b[i] }
//
func AndN(bits int) hwsim.NewPartFn {
	return newGateN("AND", bits, func(a, b bool) bool { return a && b }).NewPart
}

// NandN returns a N-bits NAND gate.
//
//	Inputs: a[N], b[N]
//	Outputs: out[N]
//	Function: for i := range out { out[i] = !(a[i] && b[i]) }
//
func NandN(bits int) hwsim.NewPartFn {
	return newGateN("NAND", bits, func(a, b bool) bool { return !(a && b) }).NewPart
}

// OrN returns a N-bits OR gate.
//
//	Inputs: a[N], b[N]
//	Outputs: out[N]
//	Function: for i := range out { out[i] = (a[i] || b[i]) }
//
func OrN(bits int) hwsim.NewPartFn {
	return newGateN("OR", bits, func(a, b bool) bool { return a || b }).NewPart
}

// NorN returns a N-bits NOR gate.
//
//	Inputs: a[N], b[N]
//	Outputs: out[N]
//	Function: for i := range out { out[i] = !(a[i] || b[i]) }
//
func NorN(bits int) hwsim.NewPartFn {
	return newGateN("NOR", bits, func(a, b bool) bool { return !(a || b) }).NewPart
}

// OrNWay returns a N-Way OR gate.
//
//	Inputs: in[n]
//	Outputs: out
//	Function: out = in[0] || in[1] || in[2] || ... || in[n-1]
//
func OrNWay(ways int) hwsim.NewPartFn {
	return (&hwsim.PartSpec{
		Name:    "OR" + strconv.Itoa(ways) + "Way",
		Inputs:  bus(ways, pIn),
		Outputs: hwsim.IO(pOut),
		Mount: func(s *hwsim.Socket) hwsim.Updater {
			in := s.Bus(pIn, ways)
			out := s.Wire(pOut)
			return hwsim.UpdaterFn(
				func(clk bool) {
					for _, i := range in {
						if i.Recv(clk) {
							out.Send(clk, true)
							return
						}
					}
					out.Send(clk, false)
				})
		}}).NewPart
}

// AndNWay returns a N-Way OR gate.
//
//	Inputs: in[n]
//	Outputs: out
//	Function: out = in[0] && in[1] && in[2] || ... && in[n-1]
//
func AndNWay(ways int) hwsim.NewPartFn {
	return (&hwsim.PartSpec{
		Name:    "AND" + strconv.Itoa(ways) + "Way",
		Inputs:  bus(ways, pIn),
		Outputs: hwsim.IO(pOut),
		Mount: func(s *hwsim.Socket) hwsim.Updater {
			in := s.Bus(pIn, ways)
			out := s.Wire(pOut)
			return hwsim.UpdaterFn(
				func(clk bool) {
					for _, i := range in {
						if i.Recv(clk) == false {
							out.Send(clk, false)
							return
						}
					}
					out.Send(clk, true)
				})
		}}).NewPart
}
