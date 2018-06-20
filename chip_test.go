package hwsim_test

import (
	"testing"

	hw "github.com/db47h/hwsim"
)

func TestChip_errors(t *testing.T) {
	unkChip, err := hw.Chip("TESTCHIP", "a, b", "out",
		// chip input a is unused
		tl.nand("a=b, b=b, out=out"),
	)
	if err != nil {
		t.Fatal(err)
	}
	data := []struct {
		name  string
		in    string
		out   string
		parts []hw.Part
		err   string
	}{
		{"true_out", "a, b", "out", []hw.Part{
			tl.nand("a=a, b=b, out=true"),
			tl.nand("a=a, b=b, out=out"),
		}, "NAND.out:true: output pin connected to constant true input"},
		{"false_out", "a, b", "out", []hw.Part{
			tl.nand("a=a, b=b, out=false"),
			tl.nand("a=a, b=b, out=out"),
		}, "NAND.out:false: output pin connected to constant false input"},
		{"multi_out", "a, b", "out", []hw.Part{
			tl.nand("a=a, b=b, out=a"),
			tl.nand("a=a, b=b, out=out"),
		}, "NAND.out:a: chip input pin used as output"},
		{"multi_out2", "a, b", "out", []hw.Part{
			tl.nand("a=a, b=b, out=x"),
			tl.nand("a=a, b=b, out=x"),
			tl.not("in=x, out=out"),
		}, "NAND.out:x: output pin already used as output"},
		{"no_output", "a, b", "out", []hw.Part{
			tl.nand("a=a, b=wx, out=out"),
		}, "pin wx not connected to any output"},
		{"no_output", "", "out", []hw.Part{
			tl.not("in=out"),
		}, "pin out not connected to any output"},
		{"no_input", "a, b", "out", []hw.Part{
			tl.nand("a=a, b=b, out=foo"),
			tl.nand("a=a, b=b, out=out"),
		}, "pin foo not connected to any input"},
		{"unconnected_in", "a, b", "out", []hw.Part{}, ""},
		{"unknown_pin", "a, b", "out", []hw.Part{
			tl.nand("a=a, typo=b, out=out"),
		}, "invalid pin name typo for part NAND"},
		{"unknown_pin", "a, b", "out", []hw.Part{
			unkChip("a=a, typo=b, out=out"),
		}, "invalid pin name typo for part TESTCHIP"},
		{"unknown_pin", "a, b", "out", []hw.Part{
			unkChip("a=a, b=b, out=out"),
		}, ""},
	}
	for _, d := range data {
		t.Run(d.name, func(t *testing.T) {
			_, err := hw.Chip(d.name, d.in, d.out, d.parts...)
			if err == nil && d.err != "" || err != nil && err.Error() != d.err {
				t.Errorf("Got error %q, expected %q", err, d.err)
				return
			}
		})
	}

}

func TestChip_omitted_pins(t *testing.T) {
	var cFalse, cTrue, cClk, a, b, c, tr, f, o0, o1 *hw.Wire
	dummy := (&hw.PartSpec{
		Name:    "dummy",
		Inputs:  hw.IO("a, b, c, t, f"),
		Outputs: hw.IO("o0, o1"),
		Mount: func(s *hw.Socket) hw.Updater {
			cFalse = s.Wire(hw.False)
			cTrue = s.Wire(hw.True)
			cClk = s.Wire(hw.Clk)
			a, b, c, tr, f, o0, o1 = s.Wire("a"), s.Wire("b"), s.Wire("c"), s.Wire("t"), s.Wire("f"), s.Wire("o0"), s.Wire("o1")
			return hw.UpdaterFn(func(clk bool) {})
		}}).NewPart
	// this is just to add another layer of testing.
	// inspecting o0 and o1 shows that another dummy wire was allocated for dummy.o0:wo0
	wrapper, err := hw.Chip("wrapper", "wa, wb", "wo0, wo1",
		dummy("a=wa, c=clk, t=true, f=false, o0=wo0"),
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = hw.NewCircuit(wrapper(""))
	if err != nil {
		t.Fatal(err)
	}

	if a != cFalse || b != cFalse || f != cFalse { // 0 = cstFalse
		t.Errorf("a = %p, b = %p, f = %p, all must be False (%p)", a, b, f, cFalse)
	}
	if tr != cTrue { // 1 = cstTrue
		t.Errorf("t = %p, must be true (%p)", tr, cTrue)
	}
	if c != cClk { // 2 = cstClk
		t.Errorf("c = %p, must be clk (%p)", c, cClk)
	}
	if o0 == nil || o0 == cFalse || o0 == cTrue || o0 == cClk {
		t.Errorf("o0 = %p, must be != nil and != cst pins", o0)
	}
	if o1 == nil || o1 == cFalse || o1 == cTrue || o1 == cClk {
		t.Errorf("o1 = %p, must be != nil and != cst pins", o1)
	}
}

func TestChip_fanout_to_outputs(t *testing.T) {
	gate, err := hw.Chip("FANOUT", "in", "a, b, bus[2], c",
		tl.or("a=in, b=in, out=a, out=b, out=bus[0..1]"),
	)
	if err != nil {
		t.Fatal(err)
	}
	wrapper1, err := hw.Chip("FANOUT_Wrapper", "in", "o[8]",
		gate("in=in, a=o[0..1], b=o[2..3], bus[0]=o[4..5], bus[1]=o[6..7]"),
	)
	if err != nil {
		t.Fatal(err)
	}
	var out int64
	c, err := hw.NewCircuit(
		//		hw.Input(func() bool { return true })("out=in"),
		wrapper1("in=true, o[0..7]=wrapOut[0..7]"),
		hw.OutputN(16, func(v int64) { out = v })("in[0..7]=wrapOut[0..7]"),
	)
	if err != nil {
		t.Fatal(err)
	}

	c.TickTock()
	if out != 255 {
		t.Fatalf("out = %d != 255", out)
	}
}
