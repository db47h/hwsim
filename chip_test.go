package hwsim_test

import (
	"testing"

	hw "github.com/db47h/hwsim"
)

func Test_Chip(t *testing.T) {
	unkChip, err := hw.Chip("TESTCHIP", hw.In{"a", "b"}, hw.Out{"out"}, hw.Parts{
		// chip input a is unused
		hw.Nand("a=b, b=b, out=out"),
	})
	if err != nil {
		t.Fatal(err)
	}
	data := []struct {
		name  string
		in    hw.In
		out   hw.Out
		parts hw.Parts
		err   string
	}{
		{"true_out", hw.In{"a", "b"}, hw.Out{"out"}, hw.Parts{
			hw.Nand("a=a, b=b, out=true"),
			hw.Nand("a=a, b=b, out=out"),
		}, "NAND.out:true: output pin connected to constant true input"},
		{"false_out", hw.In{"a", "b"}, hw.Out{"out"}, hw.Parts{
			hw.Nand("a=a, b=b, out=false"),
			hw.Nand("a=a, b=b, out=out"),
		}, "NAND.out:false: output pin connected to constant false input"},
		{"multi_out", hw.In{"a", "b"}, hw.Out{"out"}, hw.Parts{
			hw.Nand("a=a, b=b, out=a"),
			hw.Nand("a=a, b=b, out=out"),
		}, "NAND.out:a: chip input pin used as output"},
		{"multi_out2", hw.In{"a", "b"}, hw.Out{"out"}, hw.Parts{
			hw.Nand("a=a, b=b, out=x"),
			hw.Nand("a=a, b=b, out=x"),
			hw.Not("in=x, out=out"),
		}, "NAND.out:x: output pin already used as output"},
		{"no_output", hw.In{"a", "b"}, hw.Out{"out"}, hw.Parts{
			hw.Nand("a=a, b=wx, out=out"),
		}, "pin wx not connected to any output"},
		{"no_input", hw.In{"a", "b"}, hw.Out{"out"}, hw.Parts{
			hw.Nand("a=a, b=b, out=foo"),
			hw.Nand("a=a, b=b, out=out"),
		}, "pin foo not connected to any input"},
		{"unconnected_in", hw.In{"a", "b"}, hw.Out{"out"}, hw.Parts{}, ""},
		{"unknown_pin", hw.In{"a", "b"}, hw.Out{"out"}, hw.Parts{
			hw.Nand("a=a, typo=b, out=out"),
		}, "invalid pin name typo for part NAND"},
		{"unknown_pin", hw.In{"a", "b"}, hw.Out{"out"}, hw.Parts{
			unkChip("a=a, typo=b, out=out"),
		}, "invalid pin name typo for part TESTCHIP"},
		{"unknown_pin", hw.In{"a", "b"}, hw.Out{"out"}, hw.Parts{
			unkChip("a=a, b=b, out=out"),
		}, ""},
	}
	for _, d := range data {
		t.Run(d.name, func(t *testing.T) {
			_, err := hw.Chip(d.name, d.in, d.out, d.parts)
			if err == nil && d.err != "" || err != nil && err.Error() != d.err {
				t.Errorf("Got error %q, expected %q", err, d.err)
				return
			}
		})
	}

}

func Test_Chip_omitted_pins(t *testing.T) {
	var a, b, c, tr, f, o0, o1 int
	dummy := hw.MakePart(&hw.PartSpec{
		Name: "dummy",
		In:   hw.In{"a", "b", "c", "t", "f"},
		Out:  hw.Out{"o0", "o1"},
		Mount: func(s *hw.Socket) []hw.Component {
			a, b, c, tr, f, o0, o1 = s.Pin("a"), s.Pin("b"), s.Pin("c"), s.Pin("t"), s.Pin("f"), s.Pin("o0"), s.Pin("o1")
			return nil
		}})
	// this is just to add another layer of testing.
	// inspecting o0 and o1 shows that another dummy wire was allocated for dummy.o0:wo0
	wrapper, err := hw.Chip("wrapper", hw.In{"wa", "wb"}, hw.Out{"wo0", "wo1"}, hw.Parts{
		dummy("a=wa, c=clk, t=true, f=false, o0=wo0"),
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = hw.NewCircuit(0, 0, hw.Parts{wrapper("")})
	if err != nil {
		t.Fatal(err)
	}

	if a != 0 || b != 0 || f != 0 { // 0 = cstFalse
		t.Errorf("a = %v, b = %v, f = %v, all must be 0", a, b, f)
	}
	if tr != 1 { // 1 = cstTrue
		t.Errorf("t = %v, must be 1", tr)
	}
	if c != 2 { // 2 = cstClk
		t.Errorf("c = %v, must be 2", c)
	}
	if o0 < 3 || o1 < 3 { // 3 = cstCount
		t.Errorf("o0 = %v, o1 = %v, both must be > 3", o0, o1)
	}
}
