package hwsim_test

import (
	"testing"

	"github.com/db47h/hwsim"
	"github.com/db47h/hwsim/hwtest"
)

type testPart struct {
	A   [4]*hwsim.Pin `hw:"in"`
	B   [4]*hwsim.Pin `hw:"in"`
	Sel *hwsim.Pin    `hw:"in"`
	Out [4]*hwsim.Pin `hw:"out"`
}

func (t *testPart) Update(clk bool) {
	if t.Sel.Recv(clk) {
		for i, b := range t.B {
			t.Out[i].Send(clk, b.Recv(clk))
		}
	} else {
		for i, a := range t.A {
			t.Out[i].Send(clk, a.Recv(clk))
		}
	}
}

func Test_MakePart(t *testing.T) {
	m, err := hwsim.Chip("myMux4", "a[4], b[4], sel", "out[4]",
		tl.mux("a=a[0], b=b[0], sel=sel, out=out[0]"),
		tl.mux("a=a[1], b=b[1], sel=sel, out=out[1]"),
		tl.mux("a=a[2], b=b[2], sel=sel, out=out[2]"),
		tl.mux("a=a[3], b=b[3], sel=sel, out=out[3]"),
	)
	if err != nil {
		t.Fatal(err)
	}

	p := hwsim.MakePart((*testPart)(nil)).NewPart
	hwtest.ComparePart(t, m, p)
}
