// Copyright 2018 Denis Bernard <db047h@gmail.com>
// Licensed under the MIT license. See license text in the LICENSE file.

package hwsim

import (
	"strconv"
)

// Input returns a 1 bit input.
//
func Input(f func() bool) NewPartFn {
	p := &PartSpec{
		Name:    "in",
		Inputs:  nil,
		Outputs: []string{"out"},
		Mount: func(s *Socket) Updater {
			out := s.Pin("out")
			return UpdaterFn(func(clk bool) {
				out.Send(clk, f())
			})
		}}
	return p.NewPart
}

// Output returns a 1 bit output.
//
func Output(f func(value bool)) NewPartFn {
	p := &PartSpec{
		Name:    "out",
		Inputs:  []string{"in"},
		Outputs: nil,
		Mount: func(s *Socket) Updater {
			out := s.Pin("in")
			return TickerFn(
				func(clk bool) {
					f(out.Recv(clk))
				})
		}}
	return p.NewPart
}

// InputN creates an input bus of the given bits size.
//
func InputN(bits int, f func() int64) NewPartFn {
	bs := strconv.Itoa(bits)
	return (&PartSpec{
		Name:    "Input" + bs,
		Inputs:  nil,
		Outputs: IO("out[" + bs + "]"),
		Mount: func(s *Socket) Updater {
			pins := s.Bus("out", bits)
			return UpdaterFn(
				func(clk bool) {
					pins.SetInt64(clk, f())
				})
		}}).NewPart
}

// OutputN creates an output bus of the given bits size.
//
func OutputN(bits int, f func(int64)) NewPartFn {
	bs := strconv.Itoa(bits)
	return (&PartSpec{
		Name:    "Output" + bs,
		Inputs:  IO("in[" + bs + "]"),
		Outputs: nil,
		Mount: func(s *Socket) Updater {
			pins := s.Bus("in", bits)
			return TickerFn(
				func(clk bool) {
					f(pins.GetInt64(clk))
				})
		}}).NewPart
}
