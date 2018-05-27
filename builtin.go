package hdl

import "errors"

// common pin names
const (
	pinA   = "a"
	pinB   = "b"
	pinIn  = "in"
	pinOut = "out"
)

// wires is a wrapper to provide the Pinout() method for free
// to structs implementing Part.
type wires W

// TODO: rename Pinout. We may provide othe pin lists in the future. And this function actually
// returns how a part's external pins are connected to pins in its container or user.
func (p wires) Pinout() W { return W(p) }

// check that the wiring w mathes with a part's exposed pins ex
func checkWiring(w W, ex ...string) error {
	if len(w) != len(ex) {
		return errors.New("wrong number of arguments")
	}
	for _, name := range ex {
		if _, ok := w[name]; !ok {
			return errors.New("pin " + name + " not connected")
		}
	}
	return nil
}

type input struct {
	wires
	fn func() bool
}

func (i *input) Build(pins map[string]int, _ *Circuit) ([]Updater, error) {
	pin := pins[pinOut]
	return []Updater{
		func(c *Circuit) {
			c.Set(pin, i.fn())
		},
	}, nil
}

// Input creates a function based input.
//
// Output pin name: out
//
func Input(pins W, fn func() bool) Part {
	if err := checkWiring(pins, pinOut); err != nil {
		panic(err)
	}
	return &input{
		wires: wires(pins),
		fn:    fn,
	}
}

// Outputs
type output struct {
	wires
	fn func(bool)
}

func (o *output) Build(pins map[string]int, _ *Circuit) ([]Updater, error) {
	pin := pins[pinIn]
	return []Updater{
		func(c *Circuit) { o.fn(c.Get(pin)) },
	}, nil
}

// Output creates an output or probe. The fn function is
// called with the named pin state on every circuit update.
//
// Input pin name: in
//
func Output(pins W, fn func(bool)) Part {
	if err := checkWiring(pins, pinIn); err != nil {
		panic(err)
	}

	return &output{
		wires: wires(pins),
		fn:    fn,
	}
}

// Not gate
type not struct {
	wires
}

func (n not) Build(pins map[string]int, _ *Circuit) ([]Updater, error) {
	in, out := pins[pinIn], pins[pinOut]
	return []Updater{
		func(c *Circuit) { c.Set(out, !c.Get(in)) },
	}, nil
}

// Not returns a NOT gate.
//
func Not(w W) Part {
	if err := checkWiring(w, "in", "out"); err != nil {
		panic(err)
	}
	return &not{wires: wires(w)}
}

type gate struct {
	wires
	fn func(a, b bool) bool
}

func (g *gate) Build(pins map[string]int, _ *Circuit) ([]Updater, error) {
	a, b, out := pins[pinA], pins[pinB], pins[pinOut]
	return []Updater{
		func(c *Circuit) { c.Set(out, g.fn(c.Get(a), c.Get(b))) },
	}, nil
}

func newGate(w W, fn func(bool, bool) bool) Part {
	if err := checkWiring(w, pinA, pinB, pinOut); err != nil {
		panic(err)
	}
	return &gate{
		wires: wires(w),
		fn:    fn,
	}
}

// And returns a AND gate.
//
func And(w W) Part {
	return newGate(w, func(a, b bool) bool { return a && b })
}

// Nand returns a NAND gate.
//
func Nand(w W) Part {
	return newGate(w, func(a, b bool) bool { return !(a && b) })
}

// Or returns a OR gate.
//
func Or(w W) Part { return newGate(w, func(a, b bool) bool { return a || b }) }

// Nor returns a NOR gate.
//
func Nor(w W) Part {
	return newGate(w, func(a, b bool) bool { return !(a || b) })
}

// Xor returns a XOR gate.
//
func Xor(w W) Part {
	return newGate(w, func(a, b bool) bool { return a && !b || !a && b })
}

// Xnor returns a XNOR gate.
//
func Xnor(w W) Part {
	return newGate(w, func(a, b bool) bool { return a && b || !a && !b })
}
