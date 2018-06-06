/*
Package hwsim provides the necessary tools to build a virtual CPU using Go as
a hardware description language and run it.

This includes a naive hardware simulator and an API to compose basic components
(logic gates, muxers, etc.) into more complex ones.

The API is designed to mimmic a real [Hardware description language][hdl]. As a
result, it relies heavily on closures and can feel a bit awkward when
implementing custom components.

*/
package hwsim
