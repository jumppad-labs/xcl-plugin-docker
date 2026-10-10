package e2e

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jumppad-labs/xcl"
	"github.com/jumppad-labs/xcl/diff"
)

// renamedNote is the sample configuration with the note's name changed
const renamedNote = `
variable "directory" {
  default = "./out"
}

notes "note" "greeting" {
  directory = variable.directory
  name      = "welcome.txt"
  content   = "Hello from xcl"
}
`

// rewordedNote is the sample configuration with the note's content changed
const rewordedNote = `
variable "directory" {
  default = "./out"
}

notes "note" "greeting" {
  directory = variable.directory
  name      = "greeting.txt"
  content   = "Goodbye from xcl"
}
`

// applyBasicExampleExternally applies the sample configuration through the
// plugin program, keeping state in stateDir and writing notes to outDir. What
// it applied is destroyed when the test ends.
func applyBasicExampleExternally(t *testing.T, stateDir string, outDir string) *xcl.Config {
	t.Helper()

	c := newExternalConfig(t, stateDir, outDir)

	// register the destroy before applying, so a partly applied
	// configuration is removed too
	t.Cleanup(func() {
		require.NoError(t, c.Destroy())
	})

	err := c.Apply(basicExample)
	require.NoError(t, err)

	return c
}

// writeConfiguration writes content as main.xcl in a new temporary directory
// and returns the directory
func writeConfiguration(t *testing.T, content string) string {
	t.Helper()

	dir := t.TempDir()

	err := os.WriteFile(filepath.Join(dir, "main.xcl"), []byte(content), 0o644)
	require.NoError(t, err)

	return dir
}

// noteAction returns the action the diff plans for the sample's note, failing
// the test when the diff does not list it
func noteAction(t *testing.T, changes *diff.Diff) diff.Action {
	t.Helper()

	for _, resource := range changes.Resources {
		if resource.Address == noteAddress {
			return resource.Action
		}
	}

	require.Failf(t, "note not in diff", "the diff does not list %s: %+v", noteAddress, changes.Resources)

	return ""
}

func TestPlanReplacesTheNoteWhenItsNameChanges(t *testing.T) {
	stateDir := t.TempDir()
	outDir := t.TempDir()
	applyBasicExampleExternally(t, stateDir, outDir)

	planner := newExternalConfig(t, stateDir, outDir)

	changes, err := planner.Diff([]string{writeConfiguration(t, renamedNote)})
	require.NoError(t, err)
	require.Equal(t, diff.ActionReplace, noteAction(t, changes))
}

func TestPlanUpdatesTheNoteWhenItsContentChanges(t *testing.T) {
	stateDir := t.TempDir()
	outDir := t.TempDir()
	applyBasicExampleExternally(t, stateDir, outDir)

	planner := newExternalConfig(t, stateDir, outDir)

	changes, err := planner.Diff([]string{writeConfiguration(t, rewordedNote)})
	require.NoError(t, err)
	require.Equal(t, diff.ActionUpdate, noteAction(t, changes))
}
