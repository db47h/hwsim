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
	not, err := hdl.Chip([]string{"a", "b"}, []string{"out"}, []hdl.Part{
		hdl.Not(hdl.W{"in": "a", "out": "out"}),
		// ignore b
		hdl.Or(hdl.W{"a": "b", "out": hdl.GND}),
	})
	if err != nil {
		t.Fatal(err)
	}
	tr, err := hdl.Chip([]string{"a", "b"}, []string{"out"}, []hdl.Part{
		hdl.And(hdl.W{"a": hdl.True, "b": hdl.True, "out": "out"}),
		// ignore a & b
		hdl.Or(hdl.W{"a": "a", "b": "b", "out": hdl.GND}),
	})
	if err != nil {
		t.Fatal(err)
	}
	fa, err := hdl.Chip([]string{"a", "b"}, []string{"out"}, []hdl.Part{
		hdl.Or(hdl.W{"a": hdl.False, "b": hdl.False, "out": "out"}),
		// try to write 1 to the "false" pin
		hdl.Input(hdl.W{"out": hdl.GND}, func() bool { return false }),
		// ignore a & b
		hdl.Or(hdl.W{"a": "a", "b": "b", "out": hdl.GND}),
	})
	if err != nil {
		t.Fatal(err)
	}
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
		{"TRUE", tr, []bool{true, true, true, true}},
		{"FALSE", fa, []bool{false, false, false, false}},
	}
	for _, d := range td {
		t.Run(d.name, func(t *testing.T) {
			testGate(t, d.name, d.gate, d.result)
		})
	}
}

func Test_gate_custom(t *testing.T) {
	and, err := hdl.Chip([]string{"a", "b"}, []string{"out"},
		[]hdl.Part{
			hdl.Nand(hdl.W{"a": "a", "b": "b", "out": "nand"}),
			hdl.Nand(hdl.W{"a": "nand", "b": "nand", "out": "out"}),
		})
	if err != nil {
		t.Fatal(err)
	}
	or, err := hdl.Chip([]string{"a", "b"}, []string{"out"},
		[]hdl.Part{
			hdl.Nand(hdl.W{"a": "a", "b": "a", "out": "notA"}),
			hdl.Nand(hdl.W{"a": "b", "b": "b", "out": "notB"}),
			hdl.Nand(hdl.W{"a": "notA", "b": "notB", "out": "out"}),
		})
	if err != nil {
		t.Fatal(err)
	}
	nor, err := hdl.Chip([]string{"a", "b"}, []string{"out"},
		[]hdl.Part{
			or(hdl.W{"a": "a", "b": "b", "out": "orAB"}),
			hdl.Nand(hdl.W{"a": "orAB", "b": "orAB", "out": "out"}),
		})
	if err != nil {
		t.Fatal(err)
	}
	xor, err := hdl.Chip([]string{"a", "b"}, []string{"out"},
		[]hdl.Part{
			hdl.Nand(hdl.W{"a": "a", "b": "b", "out": "nandAB"}),
			hdl.Nand(hdl.W{"a": "a", "b": "nandAB", "out": "w0"}),
			hdl.Nand(hdl.W{"a": "b", "b": "nandAB", "out": "w1"}),
			hdl.Nand(hdl.W{"a": "w0", "b": "w1", "out": "out"}),
		})
	if err != nil {
		t.Fatal(err)
	}
	xnor, err := hdl.Chip([]string{"a", "b"}, []string{"out"},
		[]hdl.Part{
			or(hdl.W{"a": "a", "b": "b", "out": "or"}),
			hdl.Nand(hdl.W{"a": "a", "b": "b", "out": "nand"}),
			hdl.Nand(hdl.W{"a": "or", "b": "nand", "out": "out"}),
		})
	if err != nil {
		t.Fatal(err)
	}
	not, err := hdl.Chip([]string{"a", "b"}, []string{"out"},
		[]hdl.Part{
			hdl.Nand(hdl.W{"a": "a", "b": "a", "out": "out"}),
			// ignore b
			hdl.Or(hdl.W{"b": "b", "out": hdl.GND}),
		})
	if err != nil {
		t.Fatal(err)
	}

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
		{"NOT", not, []bool{true, true, false, false}},
	}
	for _, d := range td {
		t.Run(d.name, func(t *testing.T) {
			testGate(t, d.name, d.gate, d.result)
		})
	}
}

func TestW_Check(t *testing.T) {
	cmp := func(w1, w2 hdl.W) bool {
		if len(w1) != len(w2) {
			return false
		}
		for k, v := range w1 {
			if t, ok := w2[k]; !ok || t != v {
				return false
			}
		}
		return true
	}
	data := []struct {
		name string
		w    hdl.W
		in   []string
		out  []string
		ret  hdl.W
		err  string
	}{
		{"AllWired", hdl.W{"a": "x", "b": "y", "out": "z"}, []string{"a", "b"}, []string{"out"}, hdl.W{"a": "x", "b": "y", "out": "z"}, ""},
		{"UnwiredB", hdl.W{"a": "x", "out": "z"}, []string{"a", "b"}, []string{"out"}, hdl.W{"a": "x", "b": hdl.False, "out": "z"}, ""},
		{"ExtraPin", hdl.W{"a": "x", "b": "y", "out": "z"}, []string{"a", "b"}, nil, nil, "unknown pin \"out\""},
		{"nil", nil, []string{"in"}, nil, hdl.W{"in": hdl.False}, ""},
		{"nilnil", nil, nil, nil, nil, ""},
	}
	for _, d := range data {
		t.Run(d.name, func(t *testing.T) {
			n, err := d.w.Wire(d.in, d.out)
			if err == nil && d.err != "" || err != nil && err.Error() != d.err {
				t.Errorf("Got error %q, expected %q", d.err, err)
				return
			}
			if !cmp(n, d.ret) {
				t.Errorf("Got %v, expected %v", n, d.ret)
			}
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
	// but we wrap it inot a stand-alone chip in order to add a layer complexity
	// for testing purposes.
	clk, err := hdl.Chip([]string{"disable"}, []string{"tick"}, []hdl.Part{
		hdl.Nor(hdl.W{"a": "disable", "b": "tick", "out": "tick"}),
	})
	if err != nil {
		t.Fatal(err)
	}
	c, err := hdl.NewCircuit([]hdl.Part{
		hdl.Input(hdl.W{"out": "disable"}, func() bool { return disable }),
		clk(hdl.W{"disable": "disable", "tick": "out"}),
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
