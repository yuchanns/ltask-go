//go:build darwin

package main

import _ "embed"

//go:embed build/.lua/lib/liblua54.dylib
var lualib []byte
