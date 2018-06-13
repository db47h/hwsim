package hwsim_test

import (
	"testing"

	hw "github.com/db47h/hwsim"
	hl "github.com/db47h/hwsim/hwlib"
)

func TestChip_errors(t *testing.T) {
	unkChip, err := hw.Chip("TESTCHIP", "a, b", "out", hw.Parts{
		// chip input a is unused
		hl.Nand("a=b, b=b, out=out"),
	})
	if err != nil {
		t.Fatal(err)
	}
	data := []struct {
		name  string
		in    string
		out   string
		parts hw.Parts
		err   string
	}{
		{"true_out", "a, b", "out", hw.Parts{
			hl.Nand("a=a, b=b, out=true"),
			hl.Nand("a=a, b=b, out=out"),
		}, "NAND.out:true: output pin connected to constant true input"},
		{"false_out", "a, b", "out", hw.Parts{
			hl.Nand("a=a, b=b, out=false"),
			hl.Nand("a=a, b=b, out=out"),
		}, "NAND.out:false: output pin connected to constant false input"},
		{"multi_out", "a, b", "out", hw.Parts{
			hl.Nand("a=a, b=b, out=a"),
			hl.Nand("a=a, b=b, out=out"),
		}, "NAND.out:a: chip input pin used as output"},
		{"multi_out2", "a, b", "out", hw.Parts{
			hl.Nand("a=a, b=b, out=x"),
			hl.Nand("a=a, b=b, out=x"),
			hl.Not("in=x, out=out"),
		}, "NAND.out:x: output pin already used as output"},
		{"no_output", "a, b", "out", hw.Parts{
			hl.Nand("a=a, b=wx, out=out"),
		}, "pin wx not connected to any output"},
		{"no_output", "", "out", hw.Parts{
			hl.Not("in=out"),
		}, "pin out not connected to any output"},
		{"no_input", "a, b", "out", hw.Parts{
			hl.Nand("a=a, b=b, out=foo"),
			hl.Nand("a=a, b=b, out=out"),
		}, "pin foo not connected to any input"},
		{"unconnected_in", "a, b", "out", hw.Parts{}, ""},
		{"unknown_pin", "a, b", "out", hw.Parts{
			hl.Nand("a=a, typo=b, out=out"),
		}, "invalid pin name typo for part NAND"},
		{"unknown_pin", "a, b", "out", hw.Parts{
			unkChip("a=a, typo=b, out=out"),
		}, "invalid pin name typo for part TESTCHIP"},
		{"unknown_pin", "a, b", "out", hw.Parts{
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

func TestChip_omitted_pins(t *testing.T) {
	var a, b, c, tr, f, o0, o1 int
	dummy := (&hw.PartSpec{
		Name:    "dummy",
		Inputs:  hw.IO("a, b, c, t, f"),
		Outputs: hw.IO("o0, o1"),
		Mount: func(s *hw.Socket) []hw.Component {
			a, b, c, tr, f, o0, o1 = s.Pin("a"), s.Pin("b"), s.Pin("c"), s.Pin("t"), s.Pin("f"), s.Pin("o0"), s.Pin("o1")
			return nil
		}}).NewPart
	// this is just to add another layer of testing.
	// inspecting o0 and o1 shows that another dummy wire was allocated for dummy.o0:wo0
	wrapper, err := hw.Chip("wrapper", "wa, wb", "wo0, wo1", hw.Parts{
		dummy("a=wa, c=clk, t=true, f=false, o0=wo0"),
	})
	if err != nil {
		t.Fatal(err)
	}

	cc, err := hw.NewCircuit(0, 0, hw.Parts{wrapper("")})
	if err != nil {
		t.Fatal(err)
	}
	defer cc.Dispose()

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

func TestChip_fanout_to_outputs(t *testing.T) {
	gate, err := hw.Chip("FANOUT", "in", "a, b, bus[2], c", hw.Parts{
		hl.Or("a=in, b=in, out=a, out=b, out=bus[0..1]"),
	})
	if err != nil {
		t.Fatal(err)
	}
	wrapper1, err := hw.Chip("FANOUT_Wrapper", "in", "o[8]", hw.Parts{
		gate("in=in, a=o[0..1], b=o[2..3], bus[0]=o[4..5], bus[1]=o[6..7]"),
	})
	if err != nil {
		t.Fatal(err)
	}
	var out int64
	c, err := hw.NewCircuit(0, testTPC, hw.Parts{
		wrapper1("in=true, o[0..7]=wrapOut[0..7]"),
		hl.OutputN(16, func(v int64) { out = v })("in[0..7]=wrapOut[0..7]"),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Dispose()

	c.TickTock()
	if out != 255 {
		t.Fatalf("out = %d != 255", out)
	}
}
