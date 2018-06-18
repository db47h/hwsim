package hwsim_test

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/db47h/hwsim"
	"github.com/db47h/hwsim/hwtest"
)

const testTPC = 16

func randBool() bool {
	return rand.Int63()&(1<<62) != 0
}

// Test a basic clock with a Nand gate.
//
// The purpose of this test is to catch changes in propagation delays
// from Inputs and Outputs as well as testing loops between input and outputs.
//
// Don't do this in your own circuits! Clocks should be implemented as custom
// components or inputs. Or use a DFF.
//
// func Test_clock(t *testing.T) {
// 	var enable, tick bool

// 	check := func(v bool) {
// 		t.Helper()
// 		if tick != v {
// 			t.Errorf("expected %v, got %v", v, tick)
// 		}
// 	}
// 	// we could implement the clock directly as a Nor in the cisrcuit (with no less gate delays)
// 	// but we wrap it into a stand-alone chip in order to add a layer of complexity
// 	// for testing purposes.
// 	clk, err := hwsim.Chip("CLK", "enable", "tick",
// 		tl.nand("a=enable, b=tick, out=tick"),
// 	)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	c, err := hwsim.NewCircuit(
// 		hwsim.Input(func() bool { return enable })("out=enable"),
// 		clk("enable=enable, tick=out"),
// 		hwsim.Output(func(out bool) { tick = out })("in=out"),
// 	)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	defer c.Dispose()

// 	// we have two wires: "enable" and "out".
// 	// note that Output("out", ...) is delayed by one tick after the Nand updates it.

// 	enable = false
// 	c.Step()
// 	check(false)
// 	c.Step()
// 	// this is an expected signal change appearing in the first couple of ticks due to signal propagation delay
// 	check(true)
// 	c.Step()
// 	check(true)

// 	enable = true
// 	c.Step()
// 	check(true)
// 	c.Step()
// 	check(true)
// 	c.Step()
// 	// the clock starts ticking now.
// 	check(false)
// 	c.Step()
// 	check(true)
// 	c.Step()
// 	check(false)
// 	c.Step()
// 	check(true)
// 	enable = false
// 	c.Step()
// 	check(false)
// 	c.Step()
// 	check(true)
// 	c.Step()
// 	// the clock stops ticking now.
// 	check(true)
// 	c.Step()
// 	check(true)
// }

// // This bench is here to becnhmark the workers sync mechanism overhead.
// func BenchmarkCircuit_Step(b *testing.B) {
// 	workers := runtime.GOMAXPROCS(-1)
// 	parts := make([]hwsim.Part, 0, workers)
// 	for i := 0; i < workers; i++ {
// 		parts = append(parts, tl.not(""))
// 	}

// 	c, err := hwsim.NewCircuit(workers, testTPC, parts...)
// 	if err != nil {
// 		b.Fatal(err)
// 	}
// 	defer c.Dispose()

// 	for i := 0; i < b.N; i++ {
// 		c.Step()
// 	}
// }

func ExampleIO() {
	fmt.Println(hwsim.IO("a,b"))
	fmt.Println(hwsim.IO("a[2],b"))
	fmt.Println(hwsim.IO("a[0..0],b[1..2]"))

	// Output:
	// [a b]
	// [a[0] a[1] b]
	// [a[0] b[1] b[2]]
}

// testLib is a test library of components built entirely of nands
//
type testLib struct {
	nandSpec *hwsim.PartSpec
	not      hwsim.NewPartFn
	and      hwsim.NewPartFn
	or       hwsim.NewPartFn
	nor      hwsim.NewPartFn
	xor      hwsim.NewPartFn
	and3     hwsim.NewPartFn
	and4     hwsim.NewPartFn
	or3      hwsim.NewPartFn
	or4      hwsim.NewPartFn
	lcu      hwsim.NewPartFn
	cla4     hwsim.NewPartFn
}

