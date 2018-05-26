package main

import (
	"log"

	"github.com/db47h/hdl"
)

func main() {
	// an xor gate
	xor := hdl.NewChip(
		[]string{"a", "b"},
		[]string{"out"},
		[]hdl.Chip{
			hdl.Not("a", "nota"),
			hdl.Not("b", "notb"),
			hdl.And("a", "notb", "w1"),
			hdl.And("b", "nota", "w2"),
			hdl.Or("w1", "w2", "out"),
		})
	var a, b bool

	c, err := hdl.NewCircuit([]hdl.Chip{
		hdl.Input("a", func() bool { return a }),
		hdl.Input("b", func() bool { return b }),
		hdl.Output("out", func(o bool) {
			log.Print("out: ", o)
		}),
		xor("a", "b", "out"),
	})
	if err != nil {
		panic(err)
	}
	for i := 0; i < 10; i++ {
		c.Update(6)
	}
}
