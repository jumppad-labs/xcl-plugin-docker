package e2e

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInProcessAppliesPlansNoChangesAndDestroys(t *testing.T) {
	outDir := t.TempDir()
	c := newInProcessConfig(t, t.TempDir(), outDir)

	err := c.Apply(basicExample)
	require.NoError(t, err)

	notePath := filepath.Join(outDir, "greeting.txt")

	content, err := os.ReadFile(notePath)
	require.NoError(t, err)
	require.Equal(t, "Hello from xcl", string(content))

	changes, err := c.Diff([]string{basicExample})
	require.NoError(t, err)
	require.Equal(t, 0, changes.Changed())

	err = c.Destroy()
	require.NoError(t, err)

	_, err = os.Stat(notePath)
	require.ErrorIs(t, err, os.ErrNotExist)
}
