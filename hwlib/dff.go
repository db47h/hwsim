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
	return (&hwsim.PartSpec{
		Name:    "DFF",
		Inputs:  []string{pIn},
		Outputs: []string{pOut},
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
