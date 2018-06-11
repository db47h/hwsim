package hwlib_test

import (
	"testing"

	"github.com/db47h/hwsim/hwtest"

	hw "github.com/db47h/hwsim"
	hl "github.com/db47h/hwsim/hwlib"
)

func TestSpecMux_8(t *testing.T) {
	mux8 := hl.SpecMuxN(4).NewPart

	m, err := hw.Chip("myMux16", hw.In("a[4], b[4], sel"), hw.Out("out[4]"), hw.Parts{
		hl.Mux("a=a[0], b=b[0], sel=sel, out=out[0]"),
		hl.Mux("a=a[1], b=b[1], sel=sel, out=out[1]"),
		hl.Mux("a=a[2], b=b[2], sel=sel, out=out[2]"),
		hl.Mux("a=a[3], b=b[3], sel=sel, out=out[3]"),
	})

	if err != nil {
		t.Fatal(err)
	}

	hwtest.ComparePart(t, 4, mux8, m)
}
