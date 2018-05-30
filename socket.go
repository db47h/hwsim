package hdl

// Constant input pin names.
//
var (
	True  = "true"
	False = "false"
	GND   = "false"
)

const (
	cstFalse = iota
	cstTrue
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
		m: map[string]int{False: cstFalse, True: cstTrue},
		c: c,
	}
}

// Mount mounts the given sub-part and allocates new internal pins as necessary
// (according to pin mappings in p.Wires()).
//
func (s *Socket) Mount(p Part) []Component {
	// sub-socket for p
	sub := newSocket(s.c)
	for k, v := range p.Wires() {
		sub.m[k] = s.PinOrNew(v)
	}
	return p.Spec().Mount(sub)
}

// Pin returns the pin number allocated to the given pin name.
// This function panics if the pin does not exist.
//
func (s *Socket) Pin(name string) int {
	n, ok := s.m[name]
	if !ok {
		panic("pin " + name + " does not exist")
	}
	return n
}

// PinOrNew returns the pin number allocated to the given pin name.
// If no such pin exists a new one is allocated.
//
func (s *Socket) PinOrNew(name string) int {
	n, ok := s.m[name]
	if !ok {
		n = s.c.allocPin()
		s.m[name] = n
	}
	return n
}

// Bus returns the pin numbers allocated to the given bus name.
//
func (s *Socket) Bus(name string) []int {
	out := make([]int, 0)
	i := 0
	for {
		n, ok := s.m[BusPinName(name, i)]
		if !ok {
			break
		}
		out = append(out, n)
		i++
	}
	if len(out) == 0 {
		panic("bus " + name + " does not exist")
	}
	return out
}
