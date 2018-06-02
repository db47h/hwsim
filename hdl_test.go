package hdl_test

import (
	"runtime"
	"testing"

	"github.com/db47h/hdl"
)

var workers = runtime.NumCPU()

func Test_gate_custom(t *testing.T) {
	and, err := hdl.Chip("AND", []string{"a", "b"}, []string{"out"},
		[]hdl.Part{
			hdl.Nand(hdl.W{"a": "a", "b": "b", "out": "nand"}),
			hdl.Nand(hdl.W{"a": "nand", "b": "nand", "out": "out"}),
		})
	if err != nil {
		t.Fatal(err)
	}
	or, err := hdl.Chip("OR", []string{"a", "b"}, []string{"out"},
		[]hdl.Part{
			hdl.Nand(hdl.W{"a": "a", "b": "a", "out": "notA"}),
			hdl.Nand(hdl.W{"a": "b", "b": "b", "out": "notB"}),
			hdl.Nand(hdl.W{"a": "notA", "b": "notB", "out": "out"}),
		})
	if err != nil {
		t.Fatal(err)
	}
	nor, err := hdl.Chip("NOR", []string{"a", "b"}, []string{"out"},
		[]hdl.Part{
			or(hdl.W{"a": "a", "b": "b", "out": "orAB"}),
			hdl.Nand(hdl.W{"a": "orAB", "b": "orAB", "out": "out"}),
		})
	if err != nil {
		t.Fatal(err)
	}
	xor, err := hdl.Chip("XOR", []string{"a", "b"}, []string{"out"},
		[]hdl.Part{
			hdl.Nand(hdl.W{"a": "a", "b": "b", "out": "nandAB"}),
			hdl.Nand(hdl.W{"a": "a", "b": "nandAB", "out": "w0"}),
			hdl.Nand(hdl.W{"a": "b", "b": "nandAB", "out": "w1"}),
			hdl.Nand(hdl.W{"a": "w0", "b": "w1", "out": "out"}),
		})
	if err != nil {
		t.Fatal(err)
	}
	xnor, err := hdl.Chip("XNOR", []string{"a", "b"}, []string{"out"},
		[]hdl.Part{
			or(hdl.W{"a": "a", "b": "b", "out": "or"}),
			hdl.Nand(hdl.W{"a": "a", "b": "b", "out": "nand"}),
			hdl.Nand(hdl.W{"a": "or", "b": "nand", "out": "out"}),
		})
	if err != nil {
		t.Fatal(err)
	}
	not, err := hdl.Chip("NOT", []string{"a"}, []string{"out"},
		[]hdl.Part{
			hdl.Nand(hdl.W{"a": "a", "b": "a", "out": "out"}),
		})
	if err != nil {
		t.Fatal(err)
	}
	mux, err := hdl.Chip("NUX", []string{"a", "b", "sel"}, []string{"out"}, []hdl.Part{
		hdl.Not(hdl.W{"in": "sel", "out": "notSel"}),
		hdl.And(hdl.W{"a": "a", "b": "notSel", "out": "w0"}),
		hdl.And(hdl.W{"a": "b", "b": "sel", "out": "w1"}),
		hdl.Or(hdl.W{"a": "w0", "b": "w1", "out": "out"}),
	})
	if err != nil {
		t.Fatal(err)
	}
	dmux, err := hdl.Chip("DMUX", []string{"in", "sel"}, []string{"a", "b"}, []hdl.Part{
		hdl.Not(hdl.W{"in": "sel", "out": "notSel"}),
		hdl.And(hdl.W{"a": "in", "b": "notSel", "out": "a"}),
		hdl.And(hdl.W{"a": "in", "b": "sel", "out": "b"}),
	})
	if err != nil {
		t.Fatal(err)
	}
	td := []struct {
		name   string
		gate   hdl.NewPartFunc
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

// func TestW_Wire(t *testing.T) {
// 	cmp := func(w1, w2 hdl.W) bool {
// 		if len(w1) != len(w2) {
// 			return false
// 		}
// 		for k, v := range w1 {
// 			if t, ok := w2[k]; !ok || t != v {
// 				return false
// 			}
// 		}
// 		return true
// 	}
// 	data := []struct {
// 		name string
// 		w    hdl.W
// 		in   []string
// 		out  []string
// 		ret  hdl.W
// 		err  string
// 	}{
// 		{"AllWired", hdl.W{"a": "x", "b": "y", "out": "z"}, []string{"a", "b"}, []string{"out"}, hdl.W{"a": "x", "b": "y", "out": "z"}, ""},
// 		{"UnwiredB", hdl.W{"a": "x", "out": "z"}, []string{"a", "b"}, []string{"out"}, hdl.W{"a": "x", "b": hdl.False, "out": "z"}, ""},
// 		{"ExtraPin", hdl.W{"a": "x", "b": "y", "out": "z"}, []string{"a", "b"}, nil, nil, "unknown pin \"out\""},
// 		{"nil", nil, []string{"in"}, nil, hdl.W{"in": hdl.False}, ""},
// 		{"nilnil", nil, nil, nil, nil, ""},
// 	}
// 	for _, d := range data {
// 		t.Run(d.name, func(t *testing.T) {
// 			n, err := d.w.Wire(d.in, d.out)
// 			if err == nil && d.err != "" || err != nil && err.Error() != d.err {
// 				t.Errorf("Got error %q, expected %q", err, d.err)
// 				return
// 			}
// 			if !cmp(n, d.ret) {
// 				t.Errorf("Got %v, expected %v", n, d.ret)
// 			}
// 		})
// 	}
// }

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
	clk, err := hdl.Chip("CLK", []string{"disable"}, []string{"tick"}, []hdl.Part{
		hdl.Nor(hdl.W{"a": "disable", "b": "tick", "out": "tick"}),
	})
	if err != nil {
		t.Fatal(err)
	}
	c, err := hdl.NewCircuit([]hdl.Part{
		hdl.Input(func() bool { return disable })(hdl.W{"out": "disable"}),
		clk(hdl.W{"disable": "disable", "tick": "out"}),
		hdl.Output(func(out bool) { tick = out })(hdl.W{"in": "out"}),
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

func Test_chip_errors(t *testing.T) {
	data := []struct {
		name  string
		in    []string
		out   []string
		parts []hdl.Part
		err   string
	}{
		{"true_out", []string{"a", "b"}, []string{"out"}, []hdl.Part{
			hdl.Nand(hdl.W{"a": "a", "b": "b", "out": hdl.True}),
			hdl.Nand(hdl.W{"a": "a", "b": "b", "out": "out"}),
		}, "NAND.out:true: output pin connected to constant \"true\" input"},
		{"multi_out", []string{"a", "b"}, []string{"out"}, []hdl.Part{
			hdl.Nand(hdl.W{"a": "a", "b": "b", "out": "a"}),
			hdl.Nand(hdl.W{"a": "a", "b": "b", "out": "out"}),
		}, "NAND.out:a: output pin already used as output or is one of the chip's input pin"},
		{"no_output", []string{"a", "b"}, []string{"out"}, []hdl.Part{
			hdl.Nand(hdl.W{"a": "a", "b": "wx", "out": "out"}),
		}, "pin wx not connected to any output"},
		{"no_input", []string{"a", "b"}, []string{"out"}, []hdl.Part{
			hdl.Nand(hdl.W{"a": "a", "b": "b", "out": "foo"}),
			hdl.Nand(hdl.W{"a": "a", "b": "b", "out": "out"}),
		}, "pin foo not connected to any input"},
		{"unconnected_in", []string{"a", "b"}, []string{"out"}, []hdl.Part{}, ""},
	}
	for _, d := range data {
		t.Run(d.name, func(t *testing.T) {
			_, err := hdl.Chip(d.name, d.in, d.out, d.parts)
			if err == nil && d.err != "" || err != nil && err.Error() != d.err {
				t.Errorf("Got error %q, expected %q", err, d.err)
				return
			}
		})
	}

}
