package hwsim_test

import (
	"testing"

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