func newTestLib() *testLib {
	tl := &testLib{
		nandSpec: &hwsim.PartSpec{
			Name:    "NAND",
			Inputs:  []string{"a", "b"},
			Outputs: []string{"out"},
			Mount: func(s *hwsim.Socket) hwsim.Updater {
				a, b, out := s.Pin("a"), s.Pin("b"), s.Pin("out")
				f := hwsim.UpdaterFn(func(clk bool) {
					out.Send(clk, !(a.Recv(clk) && b.Recv(clk)))
				})
				out.Connect(f)
				return f
			}},
	}
	var err error
	tl.not, err = hwsim.Chip("not", "in", "out", tl.nand("a=in, b=in, out=out"))
	if err != nil {
		panic(err)
	}
	tl.and, err = hwsim.Chip("and", "a, b", "out", tl.nand("a=a, b=b, out=nab"), tl.not("in=nab, out=out"))
	if err != nil {
		panic(err)
	}
	tl.or, err = hwsim.Chip("or", "a, b", "out", tl.not("in=a, out=na"), tl.not("in=b, out=nb"), tl.nand("a=na, b=nb, out=out"))
	if err != nil {
		panic(err)
	}
	tl.nor, err = hwsim.Chip("nor", "a, b", "out", tl.or("a=a, b=b, out=oab"), tl.not("in=oab, out=out"))
	if err != nil {
		panic(err)
	}
	tl.xor, err = hwsim.Chip("xor", "a, b", "out",
		tl.nand("a=a, b=b, out=nab"),
		tl.nand("a=a, b=nab, out=o0"),
		tl.nand("a=nab, b=b, out=o1"),
		tl.nand("a=o0, b=o1, out=out"))
	if err != nil {
		panic(err)
	}
	tl.or3, err = hwsim.Chip("or3", "a, b, c", "out", tl.or("a=a, b=b, out=oab"), tl.or("a=oab, b=c, out=out"))
	if err != nil {
		panic(err)
	}
	tl.or4, err = hwsim.Chip("or4", "a, b, c, d", "out",
		tl.or("a=a, b=b, out=oab"),
		tl.or("a=c, b=d, out=ocd"),
		tl.or("a=oab, b=ocd, out=out"))
	if err != nil {
		panic(err)
	}
	tl.and3, err = hwsim.Chip("and3", "a, b, c", "out", tl.and("a=a, b=b, out=aab"), tl.and("a=aab, b=c, out=out"))
	if err != nil {
		panic(err)
	}
	tl.and4, err = hwsim.Chip("and4", "a, b, c, d", "out",
		tl.and("a=a, b=b, out=aab"),
		tl.and("a=c, b=d, out=acd"),
		tl.and("a=aab, b=acd, out=out"))
	if err != nil {
		panic(err)
	}

	add1, err := hwsim.Chip("1bitAdder", "a, b, c0", "s, g, p, c",
		tl.xor("a=a, b=b, out=p"),
		tl.and("a=a, b=b, out=g"),
		tl.xor("a=p, b=c0, out=s"),
	)
	if err != nil {
		panic(err)
	}
	tl.lcu, err = hwsim.Chip("4bitLCU",
		"p[4], g[4], c0", "p, g, c1, c2, c3",
		tl.and("a=c0, b=p[0], out=c1[0]"),
		tl.or("a=c1[0], b=g[0], out=c1"),

		tl.and3("a=p[0], b=p[1], c=c0, out=c2[0]"),
		tl.and("a=g[0], b=p[1], out=c2[1]"),
		tl.or3("a=c2[0], b=c2[1], c=g[1], out=c2"),

		tl.and4("a=p[0], b=p[1], c=p[2], d=c0, out=c3[0]"),
		tl.and3("a=g[0], b=p[1], c=p[2], out=c3[1]"),
		tl.and("a=g[1], b=p[2], out=c3[2]"),
		tl.or4("a=c3[0], b=c3[1], c=c3[2], d=g[2], out=c3"),

		tl.and4("a=p[0], b=p[1], c=p[2], d=p[3], out=p"),
		tl.and4("a=g[0], b=p[1], c=p[2], d=p[3], out=c4[0]"),
		tl.and3("a=g[1], b=p[2], c=p[3], out=c4[1]"),
		tl.and("a=g[2], b=p[3], out=c4[2]"),
		tl.or4("a=c4[0], b=c4[1], c=c4[2], d=g[3], out=g"),
	)
	if err != nil {
		panic(err)
	}
	tl.cla4, err = hwsim.Chip("CLA4", "a[4], b[4], c0", "out[4], p, g",
		add1("a=a[0], b=b[0], c0=c0, s=out[0], g=g[0], p=p[0]"),
		add1("a=a[1], b=b[1], c0=c1, s=out[1], g=g[1], p=p[1]"),
		add1("a=a[2], b=b[2], c0=c2, s=out[2], g=g[2], p=p[2]"),
		add1("a=a[3], b=b[3], c0=c3, s=out[3], g=g[3], p=p[3]"),
		tl.lcu("p[0..3]=p[0..3], g[0..3]=g[0..3], c0=c0, p=p, g=g, c1=c1, c2=c2, c3=c3"),
	)
	if err != nil {
		panic(err)
	}

	return tl
}

