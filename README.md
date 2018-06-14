# HW (sim)

This package provides the necessary tools to build a virtual CPU using Go as a hardware description language and run it.

This includes a naive hardware simulator and an API to compose basic components (logic gates, muxers, etc.) into more complex ones.

The API is designed to mimmic a basic [Hardware description language][hdl] and reduce typing overhead. As a result it relies heavily on closures for components implemented in Go and can feel a bit awkward in a few places (see Help Wanted below).

> **DISCLAIMER:** This is a self-educational project. I am a software engineer with no electrical engineering background, so please bear with me if some of the terms used are inaccurate or just plain wrong. If you spot any errors, please do not hesitate to file an issue or a PR.

## Implementation details

The simulator is built as an array of individual wires (think [stripboard]). It works like a double-buffer with a "current state" array and a "next state" array. At every tick of the simulation:

- all components are updated:
  - input states are read from the "current state" array
  - output states are written to the "next state" array
- the "current state" and "next state" arrays are swapped.

## Project Status

This is a naive simulation: in a complex system, a lot of gates do not change state at every tick of the simulation because their inputs do not change. For example, a 32 bits ripple-carry is built from 32*5=160 gates and needs about 65 ticks to output the correct result. As the simulation advances, only the last few gates in the input->output chain change state and need to be updated.

While this could be improved, performance is not a major goal right now since it can always be somewhat improved by using cutstom components (with logic written in Go).

The main focus is on the API: bring it in a usable and stable state. Tinkering with the simulation must be fun, not a type typing or useless scaffolding chore. Not all the features I have in mind are implemented yet. TODO's with high priority are listed in the issue tracker. Contributions welcome!

I don't really have plans for any form of GUI yet, so please don't ask.

## Quick tour

### Building a chip

By *chip* I mean these tiny black boxes with silver pins protuding from them that can do all kind of marvelous things in a digital circuit.

A chip is defined by its name, it's input and output pin names, and what circuitry is inside, performing its actual function.

For example, building an XOR gate from a set of NANDs is done like this:

[![XOR gate][imgxor]][xor]

The same in Go with hwsim and the built-in components provided by the hwlib package:

```go
    import (
        // use shorter names for the package imports. It will help your CTS...
        hw "github.com/db47h/hwsim"
        hl "github.com/db47h/hwsim/hwlib"
    )

    xor, err := hw.Chip(
        "XOR",
        "a, b",   // inputs of the created xor gate
        "out",    // outputs
        hl.Nand("a=a,      b=b,      out=nandAB"), // leftmost NAND
        hl.Nand("a=a,      b=nandAB, out=outA"),   // top NAND
        hl.Nand("a=nandAB, b=b,      out=outB"),   // bottom NAND
        hl.Nand("a=outA,   b=outB,   out=out"),    // rightmost NAND
    )
```

The returned `xor` function can then be reused as a part with inputs `a`, `b` and output `out` in another chip design.
Intermediate (chip internal) wires like `nandAB` in the above example can be declared and used on the fly.

### Custom parts

Custom parts are components whose logic is written in Go. Such components can be used to interface with Go code, or simply to improve performance
by rewriting a complex chip in Go (adders, ALUs, RAM).

A custom component is just a struct with custom field tags that implements the Updater interface:

```go
    // sample custom XOR gate

    // xorInstance represents an instance of an xor gate.
    // one such instance is created for every xor gate in a circuit.
    type xorInstance struct {
        A   int `hw:"in"`  // the tag hw:"in" indicates an input pin
        B   int `hw:"in"`
        Out int `hw:"out"` // output pin
    }

    // Update is a Component function that reads the state of the input pins
    // and writes a result to the output pins. It will be called at every
    // tick of the simulation.
    func (g *xorInstance) Update(c *hw.Circuit) {
        a, b := c.Get(g.A), c.Get(g.B)   // get pin states
        c.Set(g.Out, a && !b || !a && b) // set state of output pin
    }

    // Now we turn xorInstance into a part usable in a chip definition.
    // Just pass a nil pointer to an xorInstance to MakePart which will return
    // a *PartSpec
    var xorSpec = hw.MakePart((*xorInstance)(nil))
    // And grab its NewPart method
    var xor = hw.MakePart((*xorInstance)(nil)).NewPart
```

Another way to do it is to create a `PartSpec` struct with some closure magic:

```go
    var xorSpec = &hw.PartSpec{
        Name:    "XOR",
        Inputs:  hw.IO("a, b"),
        Outputs: hw.IO("out"),
        Mount:   func(s *Socket) []hw.Component {
            a, b, out := s.Pin("a"), s.Pin("b"), s.Pin("out")
            return []hw.Component{
                func (c *hw.Circuit) {
                    a, b := c.Get(g.a), c.Get(g.b)
                    c.Set(g.out, a && !b || !a && b)
                }}
            }}
    var xor = xorSpec.NewPart
```

The reflection approach is more readable but consumes more memory.

If defining custom components as functions is preferable, for example in a Go package providing a library of components (where we do not want to export variables):

```go
    func xor(c string) hw.Part { return xorSpec.NewPart(c) }
```

Now we can go ahead and build a half-adder:

```go
    hAdder, _ := hw.Chip(
        "H-ADDER",
        "a, b", // inputs
        "s, c", // output sum and carry
        xor("a=a, b=b, out=s"), // our custom xor gate!
        hw.And("a=a, b=b, out=c"),
    )
```

### Running a simulation

A circuit is made of a set of parts connected together. Time to test our adder:

```go
    var a, b, ci bool
    var s, co bool
    c, err := hw.NewCirtuit(0, 0,
        // feed variables a, b and ci as inputs in the circuit
        hl.Input(func() bool { return a })("out=a"),
        hl.Input(func() bool { return b })("out=b"),
        hl.Input(func() bool { return c })("out=ci"),
        // full adder
        hAdder("a=a,  b=b,  s=s0,  c=c0"),
        hAdder("a=s0, b=ci, s=sum, c=c1"),
        hl.Or(" a=c0, b=c1, out=co"),
        // outputs
        hl.Output(func (bit bool) { s = bit })("in=sum"),
        hl.Output(func (bit bool) { co = bit })("in=co"),
    )
    if err != nil {
        // panic!
    }
    defer c.Dispose()
```

And run it:

```go
    // set inputs
    // ...

    // run for 100 ticks
    for i := 0; i < 100; i++ {
        c.Step()
    }

    // check outputs
    // ...
```

## Help Wanted

A good API has good names with clearly defined entities. This package's API is far from good, with some quirks.

One of those quirks that bothers me is `MountFn` functions. They need to return a slice of `Component` because we want to get all `Components` into a single slice in `Circuit` so that updates can be split evenly between several goroutines (as opposed to organizing Components into a tree). The `PartSpec` returned by `Chip` does not return a component of its own, it simply bundles together the `Component`'s of its parts and pushes that up to its own container. It's mostly invisible until one gets into designing custom components where it adds yet another useless layer of slice creation and indentation just for the sake of that one internal API.

There are a lot of other things that would need some love. If you have any suggestions about naming or other API changes that would make everyone's life easier, feel free to file an issue or open a PR!

## License

Copyright 2018 Denis Bernard <db047h@gmail.com>

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

[hdl]: https://en.wikipedia.org/wiki/Hardware_description_language
[imgxor]: https://upload.wikimedia.org/wikipedia/commons/f/fa/XOR_from_NAND.svg
[xor]: https://en.wikipedia.org/wiki/NAND_logic#XOR
[stripboard]: https://en.wikipedia.org/wiki/Stripboard