package hwlib_test

import (
	"testing"

	"github.com/db47h/hwsim/hwtest"

	hw "github.com/db47h/hwsim"
	hl "github.com/db47h/hwsim/hwlib"
)

func TestHalfAdder(t *testing.T) {
	h, err := hw.Chip("myHalfAdder", hw.In("a, b"), hw.Out("s, c"), hw.Parts{
		hl.Xor("a=a, b=b, out=s"),
		hl.And("a=a, b=b, out=c"),
	})
	if err != nil {
		t.Fatal(err)
	}
	hwtest.ComparePart(t, 4, hl.HalfAdder, h)
}

func TestFullAdder(t *testing.T) {
	h, err := hw.Chip("myHalfAdder", hw.In("a, b"), hw.Out("s, c"), hw.Parts{
		hl.Xor("a=a, b=b, out=s"),
		hl.And("a=a, b=b, out=c"),
	})
	if err != nil {
		t.Fatal(err)
	}
	adder, err := hw.Chip("myFullAdder", hw.In("a, b, cin"), hw.Out("s, cout"), hw.Parts{
		h("a=a, b=b, s=s0, c=c0"),
		h("a=s0, b=cin, s=s, c=c1"),
		hl.Or("a=c0, b=c1, out=cout"),
	})
	if err != nil {
		t.Fatal(err)
	}
	hwtest.ComparePart(t, 8, hl.FullAdder, adder)
}

func TestAdderN(t *testing.T) {
	add4, err := hw.Chip("Adder4", hw.In("a[4], b[4]"), hw.Out("out[4], c"), hw.Parts{
		hl.HalfAdder("a=a[0], b=b[0], s=out[0], c=c0"),
		hl.FullAdder("a=a[1], b=b[1], cin=c0, s=out[1], cout=c1"),
		hl.FullAdder("a=a[2], b=b[2], cin=c1, s=out[2], cout=c2"),
		hl.FullAdder("a=a[3], b=b[3], cin=c2, s=out[3], cout=c"),
	})
	if err != nil {
		t.Fatal(err)
	}
	hwtest.ComparePart(t, 8, hl.AdderN(4), add4)
}

func Test_adders(t *testing.T) {
	and3 := hl.AndNWay(3)
	and4 := hl.AndNWay(4)
	or3 := hl.OrNWay(3)
	or4 := hl.OrNWay(4)

	add1, err := hw.Chip("1bitAdder", hw.In("a, b, c0"), hw.Out("s, g, p"), hw.Parts{
		hl.Xor("a=a, b=b, out=p"),
		hl.And("a=a, b=b, out=g"),
		hl.Xor("a=p, b=c0, out=s"),
	})
	if err != nil {
		t.Fatal(err)
	}

	lcu, err := hw.Chip("4bitLCU",
		hw.In("p[4], g[4], c0"),
		hw.Out("p, g, c1, c2, c3"), hw.Parts{
			hl.And("a=c0, b=p[0], out=c1[0]"),
			hl.Or("a=c1[0], b=g[0], out=c1"),

			and3("in[0..1]=p[0..1], in[2]=c0, out=c2[0]"),
			hl.And("a=g[0], b=p[1], out=c2[1]"),
			or3("in[0..1]=c2[0..1], in[2]=g[1], out=c2"),

			and4("in[0..2]=p[0..2], in[3]=c0, out=c3[0]"),
			and3("in[0]=g[0], in[1..2]=p[1..2], out=c3[1]"),
			hl.And("a=g[1], b=p[2], out=c3[2]"),
			or4("in[0..2]=c3[0..2], in[3]=g[2], out=c3"),

			and4("in[0..3]=p[0..3], out=p"),
			and4("in[0]=g[0], in[1..3]=p[1..3], out=c4[0]"),
			and3("in[0]=g[1], in[1..2]=p[2..3], out=c4[1]"),
			hl.And("a=g[2], b=p[3], out=c4[2]"),
			or4("in[0..2]=c4[0..2], in[3]=g[3], out=g"),
		})
	if err != nil {
		t.Fatal(err)
	}
	add4, err := hw.Chip("Adder4", hw.In("a[4], b[4], c0"), hw.Out("out[4], p, g"), hw.Parts{
		add1("a=a[0], b=b[0], c0=c0, s=out[0], g=g[0], p=p[0]"),
		add1("a=a[1], b=b[1], c0=c1, s=out[1], g=g[1], p=p[1]"),
		add1("a=a[2], b=b[2], c0=c2, s=out[2], g=g[2], p=p[2]"),
		add1("a=a[3], b=b[3], c0=c3, s=out[3], g=g[3], p=p[3]"),
		lcu("p[0..3]=p[0..3], g[0..3]=g[0..3], c0=c0, p=p, g=g, c1=c1, c2=c2, c3=c3"),
	})
	if err != nil {
		t.Fatal(err)
	}
	add16, err := hw.Chip("Adder4", hw.In("a[16], b[16], c0"), hw.Out("out[16], p, g"), hw.Parts{
		lcu("p[0..3]=p[0..3], g[0..3]=g[0..3], c0=c0, p=p, g=g, c1=c1, c2=c2, c3=c3"),
		add4("a[0..3]=a[0..3],   b[0..3]=b[0..3],   c0=c0, out[0..3]=out[0..3],   p=p[0], g=g[0]"),
		add4("a[0..3]=a[4..7],   b[0..3]=b[4..7],   c0=c1, out[0..3]=out[4..7],   p=p[1], g=g[1]"),
		add4("a[0..3]=a[8..11],  b[0..3]=b[8..11],  c0=c2, out[0..3]=out[8..11],  p=p[2], g=g[2]"),
		add4("a[0..3]=a[12..15], b[0..3]=b[12..15], c0=c3, out[0..3]=out[12..15], p=p[3], g=g[3]"),
	})
	if err != nil {
		t.Fatal(err)
	}
	wrap, err := hw.Chip("myAdder16", hw.In("a[16], b[16]"), hw.Out("out[16], c"), hw.Parts{
		add16("a[0..15]=a[0..15], b[0..15]=b[0..15], out[0..15]=out[0..15], g=c"),
	})
	if err != nil {
		t.Fatal(err)
	}
	hwtest.ComparePart(t, 16, hl.AdderN(16), wrap)

	// var a, b, out int64
	// var cc bool
	// c, err := hw.NewCircuit(0, 32, hw.Parts{
	// 	hl.InputN(4, func() int64 { return a })("out[0..3]=a[0..3]"),
	// 	hl.InputN(4, func() int64 { return b })("out[0..3]=b[0..3]"),
	// 	add4("a[0..3]=a[0..3], b[0..3]=b[0..3], out[0..3]=s[0..3], g=c"),
	// 	hl.Output(func(v bool) { cc = v })("in=c"),
	// 	hl.OutputN(4, func(v int64) { out = v })("in[0..3]=s[0..3]"),
	// })
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// testAdd := func() {
	// 	a, b = rand.Int63n(16), rand.Int63n(16)
	// 	carry := a+b >= 16
	// 	sum := (a + b) & 0xF
	// 	s := c.Steps()
	// 	for i := 0; i < 100; i++ {
	// 		c.Step()
	// 		// t.Logf("%d+%d=%d, +%v in %d steps", a, b, out, cc, c.Steps()-s)
	// 		if out == sum && cc == carry {
	// 			if cc {
	// 				out |= 0x10
	// 			}
	// 			t.Logf("%d+%d=%d, in %d steps", a, b, out, c.Steps()-s)
	// 			break
	// 		}

	// 	}
	// }

	// for i := 0; i < 20; i++ {
	// 	testAdd()
	// }
}
