package hdl

// CHIP Xor {
// 	IN a, b;
// 	OUT out;
// 	PARTS:
// 	Not(in=a, out=nota);
// 	Not(in=b, out=notb);
// 	And(a=a, b=notb, out=w1);
// 	And(a=nota, b=b, out=w2);
// 	Or(a=w1, b=w2, out=out);
// }

// CHIP Mux16 {
// 	IN a[16], a[16], sel;
// 	OUT out[16];
// 	BUILTIN Mux; // Reference to builtIn/Mux.class, that
// 	// implements both the Mux.hdl and the
// 	// Mux16.hdl built-in chips.
// }

// An Updater is an updatable component in a circuit
//
type Updater interface {
	Update(c *Circuit)
}

// A Chip encapsulates the definition of set of logic components in a circuit.
//
type Chip interface {
	// Pinout returns the chip's external pinout.
	Pinout() []string
	// Build creates a new instance of a chip as an Updater slice.
	// The provided pin numbers are the pin numbers for the external pins of the chip.
	// TODO: review all implementation for proper error messages.
	//
	Build(pins []int, c *Circuit) ([]Updater, error)
}

// custom chips
type chip struct {
	ep    []string // external pins.
	pins  []string
	parts []Chip
}

func (c *chip) Pinout() []string {
	return c.ep
}

func (c *chip) Build(pins []int, cc *Circuit) ([]Updater, error) {
	var updaters []Updater
	// map internal pin names to allocated pin #'s
	pmap := make(map[string]int)
	// the provided pins are pre-allocated pin #'s for our external pins
	for pindex, pnum := range pins {
		pmap[c.pins[pindex]] = pnum
	}
	// collect
	for _, p := range c.parts {
		// compute external pins for each part
		pinout := p.Pinout()
		ppins := make([]int, len(pinout))
		for pnum, pname := range pinout {
			var n int
			var ok bool
			if n, ok = pmap[pname]; !ok {
				n = cc.Alloc()
				pmap[pname] = n
			}
			ppins[pnum] = n
		}
		// build part
		pup, err := p.Build(ppins, cc)
		if err != nil {
			return nil, err
		}
		updaters = append(updaters, pup...)
	}
	return updaters, nil
}

// A NewChipFunc is a function that takes a number of named pins and returns a new Chip.
//
type NewChipFunc func(pins ...string) Chip

// NewChip creates a new Chip based on other Chips.
//
// An Xor gate could be created like this:
//
//	xor := NewChip(
//		[]string{"a", "b"},
//		[]string{"out"},
//		[]Chip{
//			Not("a", "nota"),
//			Not("b", "notb"),
//			And("a", "notb", "w1"),
//			And("b", "nota", "w2"),
//			Or("w1", "w2", "out")
//		})
//
func NewChip(inputs []string, outputs []string, parts []Chip) NewChipFunc {
	return NewChipFunc(func(pins ...string) Chip {
		ip := make([]string, len(inputs)+len(outputs))
		copy(ip, inputs)
		copy(ip[len(inputs):], outputs)
		return &chip{ep: pins, pins: ip, parts: parts}
	})
}

// Circuit is a runable circuit simulation.
//
type Circuit struct {
	s0    []bool
	s1    []bool
	parts []Updater
	top   int
}

// NewCircuit returns a new circuit based on the given chips.
//
func NewCircuit(cs []Chip) (*Circuit, error) {
	var parts []Updater
	pins := make(map[string]int)
	cc := new(Circuit)
	for _, c := range cs {
		pinout := c.Pinout()
		cpins := make([]int, len(pinout))
		for pnum, pname := range pinout {
			var n int
			var ok bool
			// check if pin already allocated
			if n, ok = pins[pname]; !ok {
				n = cc.Alloc()
				pins[pname] = n
			}
			cpins[pnum] = n
		}
		cparts, err := c.Build(cpins, cc)
		if err != nil {
			return nil, err
		}
		parts = append(parts, cparts...)
	}
	cc.s0 = make([]bool, cc.top)
	cc.s1 = make([]bool, cc.top)
	cc.parts = parts
	return cc, nil
}

// Alloc allocates a pin and returns its number.
//
func (c *Circuit) Alloc() int {
	top := c.top
	c.top++
	return top
}

// Get returns the state of a given input/output.
//
func (c *Circuit) Get(n int) bool {
	return c.s0[n]
}

// Set sets the state of a given input/output.
//
func (c *Circuit) Set(n int, s bool) {
	c.s1[n] = s
}

// Update advances a chip simulation by one step.
//
func (c *Circuit) Update() {
	// We can split this accross goroutines
	for _, u := range c.parts {
		u.Update(c)
	}
	c.s0, c.s1 = c.s1, c.s0
}
