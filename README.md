# HW (sim)

This package provides the necessary tools to build a virtual CPU using Go as a hardware description language and run it.

This includes a naive hardware simulator and an API to compose basic components (logic gates, muxers, etc.) into more complex ones.

The API is designed to mimmic a basic [Hardware description language][hdl] and reduce typing overhead.

> **DISCLAIMER:** This is a self-educational project. I have no electrical engineering background, so please bear with me if some of the terms used are inaccurate or just plain wrong. If you spot any errors, please do not hesitate to file an issue or a PR.

## Implementation details

The simulation is built around wires that connect components together. While a wire can recieve a signal by only one component, it can broadcast that signal to any number of components (fanout).

The simulation works by updating every half clock cycle the components that implement the Ticker interface (i.e. that have side effects or somehow "drive" the circuit, like outputs and clocked data flip-flops). The signals are then propagated through the simulation by "pulling" them up: calling Recv on a Wire triggers an update of the component feeding that Wire.

Time in the simulation is simply represented as a boolean value: `false` during the call to `Circuit.Tick()` and `true` during the call to `Circuit.Tock()`. Wires use this information to prevent recursion and provide loop detection.

As a result:

- most components like logic gates have no propagation delay
- other components like Data Flip-Flops have a one clock cycle propagation delay.
- direct wire loops are forbidden. Loops must go through a DFF or similar component.

The DFF provided in the hwlib package works like a [gated D latch][gated D latch] and can be used as a building block for all sequential components. Its output is considered stable only during calls to `Circuit.Tick()`, i.e. when the `clk` argument of `Updater.Update(clk bool)` is true.

## Project Status

The simulation implementation is sub-optimal: a push model would allow updating only a few components at every clock cycle (those for which inputs have changed). This would however make the implementation much more complex as components would need to wait for all input signals to be updated before updating themselves. Additionally, performance is not a major goal right now since it can always be somewhat improved by using cutstom components (with logic written in Go).

The main focus is on the API: bring it in a usable and stable state. Tinkering with the simulation must be fun, not a type typing fest or a code scaffolding chore. Not all the features I have in mind are implemented yet. TODO's with high priority are listed in the issue tracker. Contributions welcome!

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
        A   *hw.Wire `hw:"in"`  // the tag hw:"in" indicates an input pin
        B   *hw.Wire `hw:"in"`
        Out *hw.Wire `hw:"out"` // output pin
    }

    // Update is a Component function that reads the state of the inputs
    // and sends a result to the outputs. It will be called at every
    // half clock cycle of the simulation.
    func (g *xorInstance) Update(clk bool) {
        a, b := g.A.Recv(clk), g.B.Recv(clk) // get input signals
        g.Out.Send(clk, a && !b || !a && b)  // send output
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
        Mount:   func(s *hw.Socket) hw.Updater {
            a, b, out := s.Wire("a"), s.Wire("b"), s.Wire("out")
            return hw.UpdaterFn(
                func (clk bool) {
                    a, b := g.A.Recv(clk), g.B.Recv(clk)
                    g.Out.Send(clk, a && !b || !a && b)
                })
            }}
    var xor = xorSpec.NewPart
```

The reflection approach is more readable but is also more resource hungry.

If defining custom components as functions is preferable, for example in a Go package providing a library of components (where we do not want to export variables):

```go
    func Xor(c string) hw.Part { return xorSpec.NewPart(c) }
```

Now we can go ahead and build a half-adder:

```go
    hAdder, _ := hw.Chip(
        "Half-Adder",
        "a, b",                 // inputs
        "s, c",                 // output sum and carry
        xor("a=a, b=b, out=s"), // our custom xor gate!
        hw.And("a=a, b=b, out=c"),
    )
```

### Running a simulation

A circuit is made of a set of parts connected together. Time to test our adder:

```go
    var a, b, ci bool
    var s, co bool
    c, err := hw.NewCirtuit(
        // feed variables a, b and ci as inputs in the circuit
        hw.Input(func() bool { return a })("out=a"),
        hw.Input(func() bool { return b })("out=b"),
        hw.Input(func() bool { return c })("out=ci"),
        // full adder
        hAdder("a=a,  b=b,  s=s0,  c=c0"),
        hAdder("a=s0, b=ci, s=sum, c=c1"),
        hl.Or(" a=c0, b=c1, out=co"),
        // outputs
        hw.Output(func (bit bool) { s = bit })("in=sum"),
        hw.Output(func (bit bool) { co = bit })("in=co"),
    )
    if err != nil {
        // panic!
    }
    defer c.Dispose()
```

And run it:

```go
    // set inputs
    a, b, ci = false, true, false

    // run a single clock cycle
    c.TickTock()

    // check outputs
    if s != true && co != false {
        // bug
    }
```

## Contributing

A good API has good names with clearly defined entities. This package's API is far from good, with some quirks.

The whole `Socket` thing, along with the wiring mess in `Chip()`, are remnants of a previous implementation and are overly complex. They will probably be dusted off when I get to implement static loop detection. Until then, it works and doesn't affect the performance of the simulation, so it's not top priority. It won't have a major impact on the API either since `Socket` will remain, possibly renamed, but as an interface with the same API.

If you have any suggestions about naming or other API changes that would make everyone's life easier, feel free to file an issue or open a PR!

## License

Copyright 2018 Denis Bernard <db047h@gmail.com>

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

[hdl]: https://en.wikipedia.org/wiki/Hardware_description_language
[imgxor]: https://upload.wikimedia.org/wikipedia/commons/f/fa/XOR_from_NAND.svg
[xor]: https://en.wikipedia.org/wiki/NAND_logic#XOR
[stripboard]: https://en.wikipedia.org/wiki/Stripboard
[gated D latch]: https://en.wikipedia.org/wiki/Flip-flop_(electronics)#Gated_D_latch