/*
Package hwsim provides the necessary tools to build a virtual CPU using Go as
a hardware description language and run it.

This includes a naive hardware simulator and an API to compose basic components
(logic gates, muxers, etc.) into more complex ones.

The API is designed to mimmic a basic [Hardware description language][hdl] and
reduce typing overhead. As a result it relies heavily on closures for components
implemented in Go and can feel a bit awkward in a few places.

The sub-package hwlib provides a library of built-in logic gates as well as some
more advanced components.
*/
package hwsim
