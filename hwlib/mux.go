// Copyright 2018 Denis Bernard <db047h@gmail.com>
// Licensed under the MIT license. See license text in the LICENSE file.

package hwlib

import (
	"strconv"

	"github.com/db47h/hwsim"
)

// Mux returns a multiplexer.
//
//	Inputs: a, b, sel
//	Outputs: out
//	Function: if sel == 0 { out = a } else { out = b }
//
func Mux(w string) hwsim.Part { return mux.NewPart(w) }

var mux = hwsim.PartSpec{
	Name:    "MUX",
	Inputs:  []string{pA, pB, pSel},
	Outputs: []string{pOut},
	Mount: func(s *hwsim.Socket) hwsim.Updater {
		a, b, sel, out := s.Pin(pA), s.Pin(pB), s.Pin(pSel), s.Pin(pOut)
		return hwsim.UpdaterFn(
			func(clk bool) {
				if sel.Recv(clk) {
					out.Send(clk, b.Recv(clk))
				} else {
					out.Send(clk, a.Recv(clk))
				}
			})
	}}

// DMux returns a demultiplexer.
//
//	Inputs: in, sel
//	Outputs: a, b
//	Function: if sel == 0 { a = in; b = 0 } else { a = 0; b = in }
//
func DMux(w string) hwsim.Part { return dmux.NewPart(w) }

var dmux = hwsim.PartSpec{
	Name:    "DMUX",
	Inputs:  []string{pIn, pSel},
	Outputs: []string{pA, pB},
	Mount: func(s *hwsim.Socket) hwsim.Updater {
		in, sel, a, b := s.Pin(pIn), s.Pin(pSel), s.Pin(pA), s.Pin(pB)
		return hwsim.UpdaterFn(
			func(clk bool) {
				if sel.Recv(clk) {
					a.Send(clk, false)
					b.Send(clk, in.Recv(clk))
				} else {
					a.Send(clk, in.Recv(clk))
					b.Send(clk, false)
				}
			})
	},
}

func muxN(bits int) *hwsim.PartSpec {
	return &hwsim.PartSpec{
		Name:    "Mux" + strconv.Itoa(bits),
		Inputs:  append(bus(bits, pA, pB), pSel),
		Outputs: bus(bits, pOut),
		Mount: func(s *hwsim.Socket) hwsim.Updater {
			a, b, sel := s.Bus(pA, bits), s.Bus(pB, bits), s.Pin(pSel)
			o := s.Bus(pOut, bits)
			return hwsim.UpdaterFn(
				func(clk bool) {
					if sel.Recv(clk) {
						for i, out := range o {
							out.Send(clk, b[i].Recv(clk))
						}
					} else {
						for i, out := range o {
							out.Send(clk, a[i].Recv(clk))
						}
					}
				})
		}}
}

// MuxN returns a N-bits Mux
//
//	Inputs: a[bits], b[bits], sel
//	Outputs: out[bits]
//	Function: for i := range out { if sel == 0 { out[i] = a[i] } else { out[i] = b[i] } }
//
func MuxN(bits int) hwsim.NewPartFn {
	return muxN(bits).NewPart
}

var (
	mux16 = muxN(16)
)

// Mux16 returns a 16-bits Mux
//
//	Inputs: a[16], b[16], sel
//	Outputs: out[16]
//	Function: for i := range out { if sel == 0 { out[i] = a[i] } else { out[i] = b[i] } }
//
func Mux16(c string) hwsim.Part {
	return mux16.NewPart(c)
}

// DMuxN returns a N-bits demultiplexer.
//
//	Inputs: in[bits], sel
//	Outputs: a[bits], b[bits]
//	Function: if sel == 0 { a = in; b = 0 } else { a = 0; b = in }
//
func DMuxN(bits int) hwsim.NewPartFn {
	return (&hwsim.PartSpec{
		Name:    "DMux" + strconv.Itoa(bits),
		Inputs:  append(bus(bits, pIn), pSel),
		Outputs: bus(bits, pA, pB),
		Mount: func(s *hwsim.Socket) hwsim.Updater {
			in, sel, a, b := s.Bus(pIn, bits), s.Pin(pSel), s.Bus(pA, bits), s.Bus(pB, bits)
			return hwsim.UpdaterFn(
				func(clk bool) {
					var si, sf []*hwsim.Pin
					if sel.Recv(clk) {
						sf = a
						si = b
					} else {
						si = a
						sf = b
					}
					for i, ip := range in {
						si[i].Send(clk, ip.Recv(clk))
					}
					for _, o := range sf {
						o.Send(clk, false)
					}
				})
		}}).NewPart
}

func log2(v int) int {
	u := uint(v)
	// log2 of ways
	var log2B = [...]uint{0x2, 0xC, 0xF0, 0xFF00, 0xFFFF0000, 0xFFFFFFFF00000000}
	var log2S = [...]uint{1, 2, 4, 8, 16, 32}
	var l2 uint
	for i := len(log2B) - 1; i >= 0; i-- {
		if u&log2B[i] != 0 {
			v >>= log2S[i]
			l2 |= log2S[i]
		}
	}
	return int(l2)
}

