package notes

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jumppad-labs/xcl/logger"
)

func TestInitRegistersTheNoteType(t *testing.T) {
	directory := t.TempDir()
	t.Chdir(directory)

	p := &Plugin{}

	err := p.Init(logger.Nop(), nil)
	require.NoError(t, err)

	registered := p.GetTypes()
	require.Len(t, registered, 1)
	require.Equal(t, "notes", registered[0].Type)
	require.Equal(t, "note", registered[0].SubType)

	entries, err := os.ReadDir(directory)
	require.NoError(t, err)
	require.Empty(t, entries)
}
