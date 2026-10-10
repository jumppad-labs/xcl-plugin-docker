// Command notes serves the notes plugin as a separate plugin program. xcl
// starts this binary and talks to it over gRPC; the plugin it serves is the
// same notes.Plugin an application can register in-process.
//
// Build it with `make build`, which writes build/<plugin name>.
package main

import (
	"github.com/jumppad-labs/xcl/plugins"

	notes "github.com/jumppad-labs/xcl-plugin-template"
)

// main serves the plugin to the host that started this process
func main() {
	plugins.Serve(&notes.Plugin{})
}
