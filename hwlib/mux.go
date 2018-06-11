package hwlib

import "github.com/db47h/hwsim"

// Mux returns a multiplexer.
//
//	Inputs: a, b, sel
//	Outputs: out
//	Function: If sel=0 then out=a else out=b.
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
//	Function: If sel=0 then {a=in, b=0} else {a=0, b=in}
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
