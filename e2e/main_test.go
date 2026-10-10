// Package e2e runs the notes plugin end to end: it applies, plans and destroys
// the sample configuration in examples/basic, with the plugin registered
// in-process and with the plugin built and run as a separate plugin program.
package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jumppad-labs/xcl"
	"github.com/jumppad-labs/xcl/registry"

	notes "github.com/jumppad-labs/xcl-plugin-template"
)

// basicExample is the directory holding the sample configuration the tests
// apply
const basicExample = "../examples/basic"

// noteAddress is the address of the note the sample configuration declares
const noteAddress = "notes.note.greeting"

// pluginBinary is the path of the plugin program TestMain builds for the
// tests in this package
var pluginBinary string

func TestMain(m *testing.M) {
	buildDir, err := os.MkdirTemp("", "xcl-plugin-template-e2e")
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to create build directory: %s\n", err)
		os.Exit(1)
	}

	pluginBinary = filepath.Join(buildDir, "xcl-plugin-template")

	build := exec.Command("go", "build", "-o", pluginBinary, "../cmd/notes")
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr

	if err := build.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "unable to build the plugin program: %s\n", err)
		os.RemoveAll(buildDir)
		os.Exit(1)
	}

	code := m.Run()

	os.RemoveAll(buildDir)
	os.Exit(code)
}

// newInProcessConfig returns a Config with the plugin registered in-process,
// keeping its state in stateDir and writing notes to outDir
func newInProcessConfig(t *testing.T, stateDir string, outDir string) *xcl.Config {
	t.Helper()

	local := registry.NewLocal()
	local.RegisterPlugin(&notes.Plugin{})

	c, err := xcl.NewConfig(
		xcl.WithRegistry(local),
		xcl.WithStatePath(stateDir),
		xcl.WithVariables(map[string]any{"directory": outDir}),
	)
	require.NoError(t, err)

	return c
}

// newExternalConfig returns a Config with the plugin program TestMain built
// registered as an external plugin, keeping its state in stateDir and writing
// notes to outDir
func newExternalConfig(t *testing.T, stateDir string, outDir string) *xcl.Config {
	t.Helper()

	local := registry.NewLocal()
	local.RegisterExternalPlugin(pluginBinary)

	c, err := xcl.NewConfig(
		xcl.WithRegistry(local),
		xcl.WithStatePath(stateDir),
		xcl.WithVariables(map[string]any{"directory": outDir}),
	)
	require.NoError(t, err)

	return c
}
