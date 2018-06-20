/*
Package hwsim provides the necessary tools to build a virtual CPU using Go as
a hardware description language and run it.

This includes a naive hardware simulator and an API to compose basic components
(logic gates, muxers, etc.) into more complex ones.

The API is designed to mimmic a basic hardware description language and
reduce typing overhead.

The sub-package hwlib provides a library of built-in logic gates as well as some
more advanced components.

Copyright 2018 Denis Bernard <db047h@gmail.com>

This package is licensed under the MIT license. See license text in the LICENSE file.
*/
package hwsim
