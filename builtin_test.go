package hwsim_test

import (
	"testing"
	"testing/quick"

	hw "github.com/db47h/hwsim"
)

func testGate(t *testing.T, name string, gate hw.NewPartFn, result [][]bool) {
	t.Helper()
	part := gate(nil) // dummy gate
	inputs := make([]bool, len(part.Spec().In))
	outputs := make([]bool, len(part.Spec().Out))
	w := make(hw.Wiring)
	parts := make(hw.Parts, 0, len(part.Spec().In)+len(part.Spec().Out)+1)
	for i, n := range part.Spec().In {
		w[n] = n
		in := &inputs[i]
		parts = append(parts, hw.Input(func() bool { return *in })(hw.Wiring{"out": n}))
	}
	for i, n := range part.Spec().Out {
		w[n] = n
		out := &outputs[i]
		parts = append(parts, hw.Output(func(v bool) { *out = v })(hw.Wiring{"in": n}))
	}
	parts = append(parts, gate(w))
	c, err := hw.NewCircuit(0, testTPC, parts)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Dispose()

	tot := 1 << uint(len(part.Spec().In))
	// t.Log(tot)
	// for _, p := range parts {
	// 	t.Log(p.Spec().Name, " ", p.Wires())
	// }
	for i := 0; i < tot; i++ {
		for bit := range inputs {
			inputs[len(inputs)-bit-1] = (i & (1 << uint(bit))) != 0
		}
		c.TickTock()
		for o, out := range outputs {
			exp := result[o][i]
			if exp != out {
				t.Errorf("%s %v = %v, got %v", part.Spec().Name, inputs, exp, out)
			}
		}
	}
}

func Test_gate_builtin(t *testing.T) {
	tr, err := hw.Chip("TRUE", hw.In{"a"}, hw.Out{"out"}, hw.Parts{
		hw.And(hw.W("a=true, b=true, out=out")),
	})
	if err != nil {
		trace(t, err)
		t.Fatal(err)
	}
	fa, err := hw.Chip("FALSE", hw.In{"a"}, hw.Out{"out"}, hw.Parts{
		hw.Or(hw.W("a=false, b=false, out=out")),
	})
	if err != nil {
		trace(t, err)
		t.Fatal(err)
	}
	td := []struct {
		name   string
		gate   hw.NewPartFn
		result [][]bool // a=0 && b=0, a=0 && b=1, a=1 && b=0, a=1 && b=1
	}{
		{"NOT", hw.Not, [][]bool{{true, false}}},
		{"AND", hw.And, [][]bool{{false, false, false, true}}},
		{"NAND", hw.Nand, [][]bool{{true, true, true, false}}},
		{"OR", hw.Or, [][]bool{{false, true, true, true}}},
		{"NOR", hw.Nor, [][]bool{{true, false, false, false}}},
		{"XOR", hw.Xor, [][]bool{{false, true, true, false}}},
		{"XNOR", hw.Xnor, [][]bool{{true, false, false, true}}},
		{"TRUE", tr, [][]bool{{true, true}}},
		{"FALSE", fa, [][]bool{{false, false}}},
		{"MUX", hw.Mux, [][]bool{{false, false, false, true, true, false, true, true}}},
		{"DMUX", hw.DMux, [][]bool{{false, false, true, false}, {false, false, false, true}}},
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
	c, err := hw.NewCircuit(0, testTPC, hw.Parts{
		hw.Input16(func() int64 { return in })(hw.W("out[0..15]= t[0..15]")),
		hw.Output16(func(n int64) { out = n })(hw.W("in[0..15] = t[0..15]")),
	})
	if err != nil {
		panic(err)
	}
	defer c.Dispose()

	in = 0x80a2
	c.TickTock()
	if out != in {
		t.Fatalf("Expected %x, got %x", in, out)
	}
}

func Test_gateN_builtin(t *testing.T) {
	twoIn := hw.W("a[0..15]=a[0..15], b[0..15]=b[0..15], out[0..15]=out[0..15]")
	td := []struct {
		gate hw.PartWiring
		ctrl func(a, b int16) int16
	}{
		{hw.And16(twoIn), func(a, b int16) int16 { return a & b }},
		{hw.Nand16(twoIn), func(a, b int16) int16 { return ^(a & b) }},
		{hw.Or16(twoIn), func(a, b int16) int16 { return a | b }},
		{hw.Nor16(twoIn), func(a, b int16) int16 { return ^(a | b) }},
		{hw.Not16(hw.W("in[0..15]=a[0..15], out[0..15]=out[0..15]")), func(a, b int16) int16 { return ^a }},
	}

	_ = td

	for _, d := range td {
		t.Run(d.gate.Spec().Name, func(t *testing.T) {
			var a, b int16
			var out int16

			chip, err := hw.Chip(d.gate.Spec().Name+"wrapper", hw.In{"a[16]", "b[16]"}, hw.Out{"out[16]"}, hw.Parts{
				d.gate,
			})
			if err != nil {
				t.Fatal(err)
			}

			c, err := hw.NewCircuit(0, testTPC, hw.Parts{
				hw.Input16(func() int64 { return int64(a) })(hw.W("out[0..15]=a[0..15]")),
				hw.Input16(func() int64 { return int64(b) })(hw.W("out[0..15]=b[0..15]")),
				chip(twoIn),
				hw.Output16(func(v int64) { out = int16(v) })(hw.W("in[0..15]=out[0..15]")),
			})
			if err != nil {
				t.Fatal(err)
			}
			defer c.Dispose()

			f := func(x, y int16) bool {
				a, b = x, y
				c.TickTock()
				return out == d.ctrl(x, y)
			}
			if err = quick.Check(f, nil); err != nil {
				t.Fatal(err)
			}
		})
	}
}
