package hwsim_test

import (
	"math/rand"
	"testing"
	"time"

	hw "github.com/db47h/hwsim"
)

func TestDFF(t *testing.T) {
	var (
		in, out int64
	)

	dff4, err := hw.Chip("DFF4", hw.In{"in[4]"}, hw.Out{"out[4]"}, hw.Parts{
		hw.DFF(hw.W{"in": "in[0]", "out": "out[0]"}),
		hw.DFF(hw.W{"in": "in[1]", "out": "out[1]"}),
		hw.DFF(hw.W{"in": "in[2]", "out": "out[2]"}),
		hw.DFF(hw.W{"in": "in[3]", "out": "out[3]"}),
	})
	if err != nil {
		t.Fatal(err)
	}

	c, err := hw.NewCircuit(0, 4, hw.Parts{
		hw.Input16(func() int64 { return in })(hw.W{"out[0..3]": "in[0..3]"}),
		dff4(hw.W{"in[0..3]": "in[0..3]", "out[0..3]": "out[0..3]"}),
		hw.Output16(func(o int64) { out = o })(hw.W{"in[0..3]": "out[0..3]"}),
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
	reg, err := hw.Chip("BitReg", hw.In{"in", "load"}, hw.Out{"out"}, hw.Parts{
		hw.Mux(hw.W{"a": "out", "b": "in", "sel": "load", "out": "muxOut"}),
		hw.DFF(hw.W{"in": "muxOut", "out": "out"}),
	})

	if err != nil {
		t.Fatal(err)
	}

	var in, load, out bool

	c, err := hw.NewCircuit(0, 4, hw.Parts{
		hw.Input(func() bool { return in })(hw.W{"out": "dffI"}),
		hw.Input(func() bool { return load })(hw.W{"out": "dffLD"}),
		reg(hw.W{"in": "dffI", "load": "dffLD", "out": "dffO"}),
		hw.Output(func(b bool) { out = b })(hw.W{"in": "dffO"}),
	})

	if err != nil {
		t.Fatal(err)
	}

	rand.Seed(time.Now().UnixNano())

	p := in
	for i := 0; i < 1000; i++ {
		in = rand.Int63()&(1<<62) != 0
		load = rand.Int63()&(1<<62) != 0
		c.TickTock()
		if p != out {
			t.Fatal("p != out")
		}
		if load {
			p = in
		}
	}
}
