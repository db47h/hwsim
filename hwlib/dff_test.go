package hwlib_test

import (
	"math/rand"
	"testing"

	hw "github.com/db47h/hwsim"
	hl "github.com/db47h/hwsim/hwlib"
	"github.com/db47h/hwsim/hwtest"
)

func randBool() bool {
	return rand.Int63()&(1<<62) != 0
}

func TestDFF(t *testing.T) {
	var (
		in, out int64
	)

	dff4, err := hw.Chip("DFF4", "in[4]", "out[4]",
		hl.DFF("in=in[0], out=out[0]"),
		hl.DFF("in=in[1], out=out[1]"),
		hl.DFF("in=in[2], out=out[2]"),
		hl.DFF("in=in[3], out=out[3]"),
	)
	if err != nil {
		t.Fatal(err)
	}

	c, err := hw.NewCircuit(
		hw.InputN(16, func() int64 { return in })("out[0..3]=in[0..3]"),
		dff4("in[0..3]=in[0..3], out[0..3]=out[0..3]"),
		hw.OutputN(16, func(o int64) { out = o })("in[0..3]=out[0..3]"),
	)
	if err != nil {
		t.Fatal(err)
	}

	prev := int64(0)
	for i := int64(15); i >= 0; i-- {
		// input i
		in = i
		c.Tick()
		if prev != out {
			t.Fatalf("bad output for input %d after tick: expected out = %d, got %d", in, prev, out)
		}
		// change input
		in = 0
		c.Tock()
		if i != out {
			t.Fatalf("bad output for input %d after tock: expected out = %d, got %d", in, i, out)
		}
		prev = i
	}

	hwtest.ComparePart(t, hl.DFFN(4), dff4)
}

func Test_bit_register(t *testing.T) {
	reg, err := hw.Chip("BitReg", "in, load", "out",
		hl.Mux("a=out, b=in, sel=load, out=muxOut"),
		hl.DFF("in=muxOut, out=out"),
	)

	if err != nil {
		t.Fatal(err)
	}

	var in, load, out bool

	var c *hw.Circuit
	c, err = hw.NewCircuit(
		hw.Input(func() bool { return in })("out=dffI"),
		hw.Input(func() bool { return load })("out=dffLD"),
		reg("in=dffI, load=dffLD, out=dffO"),
		hw.Output(func(b bool) { out = b })("in=dffO"),
	)
	if err != nil {
		t.Fatal(err)
	}

	p := in
	for i := 0; i < 10; i++ {
		in = randBool()
		load = randBool()
		c.Tick()
		if p != out {
			t.Fatal("p != out")
		}
		c.Tock()
		if load {
			p = in
		}
		if load && in != out {
			t.Fatal("in != out")
		}
	}
}
