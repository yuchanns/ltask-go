//go:build linux

package main

import _ "embed"

//go:embed build/.lua/lib/liblua54.so
var lualib []byte
