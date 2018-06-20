package hwsim_test

import (
	"fmt"

	hw "github.com/db47h/hwsim"
)

// mux4 is a custom 4 bits mux.
//
type mux4Impl struct {
	A   [4]*hw.Wire `hw:"in"`     // input bus "a"
	B   [4]*hw.Wire `hw:"in"`     // input bus "b"
	S   *hw.Wire    `hw:"in,sel"` // single pin, the second tag value forces the pin name to "sel"
	Out [4]*hw.Wire `hw:"out"`    // output bus "out"
}

// Update implements Updater.
//
func (m *mux4Impl) Update(clk bool) {
	if m.S.Recv(clk) {
		for i, b := range m.B {
			m.Out[i].Send(clk, b.Recv(clk))
		}
	} else {
		for i, a := range m.A {
			m.Out[i].Send(clk, a.Recv(clk))
		}
	}
}

// no need to import reflect, just cast a nil pointer to mux4
var m4Spec = hw.MakePart((*mux4Impl)(nil))

// m4Spec is the *PartSpec for our mux4. In order to use it like the built-ins
// in hwlib, we need to get its NewPartFn method as a variable, or make it a function:
func Mux4(c string) hw.Part { return m4Spec.NewPart(c) }

// MakePart example with a custom Mux4
func ExampleMakePart() {
	var a, b, out int64
	var sel bool
	c, err := hw.NewCircuit(
		// IOs to test the circuit
		hw.InputN(4, func() int64 { return a })("out[0..3]=in_a[0..3]"),
		hw.InputN(4, func() int64 { return b })("out[0..3]=in_b[0..3]"),
		hw.Input(func() bool { return sel })("out=in_sel"),
		// our custom Mux4
		Mux4("a[0..3]=in_a[0..3], b[0..3]=in_b[0..3], sel=in_sel, out[0..3]=mux_out[0..3]"),
		// IOs continued...
		hw.OutputN(4, func(v int64) { out = v })("in[0..3]=mux_out[0..3]"),
	)
	if err != nil {
		panic(err)
	}

	a, b, sel = 1, 15, false
	c.TickTock()
	fmt.Printf("a=%d, b=%d, sel=%v => out=%d\n", a, b, sel, out)
	sel = true
	c.TickTock()
	fmt.Printf("a=%d, b=%d, sel=%v => out=%d\n", a, b, sel, out)

	// Output:
	// a=1, b=15, sel=false => out=1
	// a=1, b=15, sel=true => out=15
}
