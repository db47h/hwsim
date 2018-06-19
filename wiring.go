// Copyright 2018 Denis Bernard <db047h@gmail.com>
// Licensed under the MIT license. See license text in the LICENSE file.

package hwsim

// Constant input pin names. These pins can only be connected to the input pins of a part.
//
// Those are reserved names and should not be used as input or output names in
// custom chips.
//
var (
	False       = "false" // always false input
	True        = "true"  // alwyas true input
	Clk         = "clk"   // clock signal. True during Tick, False during Tock.
	cstPinNames = [...]string{"false", "true", "clk"}
)

const (
	cstFalse = iota
	cstTrue
	cstClk
	cstCount
)

// A Pin is a component's output pin
//
type Pin struct {
	src   Updater
	clk   bool
	value bool
}

// Connect sets the Updater as the connector's source.
//
func (c *Pin) Connect(u Updater) {
	c.src = u
}

// Send sends a signal a time clk.
//
func (c *Pin) Send(clk bool, value bool) {
	if c.clk != clk {
		c.clk, c.value = clk, value
	}
}

// Recv recieves a signal at time clk.
// It may trigger an update of the source component.
//
func (c *Pin) Recv(clk bool) bool {
	if c.clk != clk {
		c.src.Update(clk) // should trigger a send
	}
	return c.value
}

// A Connection represents a connection between the pin PP of a part and
// the pins CP in its host chip.
//
type Connection struct {
	PP string
	CP []string
}
