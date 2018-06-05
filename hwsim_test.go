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
