package hwlib

import (
	hw "github.com/db47h/hwsim"
)

// Input returns a 1-bit input updated with the value of i.
//
func Input(i *bool) hw.NewPartFn {
	return hw.Input(func() bool { return *i })
}

// Output returns a 1-bit output that updates o.
//
func Output(o *bool) hw.NewPartFn {
	return hw.Output(func(v bool) { *o = v })
}

// Input16 returns a 16-bit input updated with the value of i.
//
func Input16(i *uint16) hw.NewPartFn {
	return hw.InputN(16, func() uint64 { return uint64(*i) })
}

// Output16 returns a 16-bit output that updates o.
//
func Output16(o *uint16) hw.NewPartFn {
	return hw.OutputN(16, func(v uint64) { *o = uint16(v) })
}

// Input32 returns a 32-bit input updated with the value of i.
//
func Input32(i *uint32) hw.NewPartFn {
	return hw.InputN(32, func() uint64 { return uint64(*i) })
}

// Output32 returns a 32-bit output that updates o.
//
func Output32(o *uint32) hw.NewPartFn {
	return hw.OutputN(32, func(v uint64) { *o = uint32(v) })
}

// Input64 returns a 64-bit input updated with the value of i.
//
func Input64(i *uint64) hw.NewPartFn {
	return hw.InputN(64, func() uint64 { return uint64(*i) })
}

// Output64 returns a 64-bit output that updates o.
//
func Output64(o *uint64) hw.NewPartFn {
	return hw.OutputN(64, func(v uint64) { *o = uint64(v) })
}
