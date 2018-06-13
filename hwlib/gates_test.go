package hwlib_test

import (
	"strings"
	"testing"
	"testing/quick"

	"github.com/db47h/hwsim/hwtest"

	hw "github.com/db47h/hwsim"
	hl "github.com/db47h/hwsim/hwlib"
)

const testTPC = 8

func testGate(t *testing.T, name string, gate hw.NewPartFn, result [][]bool) {
	t.Helper()
	part := gate("").PartSpec // build dummy gate just to get to the partspec
	inputs := make([]bool, len(part.Inputs))
	outputs := make([]bool, len(part.Outputs))
	var w strings.Builder
	parts := make(hw.Parts, 0, len(part.Inputs)+len(part.Outputs)+1)
	for i, n := range part.Inputs {
		w.WriteByte(',')
		w.WriteString(n)
		w.WriteByte('=')
		w.WriteString(n)
		in := &inputs[i]
		parts = append(parts, hl.Input(func() bool { return *in })("out="+n))
	}
	for i, n := range part.Outputs {
		w.WriteByte(',')
		w.WriteString(n)
		w.WriteByte('=')
		w.WriteString(n)
		out := &outputs[i]
		parts = append(parts, hl.Output(func(v bool) { *out = v })("in="+n))
	}
	wr := w.String()
	// trim first ','
	if len(wr) > 0 {
		wr = wr[1:]
	}
	parts = append(parts, gate(wr))
	c, err := hw.NewCircuit(0, testTPC, parts)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Dispose()

	tot := 1 << uint(len(part.Inputs))
	for i := 0; i < tot; i++ {
		for bit := range inputs {
			inputs[len(inputs)-bit-1] = (i & (1 << uint(bit))) != 0
		}
		c.TickTock()
		for o, out := range outputs {
			exp := result[o][i]
			if exp != out {
				t.Errorf("%s %v = %v, got %v", part.Name, inputs, exp, out)
			}
		}
	}
}

func Test_gate_builtin(t *testing.T) {
	tr, err := hw.Chip("TRUE", hw.IO("a"), hw.IO("out"), hw.Parts{
		hl.And("a=true, b=true, out=out"),
	})
	if err != nil {
		t.Fatal(err)
	}
	fa, err := hw.Chip("FALSE", hw.IO("a"), hw.IO("out"), hw.Parts{
		hl.Or("a=false, b=false, out=out"),
	})
	if err != nil {
		t.Fatal(err)
	}
	td := []struct {
		name   string
		gate   hw.NewPartFn
		result [][]bool // a=0 && b=0, a=0 && b=1, a=1 && b=0, a=1 && b=1
	}{
		{"NOT", hl.Not, [][]bool{{true, false}}},
		{"AND", hl.And, [][]bool{{false, false, false, true}}},
		{"NAND", hl.Nand, [][]bool{{true, true, true, false}}},
		{"OR", hl.Or, [][]bool{{false, true, true, true}}},
		{"NOR", hl.Nor, [][]bool{{true, false, false, false}}},
		{"XOR", hl.Xor, [][]bool{{false, true, true, false}}},
		{"XNOR", hl.Xnor, [][]bool{{true, false, false, true}}},
		{"TRUE", tr, [][]bool{{true, true}}},
		{"FALSE", fa, [][]bool{{false, false}}},
		{"MUX", hl.Mux, [][]bool{{false, false, false, true, true, false, true, true}}},
		{"DMUX", hl.DMux, [][]bool{{false, false, true, false}, {false, false, false, true}}},
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
		hl.InputN(16, func() int64 { return in })("out[0..15]= t[0..15]"),
		hl.OutputN(16, func(n int64) { out = n })("in[0..15] = t[0..15]"),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Dispose()

	in = 0x80a2
	c.TickTock()
	if out != in {
		t.Fatalf("Expected %x, got %x", in, out)
	}
}

func Test_gateN_builtin(t *testing.T) {
	twoIn := "a[0..15]=a[0..15], b[0..15]=b[0..15], out[0..15]=out[0..15]"
	td := []struct {
		gate hw.Part
		ctrl func(a, b int16) int16
	}{
		{hl.And16(twoIn), func(a, b int16) int16 { return a & b }},
		{hl.Nand16(twoIn), func(a, b int16) int16 { return ^(a & b) }},
		{hl.Or16(twoIn), func(a, b int16) int16 { return a | b }},
		{hl.Nor16(twoIn), func(a, b int16) int16 { return ^(a | b) }},
		{hl.Not16("in[0..15]=a[0..15], out[0..15]=out[0..15]"), func(a, b int16) int16 { return ^a }},
	}

	_ = td

	for _, d := range td {
		t.Run(d.gate.Name, func(t *testing.T) {
			var a, b int16
			var out int16

			chip, err := hw.Chip(d.gate.Name+"wrapper", hw.IO("a[16], b[16]"), hw.IO("out[16]"), hw.Parts{
				d.gate,
			})
			if err != nil {
				t.Fatal(err)
			}

			c, err := hw.NewCircuit(0, testTPC, hw.Parts{
				hl.InputN(16, func() int64 { return int64(a) })("out[0..15]=a[0..15]"),
				hl.InputN(16, func() int64 { return int64(b) })("out[0..15]=b[0..15]"),
				chip(twoIn),
				hl.OutputN(16, func(v int64) { out = int16(v) })("in[0..15]=out[0..15]"),
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

func TestOrNWays(t *testing.T) {
	or4, err := hw.Chip("myOr4Way", hw.IO("in[4]"), hw.IO("out"), hw.Parts{
		hl.Or("a=in[0], b=in[1], out=o1"),
		hl.Or("a=in[2], b=in[3], out=o2"),
		hl.Or("a=o1, b=o2, out=out"),
	})
	if err != nil {
		t.Fatal(err)
	}
	hwtest.ComparePart(t, 4, hl.OrNWay(4), or4)
}

func TestAndNWays(t *testing.T) {
	and4, err := hw.Chip("myAnd4Way", hw.IO("in[4]"), hw.IO("out"), hw.Parts{
		hl.And("a=in[0], b=in[1], out=o1"),
		hl.And("a=in[2], b=in[3], out=o2"),
		hl.And("a=o1, b=o2, out=out"),
	})
	if err != nil {
		t.Fatal(err)
	}
	hwtest.ComparePart(t, 4, hl.AndNWay(4), and4)
}