var inputNames = [32]string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n", "o", "p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z", "A", "B", "C", "D", "E", "F"}

// MuxMWayN returns a M-Way N-bits Mux
//
//	Inputs: a[bits], b[bits], ... , z[bits], A[bits], ..., sel[selBits]
//	Outputs: out[bits]
//	Function: for i := range out { if sel == 0 { out[i] = a[i] } else { out[i] = b[i] }... }
//
func MuxMWayN(ways int, bits int) hwsim.NewPartFn {
	if ways > 32 {
		panic("MuxMWayN supports up to 32 ways multiplexers")
	}
	selBits := log2(ways)

	// build inputs array
	inputs := make([]string, ways*bits+int(selBits))
	for w := 0; w < ways; w++ {
		for b := 0; b < bits; b++ {
			inputs[w*bits+b] = inputNames[w] + "[" + strconv.Itoa(b) + "]"
		}
	}
	for w := 0; w < int(selBits); w++ {
		inputs[ways*bits+w] = "sel[" + strconv.Itoa(w) + "]"
	}

	p := &hwsim.PartSpec{
		Name:    "Mux" + strconv.Itoa(ways) + "Way" + strconv.Itoa(bits),
		Inputs:  inputs,
		Outputs: bus(bits, pOut),
		Mount: func(s *hwsim.Socket) hwsim.Updater {
			in := make([]hwsim.Bus, 1<<uint(selBits))
			for i := range in {
				in[i] = s.Bus(inputNames[i], bits)
			}
			sel := s.Bus(pSel, int(selBits))
			out := s.Bus(pOut, bits)
			return hwsim.UpdaterFn(
				func(clk bool) {
					selIn := in[sel.GetInt64(clk)]
					for i, o := range out {
						o.Send(clk, selIn[i].Recv(clk))
					}
				})
		}}
	return p.NewPart
}

// DMuxNWay returns a N-Way demuxer.
//
//	Inputs: in, sel[selBits]
//	Outputs: a, b, ... , z, A, ...
//	Function: if sel[0..selBits] == 0 { a == in } else if sel == 1 { b == in } ...
//
func DMuxNWay(ways int) hwsim.NewPartFn {
	if ways > 32 {
		panic("DMuxNWay supports up to 32 ways demultiplexers")
	}
	selBits := log2(ways)

	p := &hwsim.PartSpec{
		Name:    "DMux" + strconv.Itoa(ways) + "Way",
		Inputs:  append([]string{pIn}, bus(selBits, pSel)...),
		Outputs: inputNames[:ways],
		Mount: func(s *hwsim.Socket) hwsim.Updater {
			in := s.Pin(pIn)
			sel := s.Bus(pSel, selBits)
			outs := make([]*hwsim.Pin, 1<<uint(selBits))
			for i := range outs {
				outs[i] = s.Pin(inputNames[i])
			}
			return hwsim.UpdaterFn(
				func(clk bool) {
					s := int(sel.GetInt64(clk))
					for i, o := range outs {
						if i == s {
							o.Send(clk, in.Recv(clk))
						} else {
							o.Send(clk, false)
						}
					}
				})
		}}
	return p.NewPart
}

// DMuxMWayN returns a N-Way demuxer.
//
//	Inputs: in[bits], sel[selBits]
//	Outputs: a, b, ... , z, A, ...
//	Function: if sel[0..selBits] == 0 { a == in; b,c,d... = 0 } else if sel == 1 { a = 0; b == in; c,d = 0... } ...
//
func DMuxMWayN(ways int, bits int) hwsim.NewPartFn {
	if ways > 32 {
		panic("DMuxMWayN supports up to 32 ways demultiplexers")
	}
	selBits := log2(ways)

	outputs := make([]string, ways*bits)
	for w := 0; w < ways; w++ {
		for b := 0; b < bits; b++ {
			outputs[w*bits+b] = inputNames[w] + "[" + strconv.Itoa(b) + "]"
		}
	}

	p := &hwsim.PartSpec{
		Name:    "DMux" + strconv.Itoa(ways) + "Way",
		Inputs:  append(bus(bits, pIn), bus(selBits, pSel)...),
		Outputs: outputs,
		Mount: func(s *hwsim.Socket) hwsim.Updater {
			in := s.Bus(pIn, bits)
			sel := s.Bus(pSel, selBits)
			outs := make([]hwsim.Bus, 1<<uint(selBits))
			for i := range outs {
				outs[i] = s.Bus(inputNames[i], bits)
			}
			return hwsim.UpdaterFn(
				func(clk bool) {
					selV := int(sel.GetInt64(clk))
					for i, out := range outs {
						if i == selV {
							for bit, o := range out {
								o.Send(clk, in[bit].Recv(clk))
							}
						} else {
							for _, o := range out {
								o.Send(clk, false)
							}
						}
					}
				})
		}}
	return p.NewPart
}
