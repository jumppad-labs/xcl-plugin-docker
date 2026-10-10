// Package stateonly reads what the notes plugin applied as an application
// that imports only the entities package would: it never imports the plugin,
// its providers or its clients, only the block type.
package stateonly

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jumppad-labs/xcl"
	"github.com/jumppad-labs/xcl/registry"

	"github.com/jumppad-labs/xcl-plugin-template/entities"
)

// basicExample is the directory holding the sample configuration the test
// applies
const basicExample = "../../examples/basic"

// pluginBinary is the path of the plugin program TestMain builds for the
// tests in this package
var pluginBinary string

func TestMain(m *testing.M) {
	buildDir, err := os.MkdirTemp("", "xcl-plugin-template-stateonly")
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to create build directory: %s\n", err)
		os.Exit(1)
	}

	pluginBinary = filepath.Join(buildDir, "xcl-plugin-template")

	build := exec.Command("go", "build", "-o", pluginBinary, "../../cmd/notes")
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

// newConfig returns a Config with the plugin program TestMain built
// registered as an external plugin, keeping its state in stateDir and writing
// notes to outDir
func newConfig(t *testing.T, stateDir string, outDir string) *xcl.Config {
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

func TestStateReaderReadsAppliedNotesIntoTheEntityType(t *testing.T) {
	stateDir := t.TempDir()
	outDir := t.TempDir()

	applier := newConfig(t, stateDir, outDir)

	// register the destroy before applying, so a partly applied
	// configuration is removed too
	t.Cleanup(func() {
		require.NoError(t, applier.Destroy())
	})

	err := applier.Apply(basicExample)
	require.NoError(t, err)

	// a separate Config on the same state, as another program reading what
	// was applied would be
	reader := newConfig(t, stateDir, outDir)

	err = reader.Load()
	require.NoError(t, err)

	found, err := xcl.FindByType[entities.Note](reader, "notes", "note")
	require.NoError(t, err)
	require.Len(t, found, 1)

	note := found[0]
	require.Equal(t, "greeting.txt", note.Name)
	require.Equal(t, "Hello from xcl", note.Content)
	require.Equal(t, filepath.Join(outDir, "greeting.txt"), note.Path)
}
