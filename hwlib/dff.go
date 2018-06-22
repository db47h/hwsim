// Copyright 2018 Denis Bernard <db047h@gmail.com>
// Licensed under the MIT license. See license text in the LICENSE file.

package hwlib

import (
	"strconv"

	"github.com/db47h/hwsim"
)

// DFF returns a clocked data flip flop.
//
// Works like a gated D latch where E is the inverted clock signal and D the input.
//
//	Inputs: in
//	Outputs: out
//	Function: out(t) = in(t-1) // where t is the current clock cycle.
//
func DFF(c string) hwsim.Part {
	return dff.NewPart(c)
}

var dff = &hwsim.PartSpec{
	Name:    "DFF",
	Inputs:  []string{pIn},
	Outputs: []string{pOut},
	Mount: func(s *hwsim.Socket) hwsim.Updater {
		return &dffImpl{in: s.Wire(pIn), out: s.Wire(pOut)}
	}}

type dffImpl struct {
	in, out *hwsim.Wire
	v       bool
}

func (d *dffImpl) Update(clk bool) {
	// send first in order to prevent recursion
	d.out.Send(clk, d.v)
	// force input update
	v := d.in.Recv(clk)
	// change value only at ticks
	if !clk {
		d.v = v
	}
}

func (*dffImpl) Tick() {}

// DFFN creates a N bits DFF.
//
func DFFN(bits int) hwsim.NewPartFn {
	bs := strconv.Itoa(bits)
	return (&hwsim.PartSpec{
		Name:    "DFF" + bs,
		Inputs:  bus(bits, pIn),
		Outputs: bus(bits, pOut),
		Mount: func(s *hwsim.Socket) hwsim.Updater {
			in, out := s.Bus(pIn, bits), s.Bus(pOut, bits)
			v := make([]bool, bits)
			return hwsim.TickerFn(func(clk bool) {
				for n, o := range out {
					o.Send(clk, v[n])
				}
				if !clk {
					for n, i := range in {
						v[n] = i.Recv(clk)
					}
				} else {
					for _, i := range in {
						i.Recv(clk)
					}
				}
			})
		}}).NewPart
}
