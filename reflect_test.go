package hwsim_test

import (
	"testing"

	"github.com/db47h/hwsim"
	"github.com/db47h/hwsim/hwlib"
	"github.com/db47h/hwsim/hwtest"
)

type testPart struct {
	A   [8]int `hw:"in"`
	B   [8]int `hw:"in"`
	Sel int    `hw:"in"`
	Out [8]int `hw:"out"`
}

func (t *testPart) Update(c *hwsim.Circuit) {
	if c.Get(t.Sel) {
		for i, b := range t.B {
			c.Set(t.Out[i], c.Get(b))
		}
	} else {
		for i, a := range t.A {
			c.Set(t.Out[i], c.Get(a))
		}
	}
}

func Test_MakePart(t *testing.T) {
	p := hwsim.MakePart((*testPart)(nil)).NewPart
	hwtest.ComparePart(t, 8, hwlib.MuxN(8), p)
}