func (tl *testLib) nand(c string) hwsim.Part {
	return tl.nandSpec.NewPart(c)
}

var tl *testLib = newTestLib()

func Test_testLib(t *testing.T) {
	adderN := func(bits int) hwsim.NewPartFn {
		p := &hwsim.PartSpec{
			Name:    "AdderN",
			Inputs:  hwsim.IO(fmt.Sprintf("a[%d], b[%d]", bits, bits)),
			Outputs: hwsim.IO(fmt.Sprintf("c, out[%d]", bits)),
			Mount: func(s *hwsim.Socket) hwsim.Updater {
				a, b, out := s.Bus("a", bits), s.Bus("b", bits), s.Bus("out", bits)
				carry := s.Pin("c")
				f := hwsim.UpdaterFn(
					func(clk bool) {
						va := a.GetInt64(clk)
						vb := b.GetInt64(clk)
						s := va + vb
						carry.Send(clk, s >= 1<<uint(bits))
						out.SetInt64(clk, s&(1<<uint(bits)-1))
					})
				out.Connect(f)
				carry.Connect(f)
				return f
			},
		}
		return p.NewPart
	}
	wrap4, err := hwsim.Chip("CLA4wrapper", "a[4], b[4]", "out[4], c",
		tl.cla4("a[0..3]=a[0..3], b[0..3]=b[0..3], out[0..3]=out[0..3], g=c"),
	)
	if err != nil {
		t.Fatal(err)
	}
	hwtest.ComparePart(t, adderN(4), wrap4)

	// dummy := hwsim.Output(func(bool) {})
	// f := hwsim.Input(func() bool { return false })
	add16, err := hwsim.Chip("Adder16", "a[16], b[16]", "out[16], c",
		// dummy("in=p"),
		// f("out=c0"),
		tl.lcu("p[0..3]=p[0..3], g[0..3]=g[0..3], g=c, c1=c1, c2=c2, c3=c3, c0=false"),
		tl.cla4("a[0..3]=a[0..3],   b[0..3]=b[0..3],   c0=false, out[0..3]=out[0..3],   p=p[0], g=g[0]"),
		tl.cla4("a[0..3]=a[4..7],   b[0..3]=b[4..7],   c0=c1, out[0..3]=out[4..7],   p=p[1], g=g[1]"),
		tl.cla4("a[0..3]=a[8..11],  b[0..3]=b[8..11],  c0=c2, out[0..3]=out[8..11],  p=p[2], g=g[2]"),
		tl.cla4("a[0..3]=a[12..15], b[0..3]=b[12..15], c0=c3, out[0..3]=out[12..15], p=p[3], g=g[3]"),
	)
	if err != nil {
		t.Fatal(err)
	}
	hwtest.ComparePart(t, adderN(16), add16)
}
