package hwsim

import (
	"strconv"
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
	return busPinName(name, bits)
}

// Input creates a function based input.
//
//	Outputs: out
//	Function: out = f()
//
func Input(f func() bool) NewPartFn {
	p := &PartSpec{
		Name: "Input",
		In:   nil,
		Out:  Out{pOut},
		Mount: func(s *Socket) []Component {
			pin := s.Pin(pOut)
			return []Component{
				func(c *Circuit) {
					c.Set(pin, f())
				},
			}
		},
	}
	return p.NewPart
}

// Output creates an output or probe. The fn function is
// called with the named pin state on every circuit update.
//
//	Inputs: in
//	Function: f(in)
//
func Output(f func(bool)) NewPartFn {
	p := &PartSpec{
		Name: "Output",
		In:   In{pIn},
		Out:  nil,
		Mount: func(s *Socket) []Component {
			in := s.Pin(pIn)
			return []Component{
				func(c *Circuit) { f(c.Get(in)) },
			}
		},
	}
	return p.NewPart
}

var notGate = PartSpec{Name: "NOR", In: In{pIn}, Out: Out{pOut},
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
func Not(w string) Part {
	return notGate.NewPart(w)
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
	gateIn  = In{pA, pB}
	gateOut = Out{pOut}

	and  = newGate("AND", func(a, b bool) bool { return a && b })
	nand = newGate("NAND", func(a, b bool) bool { return !(a && b) })
	or   = newGate("OR", func(a, b bool) bool { return a || b })
	nor  = newGate("NOR", func(a, b bool) bool { return !(a || b) })
	xor  = newGate("XOR", func(a, b bool) bool { return a && !b || !a && b })
	xnor = newGate("XNOR", func(a, b bool) bool { return a && b || !a && !b })
)

// And returns a AND gate.
//
func And(w string) Part { return and.NewPart(w) }

// Nand returns a NAND gate.
//
func Nand(w string) Part { return nand.NewPart(w) }

// Or returns a OR gate.
//
func Or(w string) Part { return or.NewPart(w) }

// Nor returns a NOR gate.
//
func Nor(w string) Part { return nor.NewPart(w) }

// Xor returns a XOR gate.
//
func Xor(w string) Part { return xor.NewPart(w) }

// Xnor returns a XNOR gate.
//
func Xnor(w string) Part { return xnor.NewPart(w) }

// Mux returns a multiplexer.
//
//	Inputs: a, b, sel
//	Outputs: out
//	Function: If sel=0 then out=a else out=b.
//
func Mux(w string) Part { return mux.NewPart(w) }

var mux = PartSpec{
	Name: "MUX",
	In:   In{pA, pB, pSel},
	Out:  Out{pOut},
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
func DMux(w string) Part { return dmux.NewPart(w) }

var dmux = PartSpec{
	Name: "DMUX",
	In:   In{pIn, pSel},
	Out:  Out{pA, pB},
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
		In:   ExpandBus(bus(pIn, n)),
		Out:  ExpandBus(bus(pOut, n)),
		Mount: func(s *Socket) []Component {
			ins := s.Bus(pIn, n)
			outs := s.Bus(pOut, n)
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
func Not16(w string) Part { return not16.NewPart(w) }

func inputN(bits int, f func() int64) *PartSpec {
	return &PartSpec{
		Name: "INPUT" + strconv.Itoa(bits),
		In:   nil,
		Out:  ExpandBus(bus(pOut, bits)),
		Mount: func(s *Socket) []Component {
			pins := s.Bus(pOut, bits)
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
func Input16(f func() int64) NewPartFn {
	return inputN(16, f).NewPart
}

func outputN(bits int, f func(int64)) *PartSpec {
	return &PartSpec{
		Name: "OUTPUTBUS" + strconv.Itoa(bits),
		In:   ExpandBus(bus(pIn, bits)),
		Out:  nil,
		Mount: func(s *Socket) []Component {
			pins := s.Bus(pIn, bits)
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
func Output16(f func(int64)) NewPartFn {
	return outputN(16, f).NewPart
}

type gateN struct {
	bits int
	fn   func(bool, bool) bool
}

func (g *gateN) mount(s *Socket) []Component {
	a, b, out := s.Bus(pA, g.bits), s.Bus(pB, g.bits), s.Bus(pOut, g.bits)
	return []Component{
		func(c *Circuit) {
			for i := range a {
				c.Set(out[i], g.fn(c.Get(a[i]), c.Get(b[i])))
			}
		},
	}
}

func newGateN(name string, bits int, f func(bool, bool) bool) *PartSpec {
	g := &gateN{bits, f}
	return &PartSpec{
		Name:  name + strconv.Itoa(bits),
		In:    ExpandBus(bus(pA, 16), bus(pB, 16)),
		Out:   ExpandBus(bus(pOut, bits)),
		Mount: g.mount,
	}
}

var (
	and16  = newGateN("AND", 16, func(a, b bool) bool { return a && b })
	nand16 = newGateN("NAND", 16, func(a, b bool) bool { return !(a && b) })
	or16   = newGateN("OR", 16, func(a, b bool) bool { return a || b })
	nor16  = newGateN("NOR", 16, func(a, b bool) bool { return !(a || b) })
)

// And16 returns a 16 bits AND gate.
//
//	Inputs: a[16], b[16]
//	Outputs: out[16]
//	Function: for i := range out { out[i] = a[i] && b[i] }
//
func And16(w string) Part { return and16.NewPart(w) }

// Nand16 returns a 16 bits NAND gate.
//
//	Inputs: a[16], b[16]
//	Outputs: out[16]
//	Function: for i := range out { out[i] = !(a[i] && b[i]) }
//
func Nand16(w string) Part { return nand16.NewPart(w) }

// Or16 returns a 16 bits OR gate.
//
//	Inputs: a[16], b[16]
//	Outputs: out[16]
//	Function: for i := range out { out[i] = (a[i] || b[i]) }
//
func Or16(w string) Part { return or16.NewPart(w) }

// Nor16 returns a 16 bits NOR gate.
//
//	Inputs: a[16], b[16]
//	Outputs: out[16]
//	Function: for i := range out { out[i] = !(a[i] || b[i]) }
//
func Nor16(w string) Part { return nor16.NewPart(w) }

// DFF returns a clocked data flip flop.
//
//	Inputs: in
//	Outputs: out
//	Function: out(t) = in(t-1)
//
func DFF(w string) Part {
	return (&PartSpec{
		Name: "DFF",
		In:   In{pIn},
		Out:  Out{pOut},
		Mount: func(s *Socket) []Component {
			in, out := s.Pin(pIn), s.Pin(pOut)
			var curOut bool
			return []Component{
				func(c *Circuit) {
					// raising edge?
					// here we cheat by using private fields to speed up clock state tracking
					if c.tick&(c.tpc-1) == 0 {
						curOut = c.Get(in)
					}
					c.Set(out, curOut)
				}}
		}}).NewPart(w)
}
