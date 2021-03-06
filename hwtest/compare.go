// Copyright 2018 Denis Bernard <db047h@gmail.com>
// Licensed under the MIT license. See license text in the LICENSE file.

// Package hwtest provides utility functions for testing circuits.
//
package hwtest

import (
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/db47h/hwsim"
)

func connString(in, out []string) string {
	var b strings.Builder
	for _, n := range in {
		if b.Len() > 0 {
			b.WriteRune(',')
		}
		b.WriteString(n)
		b.WriteRune('=')
		b.WriteString(n)
	}
	for _, n := range out {
		if b.Len() > 0 {
			b.WriteRune(',')
		}
		b.WriteString(n)
		b.WriteRune('=')
		b.WriteString(n)
	}
	return b.String()
}

func pinList(in []string) string {
	bus := make(map[string]int)
	var pins []string

	for _, n := range in {
		if b := strings.IndexRune(n, '['); b >= 0 {
			bn := n[:b]
			idx, err := strconv.Atoi(n[b+1 : strings.IndexRune(n, ']')])
			if err != nil {
				panic(err)
			}
			if bidx, ok := bus[bn]; !ok || bidx < idx {
				bus[bn] = idx
			}
		} else {
			pins = append(pins, n)
		}
	}

	var b strings.Builder
	for k, n := range bus {
		if b.Len() > 0 {
			b.WriteRune(',')
		}
		b.WriteString(k)
		b.WriteRune('[')
		b.WriteString(strconv.Itoa(n + 1))
		b.WriteRune(']')
	}
	for _, n := range pins {
		if b.Len() > 0 {
			b.WriteRune(',')
		}
		b.WriteString(n)
	}
	return b.String()
}

func randBool() bool {
	return rand.Int63()&(1<<62) != 0
}

// ComparePart takes two parts and compares their outputs given the same inputs.
// Both parts must have the same Input/Output interface.
//
func ComparePart(t *testing.T, part1 hwsim.NewPartFn, part2 hwsim.NewPartFn) {
	t.Helper()

	rand.Seed(time.Now().UnixNano())

	ps1, ps2 := part1(""), part1("")
	conns := connString(ps1.Inputs, ps1.Outputs)
	ps1, ps2 = part1(conns), part2(conns)

	sort.Strings(ps1.Inputs)
	sort.Strings(ps1.Outputs)
	sort.Strings(ps2.Inputs)
	sort.Strings(ps2.Outputs)

	inputs := make([]bool, len(ps1.Inputs))
	outputs := make([][2]bool, len(ps1.Outputs))

	// build two wrappers with their own set of outputs
	parts1 := []hwsim.Part{ps1}
	for i, o := range ps1.Outputs {
		n := i
		parts1 = append(parts1, hwsim.Output(func(b bool) { outputs[n][0] = b })("in="+o))
	}
	parts2 := []hwsim.Part{ps2}
	for i, o := range ps2.Outputs {
		n := i
		parts2 = append(parts2, hwsim.Output(func(b bool) { outputs[n][1] = b })("in="+o))
	}
	w1, err := hwsim.Chip("wrapper1", pinList(ps1.Inputs), "", parts1...)
	if err != nil {
		t.Fatal(err)
	}
	w2, err := hwsim.Chip("wrapper2", pinList(ps2.Inputs), "", parts2...)
	if err != nil {
		t.Fatal(err)
	}

	// compare specs
	if len(ps1.Inputs) != len(ps2.Inputs) {
		t.Fatal("len(ps1.Inputs) != len(ps2.Inputs)")
	}
	if len(ps1.Outputs) != len(ps2.Outputs) {
		t.Fatal("len(ps1.Outputs) != len(ps2.Outputs)")
	}
	for i := range ps1.Inputs {
		if ps1.Inputs[i] != ps2.Inputs[i] {
			t.Fatalf("ps1.Inputs[i] = %q != ps2.Inputs[i] = %q", ps1.Inputs[i], ps2.Inputs[i])
		}
	}
	for i := range ps1.Outputs {
		if ps1.Outputs[i] != ps2.Outputs[i] {
			t.Fatalf("ps1.Outputs[i] = %q != ps2.Outputs[i] = %q", ps1.Outputs[i], ps2.Outputs[i])
		}
	}

	var parts []hwsim.Part
	for i, n := range ps1.Inputs {
		k := i
		parts = append(parts, hwsim.Input(func() bool { return inputs[k] })("out="+n))
	}
	cstr := connString(ps1.Inputs, nil)
	parts = append(parts, w1(cstr), w2(cstr))

	c, err := hwsim.NewCircuit(parts...)
	if err != nil {
		t.Fatal(err)
	}

	errString := func(oname string, ex, got bool) string {
		var b strings.Builder
		for i, n := range ps1.Inputs {
			if b.Len() > 0 {
				b.WriteString(", ")
			}
			b.WriteString(n)
			b.WriteRune('=')
			if inputs[i] {
				b.WriteString("true")
			} else {
				b.WriteString("false")
			}
		}
		for i, n := range ps1.Outputs {
			if b.Len() > 0 {
				b.WriteString(", ")
			}
			b.WriteString(n)
			b.WriteRune('=')
			if outputs[i][0] {
				b.WriteString("true")
			} else {
				b.WriteString("false")
			}
		}
		return fmt.Sprintf("\nExpected %s => %s=%v\nGot %v", b.String(), oname, ex, got)
	}

	// try all 0
	c.TickTock()
	for o, out := range outputs {
		if out[0] != out[1] {
			t.Fatal(errString(ps1.Outputs[o], out[0], out[1]))
		}
	}

	// try all 1
	for in := range inputs {
		inputs[in] = true
	}
	c.TickTock()
	for o, out := range outputs {
		if out[0] != out[1] {
			t.Fatal(errString(ps1.Outputs[o], out[0], out[1]))
		}
	}

	iter := len(ps1.Inputs)
	const maxBits = 12
	if iter > maxBits {
		iter = 1 << maxBits
		// random testing
		for i := 0; i < iter; i++ {
			for in := range inputs {
				inputs[in] = randBool()
			}
			c.Tick()
			for o, out := range outputs {
				if out[0] != out[1] {
					t.Fatal(errString(ps1.Outputs[o], out[0], out[1]))
				}
			}
			c.Tock()
			for o, out := range outputs {
				if out[0] != out[1] {
					t.Fatal(errString(ps1.Outputs[o], out[0], out[1]))
				}
			}
		}
	} else {
		// try all inputs
		iter = 1 << uint(iter)
		for i := 0; i < iter; i++ {
			for in := range inputs {
				inputs[in] = i&(1<<uint(in)) != 0
			}
			c.Tick()
			for o, out := range outputs {
				if out[0] != out[1] {
					t.Fatal(errString(ps1.Outputs[o], out[0], out[1]))
				}
			}
			c.Tock()
			for o, out := range outputs {
				if out[0] != out[1] {
					t.Fatal(errString(ps1.Outputs[o], out[0], out[1]))
				}
			}
		}
	}
}
