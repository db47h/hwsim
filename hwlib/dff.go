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
	return dffSpec.NewPart(c)
}

var dffSpec = &hwsim.PartSpec{
	Name:    "DFF",
	Inputs:  []string{pIn},
	Outputs: []string{pOut},
	Mount: func(s *hwsim.Socket) hwsim.Updater {
		return &dff{in: s.Wire(pIn), out: s.Wire(pOut)}
	}}

type dff struct {
	in, out *hwsim.Wire
	v       bool
}

func (d *dff) Update(clk bool) {
	// send first in order to prevent recursion
	d.out.Send(clk, d.v)
}

func (d *dff) PostUpdate(clk bool) {
	// force input update
	v := d.in.Recv(clk)
	// change value only at ticks
	if !clk {
		d.v = v
	}
}

// DFFN creates a N bits DFF.
//
func DFFN(bits int) hwsim.NewPartFn {
	bs := strconv.Itoa(bits)
	return (&hwsim.PartSpec{
		Name:    "DFF" + bs,
		Inputs:  bus(bits, pIn),
		Outputs: bus(bits, pOut),
		Mount: func(s *hwsim.Socket) hwsim.Updater {
			return &dffN{
				in:  s.Bus(pIn, bits),
				out: s.Bus(pOut, bits),
				v:   make([]bool, bits),
			}
		}}).NewPart
}

type dffN struct {
	in, out hwsim.Bus
	v       []bool
}

func (d *dffN) Update(clk bool) {
	for n, o := range d.out {
		o.Send(clk, d.v[n])
	}
}

func (d *dffN) PostUpdate(clk bool) {
	if !clk {
		for n, i := range d.in {
			d.v[n] = i.Recv(clk)
		}
	} else {
		for _, i := range d.in {
			i.Recv(clk)
		}
	}
}
