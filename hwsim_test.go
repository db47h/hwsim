package hwsim_test

import (
	"testing"

	hw "github.com/db47h/hwsim"
	"github.com/pkg/errors"
)

const testTPC = 16

func trace(t *testing.T, err error) {
	t.Helper()
	if err, ok := err.(interface {
		StackTrace() errors.StackTrace
	}); ok {
		for _, f := range err.StackTrace() {
			t.Logf("%+v ", f)
		}
	}
}

func Test_gate_custom(t *testing.T) {
	and, err := hw.Chip("AND", hw.In{"a", "b"}, hw.Out{"out"},
		hw.Parts{
			hw.Nand(hw.W{"a": "a", "b": "b", "out": "nand"}),
			hw.Nand(hw.W{"a": "nand", "b": "nand", "out": "out"}),
		})
	if err != nil {
		t.Fatal(err)
	}
	or, err := hw.Chip("OR", hw.In{"a", "b"}, hw.Out{"out"},
		hw.Parts{
			hw.Nand(hw.W{"a": "a", "b": "a", "out": "notA"}),
			hw.Nand(hw.W{"a": "b", "b": "b", "out": "notB"}),
			hw.Nand(hw.W{"a": "notA", "b": "notB", "out": "out"}),
		})
	if err != nil {
		t.Fatal(err)
	}
	nor, err := hw.Chip("NOR", hw.In{"a", "b"}, hw.Out{"out"},
		hw.Parts{
			or(hw.W{"a": "a", "b": "b", "out": "orAB"}),
			hw.Nand(hw.W{"a": "orAB", "b": "orAB", "out": "out"}),
		})
	if err != nil {
		t.Fatal(err)
	}
	xor, err := hw.Chip("XOR", hw.In{"a", "b"}, hw.Out{"out"},
		hw.Parts{
			hw.Nand(hw.W{"a": "a", "b": "b", "out": "nandAB"}),
			hw.Nand(hw.W{"a": "a", "b": "nandAB", "out": "w0"}),
			hw.Nand(hw.W{"a": "b", "b": "nandAB", "out": "w1"}),
			hw.Nand(hw.W{"a": "w0", "b": "w1", "out": "out"}),
		})
	if err != nil {
		t.Fatal(err)
	}
	xnor, err := hw.Chip("XNOR", hw.In{"a", "b"}, hw.Out{"out"},
		hw.Parts{
			or(hw.W{"a": "a", "b": "b", "out": "or"}),
			hw.Nand(hw.W{"a": "a", "b": "b", "out": "nand"}),
			hw.Nand(hw.W{"a": "or", "b": "nand", "out": "out"}),
		})
	if err != nil {
		t.Fatal(err)
	}
	not, err := hw.Chip("NOT", hw.In{"a"}, hw.Out{"out"},
		hw.Parts{
			hw.Nand(hw.W{"a": "a", "b": "a", "out": "out"}),
		})
	if err != nil {
		t.Fatal(err)
	}
	mux, err := hw.Chip("NUX", hw.In{"a", "b", "sel"}, hw.Out{"out"}, hw.Parts{
		hw.Not(hw.W{"in": "sel", "out": "notSel"}),
		hw.And(hw.W{"a": "a", "b": "notSel", "out": "w0"}),
		hw.And(hw.W{"a": "b", "b": "sel", "out": "w1"}),
		hw.Or(hw.W{"a": "w0", "b": "w1", "out": "out"}),
	})
	if err != nil {
		t.Fatal(err)
	}
	dmux, err := hw.Chip("DMUX", hw.In{"in", "sel"}, hw.Out{"a", "b"}, hw.Parts{
		hw.Not(hw.W{"in": "sel", "out": "notSel"}),
		hw.And(hw.W{"a": "in", "b": "notSel", "out": "a"}),
		hw.And(hw.W{"a": "in", "b": "sel", "out": "b"}),
	})
	if err != nil {
		t.Fatal(err)
	}
	td := []struct {
		name   string
		gate   hw.NewPartFn
		result [][]bool
	}{
		{"AND", and, [][]bool{{false, false, false, true}}},
		{"OR", or, [][]bool{{false, true, true, true}}},
		{"NOR", nor, [][]bool{{true, false, false, false}}},
		{"XOR", xor, [][]bool{{false, true, true, false}}},
		{"XNOR", xnor, [][]bool{{true, false, false, true}}},
		{"NOT", not, [][]bool{{true, false}}},
		{"MUX", mux, [][]bool{{false, false, false, true, true, false, true, true}}},
		{"DMUX", dmux, [][]bool{{false, false, true, false}, {false, false, false, true}}},
	}
	for _, d := range td {
		t.Run(d.name, func(t *testing.T) {
			testGate(t, d.name, d.gate, d.result)
		})
	}
}

