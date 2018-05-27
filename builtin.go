package hdl

import "errors"

// common pin names
const (
	pinA   = "a"
	pinB   = "b"
	pinIn  = "in"
	pinOut = "out"
)

// standard wiring
type pinout struct {
	pins W
}

func (p *pinout) Pinout() W { return p.pins }

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

// Inputs
type inputImpl struct {
	pin int
	fn  func() bool
}

func (i *inputImpl) Update(c *Circuit) {
	c.Set(i.pin, i.fn())
}

type input struct {
	pinout
	fn func() bool
}

func (i *input) Build(pins map[string]int, _ *Circuit) ([]Updater, error) {
	return []Updater{&inputImpl{pin: pins[pinOut], fn: i.fn}}, nil
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
		pinout: pinout{pins},
		fn:     fn,
	}
}

// Outputs
type outputImpl struct {
	pin int
	fn  func(bool)
}

func (o *outputImpl) Update(c *Circuit) {
	o.fn(c.Get(o.pin))
}

type output struct {
	pinout
	fn func(bool)
}

func (o *output) Build(pins map[string]int, _ *Circuit) ([]Updater, error) {
	return []Updater{&outputImpl{pin: pins[pinIn], fn: o.fn}}, nil
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
		pinout: pinout{pins},
		fn:     fn,
	}
}

// Not gate
type notImpl struct {
	in  int
	out int
}

func (n *notImpl) Update(c *Circuit) {
	c.Set(n.out, !c.Get(n.in))
}

type not struct {
	pinout
}

func (n not) Build(pins map[string]int, _ *Circuit) ([]Updater, error) {
	return []Updater{&notImpl{in: pins[pinIn], out: pins[pinOut]}}, nil
}

// Not returns a NOT gate.
//
func Not(w W) Part {
	if err := checkWiring(w, "in", "out"); err != nil {
		panic(err)
	}
	return &not{pinout: pinout{w}}
}

type gateImpl struct {
	a   int
	b   int
	out int
	fn  func(a, b bool) bool
}

func (g *gateImpl) Update(c *Circuit) {
	c.Set(g.out, g.fn(c.Get(g.a), c.Get(g.b)))
}

type gate struct {
	pinout
	fn func(a, b bool) bool
}

func (g *gate) Build(pins map[string]int, _ *Circuit) ([]Updater, error) {
	return []Updater{&gateImpl{a: pins[pinA], b: pins[pinB], out: pins[pinOut], fn: g.fn}}, nil
}

func newGate(w W, fn func(bool, bool) bool) Part {
	if err := checkWiring(w, pinA, pinB, pinOut); err != nil {
		panic(err)
	}
	return &gate{
		pinout: pinout{w},
		fn:     fn,
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
