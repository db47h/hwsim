package hwlib_test

import (
	"testing"

	"github.com/db47h/hwsim/hwtest"

	hw "github.com/db47h/hwsim"
	hl "github.com/db47h/hwsim/hwlib"
)

func TestMuxN(t *testing.T) {
	m, err := hw.Chip("myMux16", hw.In("a[4], b[4], sel"), hw.Out("out[4]"), hw.Parts{
		hl.Mux("a=a[0], b=b[0], sel=sel, out=out[0]"),
		hl.Mux("a=a[1], b=b[1], sel=sel, out=out[1]"),
		hl.Mux("a=a[2], b=b[2], sel=sel, out=out[2]"),
		hl.Mux("a=a[3], b=b[3], sel=sel, out=out[3]"),
	})

	if err != nil {
		t.Fatal(err)
	}

	hwtest.ComparePart(t, 4, hl.MuxN(4), m)
}

func TestMuxMWayN(t *testing.T) {
	mux4 := hl.MuxN(4)
	mux44, err := hw.Chip("myMux4Way4", hw.In("a[4], b[4], c[4], d[4], sel[2]"), hw.Out("out[4]"), hw.Parts{
		mux4("a[0..3]=a[0..3], b[0..3]=b[0..3], sel=sel[0], out[0..3]=m0[0..3]"),
		mux4("a[0..3]=c[0..3], b[0..3]=d[0..3], sel=sel[0], out[0..3]=m1[0..3]"),
		mux4("a[0..3]=m0[0..3], b[0..3]=m1[0..3], sel=sel[1], out[0..3]=out[0..3]"),
	})
	if err != nil {
		t.Fatal(err)
	}
	hwtest.ComparePart(t, 4, hl.MuxMWayN(4, 4), mux44)
}
