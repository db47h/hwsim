package hwsim

// Constant input pin names.
//
var (
	False = "false"
	True  = "true"
	GND   = "false"
	Clk   = "clk"
)

const (
	cstFalse = iota
	cstTrue
	cstClk
	cstCount
)

// A Socket maps a part's pin names to pin numbers in a circuit.
//
type Socket struct {
	m map[string]int
	c *Circuit
}

func newSocket(c *Circuit) *Socket {
	return &Socket{
		m: make(map[string]int),
		c: c,
	}
}

// Pin returns the pin number allocated to the given pin name.
// This function returns the "False" pin if the requested name was not found.
//
func (s *Socket) Pin(name string) int {
	return s.m[name]
}

// PinOrNew returns the pin number allocated to the given pin name.
// If no such pin exists a new one is allocated.
//
func (s *Socket) PinOrNew(name string) int {
	n, ok := s.m[name]
	if !ok {
		switch name {
		case Clk:
			n = cstClk
		case False:
			n = cstFalse
		case True:
			n = cstTrue
		default:
			n = s.c.allocPin()
		}
		s.m[name] = n
	}
	return n
}

// Bus returns the pin numbers allocated to the given bus name.
//
func (s *Socket) Bus(name string, size int) []int {
	out := make([]int, size)
	for i := range out {
		out[i] = s.m[BusPinName(name, i)]
	}
	return out
}
