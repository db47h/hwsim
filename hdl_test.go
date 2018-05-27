package hdl_test

import (
	"runtime"
	"testing"

	"github.com/db47h/hdl"
)

var workers = runtime.NumCPU() // this is naive but should be good enough for testing.

func testGate(t *testing.T, name string, gate hdl.NewPartFunc, result []bool) {
	var a, b, out bool
	t.Helper()
	c, err := hdl.NewCircuit([]hdl.Part{
		hdl.Input(hdl.W{"out": "a"}, func() bool { return a }),
		hdl.Input(hdl.W{"out": "b"}, func() bool { return b }),
		gate(hdl.W{"a": "a", "b": "b", "out": "out"}),
		hdl.Output(hdl.W{"in": "out"}, func(o bool) { out = o }),
	})
	if err != nil {
		t.Fatal(err)
	}

	res := 0
	for _, a = range []bool{false, true} {
		for _, b = range []bool{false, true} {
			for i := 0; i < 10; i++ {
				c.Update(workers)
			}
			if out != result[res] {
				t.Errorf("got %v %s %v = %v, expected %v", a, name, b, out, result[res])
			}
			res++
		}
	}
}

func Test_gate_builtin(t *testing.T) {
	// turn a not into a 2-input gate that ignores b
	not := hdl.Chip([]string{"a", "b"}, []string{"out"},
		[]hdl.Part{
			hdl.Output(hdl.W{"in": "b"}, func(bool) {}), // eat b silently
			hdl.Not(hdl.W{"in": "a", "out": "out"}),
		})
	td := []struct {
		name   string
		gate   hdl.NewPartFunc
		result []bool // a=0 && b=0, a=0 && b=1, a=1 && b=0, a=1 && b=1
	}{
		{"AND", hdl.And, []bool{false, false, false, true}},
		{"NAND", hdl.Nand, []bool{true, true, true, false}},
		{"OR", hdl.Or, []bool{false, true, true, true}},
		{"NOR", hdl.Nor, []bool{true, false, false, false}},
		{"XOR", hdl.Xor, []bool{false, true, true, false}},
		{"XNOR", hdl.Xnor, []bool{true, false, false, true}},
		{"NOT", not, []bool{true, true, false, false}},
	}
	for _, d := range td {
		t.Run(d.name, func(t *testing.T) {
			testGate(t, d.name, d.gate, d.result)
		})
	}
}

func Test_gate_custom(t *testing.T) {
	and := hdl.Chip([]string{"a", "b"}, []string{"out"},
		[]hdl.Part{
			hdl.Nand(hdl.W{"a": "a", "b": "b", "out": "nand"}),
			hdl.Nand(hdl.W{"a": "nand", "b": "nand", "out": "out"}),
		})
	or := hdl.Chip([]string{"a", "b"}, []string{"out"},
		[]hdl.Part{
			hdl.Nand(hdl.W{"a": "a", "b": "a", "out": "notA"}),
			hdl.Nand(hdl.W{"a": "b", "b": "b", "out": "notB"}),
			hdl.Nand(hdl.W{"a": "notA", "b": "notB", "out": "out"}),
		})
	nor := hdl.Chip([]string{"a", "b"}, []string{"out"},
		[]hdl.Part{
			or(hdl.W{"a": "a", "b": "b", "out": "orAB"}),
			hdl.Nand(hdl.W{"a": "orAB", "b": "orAB", "out": "out"}),
		})
	xor := hdl.Chip([]string{"a", "b"}, []string{"out"},
		[]hdl.Part{
			hdl.Nand(hdl.W{"a": "a", "b": "b", "out": "nandAB"}),
			hdl.Nand(hdl.W{"a": "a", "b": "nandAB", "out": "w0"}),
			hdl.Nand(hdl.W{"a": "b", "b": "nandAB", "out": "w1"}),
			hdl.Nand(hdl.W{"a": "w0", "b": "w1", "out": "out"}),
		})
	xnor := hdl.Chip([]string{"a", "b"}, []string{"out"},
		[]hdl.Part{
			or(hdl.W{"a": "a", "b": "b", "out": "or"}),
			hdl.Nand(hdl.W{"a": "a", "b": "b", "out": "nand"}),
			hdl.Nand(hdl.W{"a": "or", "b": "nand", "out": "out"}),
		})

	td := []struct {
		name   string
		gate   hdl.NewPartFunc
		result []bool
	}{
		{"AND", and, []bool{false, false, false, true}},
		{"OR", or, []bool{false, true, true, true}},
		{"NOR", nor, []bool{true, false, false, false}},
		{"XOR", xor, []bool{false, true, true, false}},
		{"XNOR", xnor, []bool{true, false, false, true}},
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
// Clocks should be implemented as custom inputs. Really.
//
func Test_clock(t *testing.T) {
	var disable, tick bool

	check := func(v bool) {
		t.Helper()
		if tick != v {
			t.Errorf("expected %v, got %v", v, tick)
		}
	}
	c, err := hdl.NewCircuit([]hdl.Part{
		hdl.Input(hdl.W{"out": "disable"}, func() bool { return disable }),
		hdl.Nor(hdl.W{"a": "disable", "b": "out", "out": "out"}),
		hdl.Output(hdl.W{"in": "out"}, func(out bool) { tick = out }),
	})
	if err != nil {
		t.Fatal(err)
	}
	// we have two wires: "disable" and "out".
	// note that Output("out", ...) is delayed by one tick after the Nand updates it.

	disable = true
	c.Update(0)
	check(false)
	c.Update(0)
	// this is an expected signal change appearing in the first couple of ticks due to signal propagation delay
	check(true)
	c.Update(0)
	check(false)
	c.Update(0)
	check(false)

	disable = false
	c.Update(0)
	check(false)
	c.Update(0)
	check(false)
	c.Update(0)
	// the clock starts ticking now.
	check(true)
	c.Update(0)
	check(false)
	c.Update(0)
	check(true)
	disable = true
	c.Update(0)
	check(false)
	c.Update(0)
	check(true)
	c.Update(0)
	// the clock stops ticking now.
	check(false)
	c.Update(0)
	check(false)
}