// Test a basic clock with a Nor gate.
//
// The purpose of this test is to catch changes in propagation delays
// from Inputs and Outputs as well as testing loops between input and outputs.
//
// Clocks should be implemented as custom components or inputs. Really.
//
func Test_clock(t *testing.T) {
	var disable, tick bool

	check := func(v bool) {
		t.Helper()
		if tick != v {
			t.Errorf("expected %v, got %v", v, tick)
		}
	}
	// we could implement the clock directly as a Nor in the cisrcuit (with no less gate delays)
	// but we wrap it into a stand-alone chip in order to add a layer complexity
	// for testing purposes.
	clk, err := hw.Chip("CLK", hw.In{"disable"}, hw.Out{"tick"}, hw.Parts{
		hw.Nor(hw.W{"a": "disable", "b": "tick", "out": "tick"}),
	})
	if err != nil {
		t.Fatal(err)
	}
	c, err := hw.NewCircuit(0, testTPC, hw.Parts{
		hw.Input(func() bool { return disable })(hw.W{"out": "disable"}),
		clk(hw.W{"disable": "disable", "tick": "out"}),
		hw.Output(func(out bool) { tick = out })(hw.W{"in": "out"}),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Dispose()

	// we have two wires: "disable" and "out".
	// note that Output("out", ...) is delayed by one tick after the Nand updates it.

	disable = true
	c.Step()
	check(false)
	c.Step()
	// this is an expected signal change appearing in the first couple of ticks due to signal propagation delay
	check(true)
	c.Step()
	check(false)
	c.Step()
	check(false)

	disable = false
	c.Step()
	check(false)
	c.Step()
	check(false)
	c.Step()
	// the clock starts ticking now.
	check(true)
	c.Step()
	check(false)
	c.Step()
	check(true)
	disable = true
	c.Step()
	check(false)
	c.Step()
	check(true)
	c.Step()
	// the clock stops ticking now.
	check(false)
	c.Step()
	check(false)
}

func Test_chip_errors(t *testing.T) {
	unkChip, err := hw.Chip("TESTCHIP", hw.In{"a", "b"}, hw.Out{"out"}, hw.Parts{
		// chip input a is unused
		hw.Nand(hw.W{"a": "b", "b": "b", "out": "out"}),
	})
	if err != nil {
		t.Fatal(err)
	}
	data := []struct {
		name  string
		in    hw.In
		out   hw.Out
		parts hw.Parts
		err   string
	}{
		{"true_out", hw.In{"a", "b"}, hw.Out{"out"}, hw.Parts{
			hw.Nand(hw.W{"a": "a", "b": "b", "out": hw.True}),
			hw.Nand(hw.W{"a": "a", "b": "b", "out": "out"}),
		}, "NAND.out:true: output pin connected to constant true input"},
		{"false_out", hw.In{"a", "b"}, hw.Out{"out"}, hw.Parts{
			hw.Nand(hw.W{"a": "a", "b": "b", "out": hw.False}),
			hw.Nand(hw.W{"a": "a", "b": "b", "out": "out"}),
		}, "NAND.out:false: output pin connected to constant false input"},
		{"multi_out", hw.In{"a", "b"}, hw.Out{"out"}, hw.Parts{
			hw.Nand(hw.W{"a": "a", "b": "b", "out": "a"}),
			hw.Nand(hw.W{"a": "a", "b": "b", "out": "out"}),
		}, "NAND.out:a: chip input pin used as output"},
		{"multi_out2", hw.In{"a", "b"}, hw.Out{"out"}, hw.Parts{
			hw.Nand(hw.W{"a": "a", "b": "b", "out": "x"}),
			hw.Nand(hw.W{"a": "a", "b": "b", "out": "x"}),
			hw.Not(hw.W{"in": "x", "out": "out"}),
		}, "NAND.out:x: output pin already used as output"},
		{"no_output", hw.In{"a", "b"}, hw.Out{"out"}, hw.Parts{
			hw.Nand(hw.W{"a": "a", "b": "wx", "out": "out"}),
		}, "pin wx not connected to any output"},
		{"no_input", hw.In{"a", "b"}, hw.Out{"out"}, hw.Parts{
			hw.Nand(hw.W{"a": "a", "b": "b", "out": "foo"}),
			hw.Nand(hw.W{"a": "a", "b": "b", "out": "out"}),
		}, "pin foo not connected to any input"},
		{"unconnected_in", hw.In{"a", "b"}, hw.Out{"out"}, hw.Parts{}, ""},
		{"unknown_pin", hw.In{"a", "b"}, hw.Out{"out"}, hw.Parts{
			hw.Nand(hw.W{"a": "a", "typo": "b", "out": "out"}),
		}, "invalid pin name typo for part NAND"},
		{"unknown_pin", hw.In{"a", "b"}, hw.Out{"out"}, hw.Parts{
			unkChip(hw.W{"a": "a", "typo": "b", "out": "out"}),
		}, "invalid pin name typo for part TESTCHIP"},
		{"unknown_pin", hw.In{"a", "b"}, hw.Out{"out"}, hw.Parts{
			unkChip(hw.W{"a": "a", "b": "b", "out": "out"}),
		}, ""},
	}
	for _, d := range data {
		t.Run(d.name, func(t *testing.T) {
			_, err := hw.Chip(d.name, d.in, d.out, d.parts)
			if err == nil && d.err != "" || err != nil && err.Error() != d.err {
				t.Errorf("Got error %q, expected %q", err, d.err)
				return
			}
		})
	}

}

func Test_omitted_pins(t *testing.T) {
	var a, b, c, tr, f, o0, o1 int
	dummy := hw.MakePart(&hw.PartSpec{
		Name: "dummy",
		In:   hw.In{"a", "b", "c", "t", "f"},
		Out:  hw.Out{"o0", "o1"},
		Mount: func(s *hw.Socket) []hw.Component {
			a, b, c, tr, f, o0, o1 = s.Pin("a"), s.Pin("b"), s.Pin("c"), s.Pin("t"), s.Pin("f"), s.Pin("o0"), s.Pin("o1")
			return nil
		}})
	// this is just to add another layer of testing.
	// inspecting o0 and o1 shows that another dummy wire was allocated for dummy.o0:wo0
	wrapper, err := hw.Chip("wrapper", hw.In{"wa", "wb"}, hw.Out{"wo0", "wo1"}, hw.Parts{
		dummy(hw.W{"a": "wa", "c": hw.Clk, "t": hw.True, "f": hw.False, "o0": "wo0"}),
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = hw.NewCircuit(0, 0, hw.Parts{wrapper(nil)})
	if err != nil {
		t.Fatal(err)
	}

	if a != 0 || b != 0 || f != 0 { // 0 = cstFalse
		t.Errorf("a = %v, b = %v, f = %v, all must be 0", a, b, f)
	}
	if tr != 1 { // 1 = cstTrue
		t.Errorf("t = %v, must be 1", tr)
	}
	if c != 2 { // 2 = cstClk
		t.Errorf("c = %v, must be 2", c)
	}
	if o0 < 3 || o1 < 3 { // 3 = cstCount
		t.Errorf("o0 = %v, o1 = %v, both must be > 3", o0, o1)
	}
}
