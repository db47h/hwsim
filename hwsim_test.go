package hwsim_test

import (
	"math/rand"
	"runtime"
	"strings"
	"testing"
	"time"

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

func randBool() bool {
	return rand.Int63()&(1<<62) != 0
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

// This bench is here to becnhmark the workers sync mechanism overhead.
func BenchmarkCircuit_Step(b *testing.B) {
	workers := runtime.GOMAXPROCS(-1)
	parts := make(hw.Parts, 0, workers)
	for i := 0; i < workers; i++ {
		parts = append(parts, hw.Not(""))
	}
	c, err := hw.NewCircuit(workers, testTPC, parts)
	if err != nil {
		b.Fatal(err)
	}
	for i := 0; i < b.N; i++ {
		c.Step()
	}
}

func connString(in, out []string) string {
	var b strings.Builder
	for _, n := range in {
		if b.Len() > 0 {
			b.WriteRune(',')
		}
		b.WriteString(n)
		b.WriteRune('=')
		b.WriteString(n)
	}
	for _, n := range out {
		if b.Len() > 0 {
			b.WriteRune(',')
		}
		b.WriteString(n)
		b.WriteRune('=')
		b.WriteString(n)
	}
	return b.String()
}

func testPart(t *testing.T, tpc uint, part1 hw.NewPartFn, part2 hw.NewPartFn) {
	t.Helper()

	seed := time.Now().UnixNano() & 0xFFFFFFFF
	rand.Seed(seed)

	ps1, ps2 := part1(""), part1("")
	conns := connString(ps1.Inputs, ps1.Outputs)
	ps1, ps2 = part1(conns), part2(conns)

	inputs := make([]bool, len(ps1.Inputs))
	outputs := make([][2]bool, len(ps1.Outputs))

	// build two wrappers with their own set of outputs
	parts1 := hw.Parts{ps1}
	for i, o := range ps1.Outputs {
		n := i
		parts1 = append(parts1, hw.Output(func(b bool) { outputs[n][0] = b })("in="+o))
	}

	parts2 := hw.Parts{ps2}
	for i, o := range ps2.Outputs {
		n := i
		parts2 = append(parts2, hw.Output(func(b bool) { outputs[n][1] = b })("in="+o))
	}

	w1, err := hw.Chip("wrapper1", ps1.Inputs, nil, parts1)
	if err != nil {
		t.Fatal(err)
	}
	w2, err := hw.Chip("wrapper2", ps2.Inputs, nil, parts2)
	if err != nil {
		t.Fatal(err)
	}

	// compare specs
	if len(ps1.Inputs) != len(ps2.Inputs) {
		t.Fatal("len(ps1.Inputs) != len(ps2.Inputs)")
	}
	if len(ps1.Outputs) != len(ps2.Outputs) {
		t.Fatal("len(ps1.Outputs) != len(ps2.Outputs)")
	}
	for i := range ps1.Inputs {
		if ps1.Inputs[i] != ps2.Inputs[i] {
			t.Fatalf("ps1.Inputs[i] = %q != ps2.Inputs[i] = %q", ps1.Inputs[i], ps2.Inputs[i])
		}
	}
	for i := range ps1.Outputs {
		if ps1.Outputs[i] != ps2.Outputs[i] {
			t.Fatalf("ps1.Outputs[i] = %q != ps2.Outputs[i] = %q", ps1.Outputs[i], ps2.Outputs[i])
		}
	}

	var parts hw.Parts
	for i, n := range ps1.Inputs {
		k := i
		parts = append(parts, hw.Input(func() bool { return inputs[k] })("out="+n))
	}
	cstr := connString(ps1.Inputs, nil)
	parts = append(parts, w1(cstr), w2(cstr))

	c, err := hw.NewCircuit(0, tpc, parts)
	if err != nil {
		t.Fatal(err)
	}

	// random testing. Plan to add a callback to set inputs
	iter := len(ps1.Inputs)
	if iter > 10 {
		iter = 10
	}
	c.Tick()
	iter = 1 << uint(iter)
	for i := 0; i < iter; i++ {
		for in := range inputs {
			inputs[in] = randBool()
		}
		c.Tock()
		c.Tick()
		for o, out := range outputs {
			if out[0] != out[1] {
				var b strings.Builder
				for i, n := range ps1.Inputs {
					if b.Len() > 0 {
						b.WriteString(", ")
					}
					b.WriteString(n)
					b.WriteRune('=')
					if inputs[i] {
						b.WriteString("true")
					} else {
						b.WriteString("false")
					}
				}
				t.Fatalf("\nExpected %s => %s=%v\nGot %v", b.String(), ps1.Outputs[o], out[0], out[1])
			}
		}
	}
}

func Test_testPart(t *testing.T) {
	or, err := hw.Chip("custom_or", hw.In("a,b"), hw.Out("out"), hw.Parts{
		hw.Nand("a=a, b=a, out=notA"),
		hw.Nand("a=b, b=b, out=notB"),
		hw.Nand("a=notA, b=notB, out=out"),
	})
	if err != nil {
		t.Fatal(err)
	}
	testPart(t, 4, hw.Or, or)
}
