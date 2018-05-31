package hdl_test

import (
	"testing"

	"github.com/db47h/hdl"
)

func testGate(t *testing.T, name string, gate hdl.NewPartFunc, result [][]bool) {
	t.Helper()
	part := gate(nil) // dummy gate
	inputs := make([]bool, len(part.Spec().In))
	outputs := make([]bool, len(part.Spec().Out))
	w := make(hdl.W)
	parts := make([]hdl.Part, 0, len(part.Spec().In)+len(part.Spec().Out)+1)
	for i, n := range part.Spec().In {
		w[n] = n
		in := &inputs[i]
		parts = append(parts, hdl.Input(hdl.W{"out": n}, func() bool { return *in }))
	}
	for i, n := range part.Spec().Out {
		w[n] = n
		out := &outputs[i]
		parts = append(parts, hdl.Output(hdl.W{"in": n}, func(v bool) { *out = v }))
	}
	parts = append(parts, gate(w))
	c, err := hdl.NewCircuit(parts)
	if err != nil {
		t.Fatal(err)
	}

	tot := 1 << uint(len(part.Spec().In))
	// t.Log(tot)
	// for _, p := range parts {
	// 	t.Log(p.Spec().Name, " ", p.Wires())
	// }
	for i := 0; i < tot; i++ {
		for bit := range inputs {
			inputs[len(inputs)-bit-1] = (i & (1 << uint(bit))) != 0
		}
		for u := 0; u < 10; u++ {
			c.Update(workers)
		}
		for o, out := range outputs {
			exp := result[o][i]
			if exp != out {
				t.Errorf("%s %v = %v, got %v", part.Spec().Name, inputs, exp, out)
			}
		}
	}
}

func Test_gate_builtin(t *testing.T) {
	tr, err := hdl.Chip("TRUE", []string{"a"}, []string{"out"}, []hdl.Part{
		hdl.And(hdl.W{"a": hdl.True, "b": hdl.True, "out": "out"}),
	})
	if err != nil {
		t.Fatal(err)
	}
	fa, err := hdl.Chip("FALSE", []string{"a"}, []string{"out"}, []hdl.Part{
		hdl.Or(hdl.W{"a": hdl.False, "b": hdl.False, "out": "out"}),
	})
	if err != nil {
		t.Fatal(err)
	}
	td := []struct {
		name   string
		gate   hdl.NewPartFunc
		result [][]bool // a=0 && b=0, a=0 && b=1, a=1 && b=0, a=1 && b=1
	}{
		{"NOT", hdl.Not, [][]bool{{true, false}}},
		{"AND", hdl.And, [][]bool{{false, false, false, true}}},
		{"NAND", hdl.Nand, [][]bool{{true, true, true, false}}},
		{"OR", hdl.Or, [][]bool{{false, true, true, true}}},
		{"NOR", hdl.Nor, [][]bool{{true, false, false, false}}},
		{"XOR", hdl.Xor, [][]bool{{false, true, true, false}}},
		{"XNOR", hdl.Xnor, [][]bool{{true, false, false, true}}},
		{"TRUE", tr, [][]bool{{true, true}}},
		{"FALSE", fa, [][]bool{{false, false}}},
		{"MUX", hdl.Mux, [][]bool{{false, false, false, true, true, false, true, true}}},
		{"DMUX", hdl.DMux, [][]bool{{false, false, true, false}, {false, false, false, true}}},
	}
	for _, d := range td {
		t.Run(d.name, func(t *testing.T) {
			testGate(t, d.name, d.gate, d.result)
		})
	}
}

func TestInput16(t *testing.T) {
	in := int64(0)
	out := int64(0)
	c, err := hdl.NewCircuit([]hdl.Part{
		hdl.Input16(hdl.W{"out[0..15]": "t[0..15]"}, func() int64 { return in }),
		hdl.Output16(hdl.W{"in[0..15]": "t[0..15]"}, func(n int64) { out = n }),
	})
	if err != nil {
		panic(err)
	}
	in = 0x80a2
	for i := 0; i < 2; i++ {
		c.Update(workers)
	}
	if out != in {
		t.Fatalf("Expected %x, got %x", in, out)
	}
}
