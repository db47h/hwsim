package hwtest_test

import (
	"testing"

	hw "github.com/db47h/hwsim"
	"github.com/db47h/hwsim/hwtest"
)

var nand = &hw.PartSpec{
	Name:    "nand",
	Inputs:  []string{"a", "b"},
	Outputs: []string{"out"},
	Mount: func(s *hw.Socket) hw.Updater {
		a, b, out := s.Wire("a"), s.Wire("b"), s.Wire("out")
		return hw.UpdaterFn(func(clk bool) {
			out.Send(clk, !(a.Recv(clk) && b.Recv(clk)))
		})
	}}
var or = &hw.PartSpec{
	Name:    "or",
	Inputs:  []string{"a", "b"},
	Outputs: []string{"out"},
	Mount: func(s *hw.Socket) hw.Updater {
		a, b, out := s.Wire("a"), s.Wire("b"), s.Wire("out")
		return hw.UpdaterFn(func(clk bool) {
			out.Send(clk, a.Recv(clk) || b.Recv(clk))
		})
	}}

func TestComparePart(t *testing.T) {
	or2, err := hw.Chip("custom_or", "a,b", "out",
		nand.NewPart("a=a, b=a, out=notA"),
		nand.NewPart("a=b, b=b, out=notB"),
		nand.NewPart("a=notA, b=notB, out=out"),
	)
	if err != nil {
		t.Fatal(err)
	}
	hwtest.ComparePart(t, or.NewPart, or2)
}
