package hdl

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
		Build: func(pins map[string]int, _ *Circuit) []Component {
			pin := pins[pOut]
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
		Build: func(pins map[string]int, _ *Circuit) []Component {
			in := pins[pIn]
			return []Component{
				func(c *Circuit) { f(c.Get(in)) },
			}
		},
	}
	return p.Wire(w)
}

var notGate = PartSpec{Name: "NOR", In: []string{pIn}, Out: []string{pOut},
	Build: func(pins map[string]int, _ *Circuit) []Component {
		in, out := pins[pIn], pins[pOut]
		return []Component{
			func(c *Circuit) { c.Set(out, !c.Get(in)) },
		}
	}}

// Not returns a NOT gate.
//
// Input pin name: in
//
func Not(w W) Part {
	return notGate.Wire(w)
}

// other gates
type gate func(a, b bool) bool

func (g gate) Build(pins map[string]int, _ *Circuit) []Component {
	a, b, out := pins[pA], pins[pB], pins[pOut]
	return []Component{
		func(c *Circuit) { c.Set(out, g(c.Get(a), c.Get(b))) },
	}
}

func newGate(name string, fn func(a, b bool) bool) *PartSpec {
	return &PartSpec{
		Name:  name,
		In:    gateIn,
		Out:   gateOut,
		Build: gate(fn).Build,
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
	Build: func(pins map[string]int, _ *Circuit) []Component {
		a, b, sel, out := pins[pA], pins[pB], pins[pSel], pins[pOut]
		return []Component{func(c *Circuit) {
			if c.Get(sel) {
				c.Set(out, c.Get(b))
			} else {
				c.Set(out, c.Get(a))
			}
		}}
	}}

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
	Build: func(pins map[string]int, _ *Circuit) []Component {
		in, sel, a, b := pins[pIn], pins[pSel], pins[pA], pins[pB]
		return []Component{func(c *Circuit) {
			if c.Get(sel) {
				c.Set(a, false)
				c.Set(b, c.Get(in))
			} else {
				c.Set(a, c.Get(in))
				c.Set(b, false)
			}
		}}
	}}
