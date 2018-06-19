// Copyright 2018 Denis Bernard <db047h@gmail.com>
// Licensed under the MIT license. See license text in the LICENSE file.

package hwlib

import "github.com/db47h/hwsim"

// DFF returns a clocked data flip flop.
//
//	Inputs: in
//	Outputs: out
//	Function: out(t) = in(t-1) // where t is the current clock cycle.
//
func DFF(w string) hwsim.Part {
	return dff.NewPart(w)
}

var dff = &hwsim.PartSpec{
	Name:    "DFF",
	Inputs:  []string{pIn},
	Outputs: []string{pOut},
	Mount: func(s *hwsim.Socket) hwsim.Updater {
		return &dffImpl{in: s.Pin(pIn), out: s.Pin(pOut)}
	}}

type dffImpl struct {
	in, out *hwsim.Pin
	v       bool
	n       bool
}

func (d *dffImpl) Update(clk bool) {
	if clk {
		// send first in order to prevent update loops
		d.out.Send(clk, d.v)
		d.n = d.in.Recv(clk)
	} else {
		d.v = d.n
		d.out.Send(clk, d.v)
	}
}

func (*dffImpl) Tick() {}
