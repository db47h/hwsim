package hdl

// common pin names
const (
	pinA   = "a"
	pinB   = "b"
	pinIn  = "in"
	pinOut = "out"
)

// Input creates a function based input.
//
// Output pin name: out
//
func Input(w W, f func() bool) Part {
	p := &PartSpec{
		In:  nil,
		Out: []string{pinOut},
		Build: func(pins map[string]int, _ *Circuit) ([]Updater, error) {
			pin := pins[pinOut]
			return []Updater{
				func(c *Circuit) {
					c.Set(pin, f())
				},
			}, nil
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
		In:  []string{pinIn},
		Out: nil,
		Build: func(pins map[string]int, _ *Circuit) ([]Updater, error) {
			in := pins[pinIn]
			return []Updater{
				func(c *Circuit) { f(c.Get(in)) },
			}, nil
		},
	}
	return p.Wire(w)
}

var notGate = PartSpec{
	In:  []string{pinIn},
	Out: []string{pinOut},
	Build: func(pins map[string]int, _ *Circuit) ([]Updater, error) {
		in, out := pins[pinIn], pins[pinOut]
		return []Updater{
			func(c *Circuit) { c.Set(out, !c.Get(in)) },
		}, nil
	},
}

// Not returns a NOT gate.
//
func Not(w W) Part {
	return notGate.Wire(w)
}

// other gates
type gate func(a, b bool) bool

func (g gate) Build(pins map[string]int, _ *Circuit) ([]Updater, error) {
	a, b, out := pins[pinA], pins[pinB], pins[pinOut]
	return []Updater{
		func(c *Circuit) { c.Set(out, g(c.Get(a), c.Get(b))) },
	}, nil
}

func newGate(fn func(a, b bool) bool) *PartSpec {
	return &PartSpec{
		In:    gateIn,
		Out:   gateOut,
		Build: gate(fn).Build,
	}
}

var (
	gateIn  = []string{pinA, pinB}
	gateOut = []string{pinOut}

	and  = newGate(func(a, b bool) bool { return a && b })
	nand = newGate(func(a, b bool) bool { return !(a && b) })
	or   = newGate(func(a, b bool) bool { return a || b })
	nor  = newGate(func(a, b bool) bool { return !(a || b) })
	xor  = newGate(func(a, b bool) bool { return a && !b || !a && b })
	xnor = newGate(func(a, b bool) bool { return a && b || !a && !b })
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
