package hdl_test

import (
	"runtime"
	"testing"

	"github.com/db47h/hdl"
)

var workers = runtime.NumCPU() // this is naive but should be good enough for testing.

func testGate(t *testing.T, name string, gate hdl.Part, result []bool) {
	var a, b, out bool
	t.Helper()
	c, err := hdl.NewCircuit([]hdl.Part{
		hdl.Input("a", func() bool { return a }),
		hdl.Input("b", func() bool { return b }),
		hdl.Output("out", func(o bool) { out = o }),
		gate,
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
	td := []struct {
		name   string
		gate   hdl.Part
		result []bool // a=0 && b=0, a=0 && b=1, a=1 && b=0, a=1 && b=1
	}{
		{"AND", hdl.And("a", "b", "out"), []bool{false, false, false, true}},
		{"NAND", hdl.Nand("a", "b", "out"), []bool{true, true, true, false}},
		{"OR", hdl.Or("a", "b", "out"), []bool{false, true, true, true}},
		{"NOR", hdl.Nor("a", "b", "out"), []bool{true, false, false, false}},
		{"XOR", hdl.Xor("a", "b", "out"), []bool{false, true, true, false}},
		{"XNOR", hdl.Xnor("a", "b", "out"), []bool{true, false, false, true}},
		{"NOTa", hdl.Not("a", "out"), []bool{true, true, false, false}},
		{"NOTb", hdl.Not("b", "out"), []bool{true, false, true, false}},
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
			hdl.Nand("a", "b", "nand"),
			hdl.Nand("nand", "nand", "out"),
		})
	or := hdl.Chip([]string{"a", "b"}, []string{"out"},
		[]hdl.Part{
			hdl.Nand("a", "a", "notA"),
			hdl.Nand("b", "b", "notB"),
			hdl.Nand("notA", "notB", "out"),
		})
	nor := hdl.Chip([]string{"a", "b"}, []string{"out"},
		[]hdl.Part{
			or("a", "b", "orAB"),
			hdl.Nand("orAB", "orAB", "out"),
		})
	xor := hdl.Chip([]string{"a", "b"}, []string{"out"},
		[]hdl.Part{
			hdl.Nand("a", "b", "nandAB"),
			hdl.Nand("a", "nandAB", "w0"),
			hdl.Nand("b", "nandAB", "w1"),
			hdl.Nand("w0", "w1", "out"),
		})
	xnor := hdl.Chip([]string{"a", "b"}, []string{"out"},
		[]hdl.Part{
			or("a", "b", "or"),
			hdl.Nand("a", "b", "nand"),
			hdl.Nand("or", "nand", "out"),
		})

	td := []struct {
		name   string
		gate   hdl.Part
		result []bool
	}{
		{"AND", and("a", "b", "out"), []bool{false, false, false, true}},
		{"OR", or("a", "b", "out"), []bool{false, true, true, true}},
		{"NOR", nor("a", "b", "out"), []bool{true, false, false, false}},
		{"XOR", xor("a", "b", "out"), []bool{false, true, true, false}},
		{"XNOR", xnor("a", "b", "out"), []bool{true, false, false, true}},
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
		hdl.Input("disable", func() bool { return disable }),
		hdl.Nor("disable", "out", "out"),
		hdl.Output("out", func(out bool) { tick = out }),
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
