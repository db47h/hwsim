package hwsim_test

import (
	"testing"

	hw "github.com/db47h/hwsim"
	hl "github.com/db47h/hwsim/hwlib"
)

func TestDFF(t *testing.T) {
	var (
		in, out int64
	)

	dff4, err := hw.Chip("DFF4", "in[4],b[2]", "out4[5]", hw.Parts{
		hl.DFF("in=in[0], out=out4[0]"),
		hl.DFF("in=in[1], out=out4[1]"),
		hl.DFF("in=in[2], out=out4[2]"),
		hl.DFF("in=in[3], out=out4[3]"),
	})
	if err != nil {
		t.Fatal(err)
	}

	c, err := hw.NewCircuit(0, 4, hw.Parts{
		hl.InputN(16, func() int64 { return in })("out[0..3]=in[0..3]"),
		dff4("in[0..3]=in[0..3], out4[0..3]=out[0..3]"),
		hl.OutputN(16, func(o int64) { out = o })("in[0..3]=out[0..3]"),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Dispose()

	var prev int64
	for i := int64(15); i >= 0; i-- {
		// because inputs are delayed by one tick, DFFs do not see the new value
		// when we change it right at the beginning of a new clock cycle.
		// Additionally when a DFF is used as a part of another chip, that chip's
		// output should be read only at the next tick.

		// input i
		in = i

		c.TickTock()

		if prev != out {
			t.Fatalf("bad output for input %d: expected out = %d, got %d", prev, prev, out)
		}

		// here's the value that we should see at the end of the next cycle
		prev = i
	}
}

func Test_bit_register(t *testing.T) {
	reg, err := hw.Chip("BitReg", "in, load", "out", hw.Parts{
		hl.Mux("a=out, b=in, sel=load, out=muxOut"),
		hl.DFF("in=muxOut, out=out"),
	})

	if err != nil {
		t.Fatal(err)
	}

	var in, load, out bool

	c, err := hw.NewCircuit(0, 4, hw.Parts{
		hl.Input(func() bool { return in })("out=dffI"),
		hl.Input(func() bool { return load })("out=dffLD"),
		reg("in=dffI, load=dffLD, out=dffO"),
		hl.Output(func(b bool) { out = b })("in=dffO"),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Dispose()

	p := in
	for i := 0; i < 1000; i++ {
		in = randBool()
		load = randBool()
		c.TickTock()
		if p != out {
			t.Fatal("p != out")
		}
		if load {
			p = in
		}
	}
}
