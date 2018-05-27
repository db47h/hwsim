package hdl

// Inputs
type inputImpl struct {
	pin int
	fn  func() bool
}

func (i *inputImpl) Update(c *Circuit) {
	c.Set(i.pin, i.fn())
}

type input struct {
	pins []string
	fn   func() bool
}

func (i *input) Pinout() ([]string, []string) { return nil, i.pins }
func (i *input) Build(pins []int, _ *Circuit) ([]Updater, error) {
	return []Updater{&inputImpl{pin: pins[0], fn: i.fn}}, nil
}

// Input creates a function based input.
//
func Input(pin string, fn func() bool) Part {
	return &input{
		pins: []string{pin},
		fn:   fn,
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
	pins []string
	fn   func(bool)
}

func (o *output) Pinout() ([]string, []string) { return o.pins, nil }
func (o *output) Build(pins []int, _ *Circuit) ([]Updater, error) {
	return []Updater{&outputImpl{pin: pins[0], fn: o.fn}}, nil
}

// Output creates an output or probe. The fn function is
// called with the named pin state on every circuit update.
//
func Output(pin string, fn func(bool)) Part {
	return &output{
		pins: []string{pin},
		fn:   fn,
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

type not [2]string

func (n not) Pinout() ([]string, []string) { return n[:1], n[1:] }
func (n not) Build(pins []int, _ *Circuit) ([]Updater, error) {
	return []Updater{&notImpl{in: pins[0], out: pins[1]}}, nil
}

// Not returns a Not gate.
//
func Not(in, out string) Part {
	return not{in, out}
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
	pins []string
	fn   func(a, b bool) bool
}

func (g *gate) Pinout() ([]string, []string) { return g.pins[0:2], g.pins[2:] }
func (g *gate) Build(pins []int, _ *Circuit) ([]Updater, error) {
	return []Updater{&gateImpl{a: pins[0], b: pins[1], out: pins[2], fn: g.fn}}, nil
}

// NewGate returns a Gate-like chip with two inputs, one output
// where the output is the result of fn(inA, inB).
//
func NewGate(a, b, out string, fn func(bool, bool) bool) Part {
	return &gate{
		pins: []string{a, b, out},
		fn:   fn,
	}
}

// And returns a AND gate.
//
func And(a, b, out string) Part {
	return NewGate(a, b, out, func(a, b bool) bool { return a && b })
}

// Nand returns a NAND gate.
//
func Nand(a, b, out string) Part {
	return NewGate(a, b, out, func(a, b bool) bool { return !(a && b) })
}

// Or returns a OR gate.
//
func Or(a, b, out string) Part { return NewGate(a, b, out, func(a, b bool) bool { return a || b }) }

// Nor returns a NOR gate.
//
func Nor(a, b, out string) Part {
	return NewGate(a, b, out, func(a, b bool) bool { return !(a || b) })
}

// Xor returns a XOR gate.
//
func Xor(a, b, out string) Part {
	return NewGate(a, b, out, func(a, b bool) bool { return a && !b || !a && b })
}

// Xnor returns a XNOR gate.
//
func Xnor(a, b, out string) Part {
	return NewGate(a, b, out, func(a, b bool) bool { return a && b || !a && !b })
}
