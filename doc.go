// Copyright 2018 Denis Bernard <db047h@gmail.com>
// Licensed under the MIT license. See license text in the LICENSE file.

/*
Package hwsim provides the necessary tools to build a virtual CPU using Go as
a hardware description language and run it.

This includes a naive hardware simulator and an API to compose basic components
(logic gates, muxers, etc.) into more complex ones.

The API is designed to mimmic a basic hardware description language and
reduce typing overhead.

The sub-package hwlib provides a library of built-in logic gates as well as some
more advanced components.

The simulation is built around wires that connect components together. While a
Wire can recieve a signal by only one component, it can broadcast that signal to
any number of components (fanout).

The simulation works by updating every half clock cycle the components that
implement the Ticker interface (i.e. that have side effects or somehow "drive"
the circuit, like outputs and clocked data flip-flops). The signals are then
propagated through the simulation by "pulling" them up: calling Recv on a Wire
triggers an update of the component feeding that Wire.

Time in the simulation is simply represented as a boolean value, true during
the call to Circuit.Tick() and false during the call to Circuit.Tock(). Wires
use this information to prevent recursion and provide loop detection.

As a result:

	- Most components ignore the clk argument to Updater.Update(clk bool) and
	  just forward it to the Send/Recv methods of their connected Wires.
	- Most components like logic gates have no propagation delay.
	- Other components like Data Flip-Flops (DFF) have a one clock cycle
	  propagation delay.
	- Direct wire loops are forbidden. Loops must go through a DFF or similar
	  component.

The DFF provided in the hwlib package works like a gated D latch and can be used
as a building block for all sequential components. Its output is considered
stable only during calls to Circuit.Tick(), i.e. when the clk argument of
Updater.Update(clk bool) is true.
*/
package hwsim
