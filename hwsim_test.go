package hwsim_test

import (
	"testing"

	hw "github.com/db47h/hwsim"
	"github.com/pkg/errors"
)

const testTPC = 16

func trace(t *testing.T, err error) {
	t.Helper()
	if err, ok := err.(interface {
		StackTrace() errors.StackTrace
	}); ok {
		for _, f := range err.StackTrace() {
			t.Logf("%+v ", f)
		}
	}
}

// Test a basic clock with a Nor gate.
//
// The purpose of this test is to catch changes in propagation delays
// from Inputs and Outputs as well as testing loops between input and outputs.
//
// Don't do this in your own circuits! Clocks should be implemented as custom
// components or inputs. Or use a DFF.
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
	clk, err := hw.Chip("CLK", hw.In("disable"), hw.Out("tick"), hw.Parts{
		hw.Nor("a=disable, b=tick, out=tick"),
	})
	if err != nil {
		t.Fatal(err)
	}
	c, err := hw.NewCircuit(0, testTPC, hw.Parts{
		hw.Input(func() bool { return disable })("out=disable"),
		clk("disable=disable, tick=out"),
		hw.Output(func(out bool) { tick = out })("in=out"),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Dispose()

	// we have two wires: "disable" and "out".
	// note that Output("out", ...) is delayed by one tick after the Nand updates it.

	disable = true
	c.Step()
	check(false)
	c.Step()
	// this is an expected signal change appearing in the first couple of ticks due to signal propagation delay
	check(true)
	c.Step()
	check(false)
	c.Step()
	check(false)

	disable = false
	c.Step()
	check(false)
	c.Step()
	check(false)
	c.Step()
	// the clock starts ticking now.
	check(true)
	c.Step()
	check(false)
	c.Step()
	check(true)
	disable = true
	c.Step()
	check(false)
	c.Step()
	check(true)
	c.Step()
	// the clock stops ticking now.
	check(false)
	c.Step()
	check(false)
}
