// Copyright 2018 Denis Bernard <db047h@gmail.com>
// Licensed under the MIT license. See license text in the LICENSE file.

package hwlib

import (
	"strconv"

	"github.com/db47h/hwsim"
)

// Input creates a function based input.
//
//	Outputs: out
//	Function: out = f()
//
func Input(f func() bool) hwsim.NewPartFn {
	p := &hwsim.PartSpec{
		Name:    "Input",
		Inputs:  nil,
		Outputs: []string{pOut},
		Mount: func(s *hwsim.Socket) []hwsim.Updater {
			pin := s.Pin(pOut)
			return hwsim.UpdaterFn(func(c *hwsim.Circuit) { c.Set(pin, f()) })
		}}
	return p.NewPart
}

// Output creates an output or probe. The fn function is
// called with the named pin state on every circuit update.
//
//	Inputs: in
//	Function: f(in)
//
func Output(f func(bool)) hwsim.NewPartFn {
	p := &hwsim.PartSpec{
		Name:    "Output",
		Inputs:  []string{pIn},
		Outputs: nil,
		Mount: func(s *hwsim.Socket) []hwsim.Updater {
			in := s.Pin(pIn)
			return hwsim.UpdaterFn(func(c *hwsim.Circuit) { f(c.Get(in)) })
		}}
	return p.NewPart
}

// InputN creates an input bus of the given bits size.
//
func InputN(bits int, f func() int64) hwsim.NewPartFn {
	return (&hwsim.PartSpec{
		Name:    "INPUT" + strconv.Itoa(bits),
		Inputs:  nil,
		Outputs: bus(bits, pOut),
		Mount: func(s *hwsim.Socket) []hwsim.Updater {
			pins := s.Bus(pOut, bits)
			return hwsim.UpdaterFn(func(c *hwsim.Circuit) {
				c.SetInt64(pins, f())
			})
		}}).NewPart
}

// OutputN creates an output bus of the given bits size.
//
func OutputN(bits int, f func(int64)) hwsim.NewPartFn {
	return (&hwsim.PartSpec{
		Name:    "OUTPUTBUS" + strconv.Itoa(bits),
		Inputs:  bus(bits, pIn),
		Outputs: nil,
		Mount: func(s *hwsim.Socket) []hwsim.Updater {
			pins := s.Bus(pIn, bits)
			return hwsim.UpdaterFn(func(c *hwsim.Circuit) {
				f(c.GetInt64(pins))
			})
		}}).NewPart
}
