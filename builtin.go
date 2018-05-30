package hdl

import "strconv"

// common pin names
const (
	pA   = "a"
	pB   = "b"
	pIn  = "in"
	pSel = "sel"
	pOut = "out"
)

// Input creates a function based input.
//
// Output pin name: out
//
func Input(w W, f func() bool) Part {
	p := &PartSpec{
		Name: "Input",
		In:   nil,
		Out:  []string{pOut},
		Mount: func(s *Socket) []Component {
			pin := s.Pin(pOut)
			return []Component{
				func(c *Circuit) {
					c.Set(pin, f())
				},
			}
		},
	}
	return p.Wire(w)
}

// Output creates an output or probe. The fn function is
// called with the named pin state on every circuit update.
//
// Input pin name: in
//
func Output(w W, f func(bool)) Part {
	p := &PartSpec{
		Name: "Output",
		In:   []string{pIn},
		Out:  nil,
		Mount: func(s *Socket) []Component {
			in := s.Pin(pIn)
			return []Component{
				func(c *Circuit) { f(c.Get(in)) },
			}
		},
	}
	return p.Wire(w)
}

var notGate = PartSpec{Name: "NOR", In: []string{pIn}, Out: []string{pOut},
	Mount: func(s *Socket) []Component {
		in, out := s.Pin(pIn), s.Pin(pOut)
		return []Component{
			func(c *Circuit) { c.Set(out, !c.Get(in)) },
		}
	},
}

// Not returns a NOT gate.
//
// Input pin name: in
//
func Not(w W) Part {
	return notGate.Wire(w)
}

// other gates
type gate func(a, b bool) bool

func (g gate) mount(s *Socket) []Component {
	a, b, out := s.Pin(pA), s.Pin(pB), s.Pin(pOut)
	return []Component{
		func(c *Circuit) { c.Set(out, g(c.Get(a), c.Get(b))) },
	}
}

func newGate(name string, fn func(a, b bool) bool) *PartSpec {
	return &PartSpec{
		Name:  name,
		In:    gateIn,
		Out:   gateOut,
		Mount: gate(fn).mount,
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
func And(w W) Part { return and.Wire(w) }

// Nand returns a NAND gate.
//
func Nand(w W) Part { return nand.Wire(w) }

// Or returns a OR gate.
//
func Or(w W) Part { return or.Wire(w) }

// Nor returns a NOR gate.
//
func Nor(w W) Part { return nor.Wire(w) }

// Xor returns a XOR gate.
//
func Xor(w W) Part { return xor.Wire(w) }

// Xnor returns a XNOR gate.
//
func Xnor(w W) Part { return xnor.Wire(w) }

// Mux returns a multiplexer.
//
//	Inputs: a, b, sel
//	Outputs: out
//	Function: If sel=0 then out=a else out=b.
//
func Mux(w W) Part { return mux.Wire(w) }

var mux = PartSpec{
	Name: "MUX",
	In:   []string{pA, pB, pSel},
	Out:  []string{pOut},
	Mount: func(s *Socket) []Component {
		a, b, sel, out := s.Pin(pA), s.Pin(pB), s.Pin(pSel), s.Pin(pOut)
		return []Component{func(c *Circuit) {
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
func DMux(w W) Part { return dmux.Wire(w) }

var dmux = PartSpec{
	Name: "DMUX",
	In:   []string{pIn, pSel},
	Out:  []string{pA, pB},
	Mount: func(s *Socket) []Component {
		in, sel, a, b := s.Pin(pIn), s.Pin(pSel), s.Pin(pA), s.Pin(pB)
		return []Component{func(c *Circuit) {
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

// nbits gates
func notN(n int) *PartSpec {
	return &PartSpec{
		Name: "NOT" + strconv.Itoa(n),
		In:   Bus(pIn, n),
		Out:  Bus(pOut, n),
		Mount: func(s *Socket) []Component {
			ins := s.Bus(pIn)
			outs := s.Bus(pOut)
			return []Component{func(c *Circuit) {
				for i, pin := range ins {
					c.Set(outs[i], !c.Get(pin))
				}
			}}
		},
	}
}

var (
	not16 = notN(16)
)

// Not16 returns a 16 bits NOT gate.
//
func Not16(w W) Part { return not16.Wire(w) }

func inputN(bits int, f func() int64) *PartSpec {
	return &PartSpec{
		Name: "INPUTBUS" + strconv.Itoa(bits),
		In:   nil,
		Out:  Bus(pOut, bits),
		Mount: func(s *Socket) []Component {
			pins := s.Bus(pOut)
			return []Component{func(c *Circuit) {
				in := f()
				for bit := 0; bit < len(pins); bit++ {
					c.Set(pins[bit], in&(1<<uint(bit)) != 0)
				}
			}}
		},
	}
}

// Input16 creates a 16 bits input bus.
//
func Input16(w W, f func() int64) Part {
	return inputN(16, f).Wire(w)
}

func outputN(bits int, f func(int64)) *PartSpec {
	return &PartSpec{
		Name: "OUTPUTBUS" + strconv.Itoa(bits),
		In:   Bus(pIn, bits),
		Out:  nil,
		Mount: func(s *Socket) []Component {
			pins := s.Bus(pIn)
			return []Component{func(c *Circuit) {
				var out int64
				for i := 0; i < len(pins); i++ {
					if c.Get(pins[i]) {
						out |= 1 << uint(i)
					}
				}
				f(out)
			}}
		},
	}
}

// Output16 creates a 16 bits output bus.
//
func Output16(w W, f func(int64)) Part {
	return outputN(16, f).Wire(w)
}
