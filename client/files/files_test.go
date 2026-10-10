package files_test

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jumppad-labs/xcl-plugin-template/client/files"
)

// helloChecksum is the sha256 of the bytes "hello"
const helloChecksum = "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"

func TestWriteCreatesParentDirectoriesAndReportsChecksumAndMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a", "b", "note.txt")

	info, err := files.NewLocal().Write(context.Background(), path, []byte("hello"), 0o644)
	require.NoError(t, err)

	require.Equal(t, helloChecksum, info.Checksum)
	require.Equal(t, fs.FileMode(0o644), info.Mode)

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "hello", string(content))

	stat, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, fs.FileMode(0o644), stat.Mode().Perm())
}

func TestWriteOverExistingFileSetsTheNewMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "note.txt")
	require.NoError(t, os.WriteFile(path, []byte("old"), 0o644))
	require.NoError(t, os.Chmod(path, 0o644))

	info, err := files.NewLocal().Write(context.Background(), path, []byte("hello"), 0o600)
	require.NoError(t, err)
	require.Equal(t, fs.FileMode(0o600), info.Mode)

	stat, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, fs.FileMode(0o600), stat.Mode().Perm())

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "hello", string(content))
}

func TestChmodChangesTheMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "note.txt")
	require.NoError(t, os.WriteFile(path, []byte("hello"), 0o644))
	require.NoError(t, os.Chmod(path, 0o644))

	require.NoError(t, files.NewLocal().Chmod(context.Background(), path, 0o600))

	stat, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, fs.FileMode(0o600), stat.Mode().Perm())
}

func TestStatReportsChecksumAndMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "note.txt")
	require.NoError(t, os.WriteFile(path, []byte("hello"), 0o640))
	require.NoError(t, os.Chmod(path, 0o640))

	info, err := files.NewLocal().Stat(context.Background(), path)
	require.NoError(t, err)

	require.Equal(t, helloChecksum, info.Checksum)
	require.Equal(t, fs.FileMode(0o640), info.Mode)
}

func TestStatOfAMissingFileWrapsErrNotExist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.txt")

	_, err := files.NewLocal().Stat(context.Background(), path)
	require.ErrorIs(t, err, fs.ErrNotExist)
}

func TestRemoveDeletesTheFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "note.txt")
	require.NoError(t, os.WriteFile(path, []byte("hello"), 0o644))

	require.NoError(t, files.NewLocal().Remove(context.Background(), path))

	_, err := os.Stat(path)
	require.ErrorIs(t, err, fs.ErrNotExist)
}

func TestRemoveOfAMissingFileSucceeds(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.txt")

	require.NoError(t, files.NewLocal().Remove(context.Background(), path))
}

func TestWriteWithACancelledContextReturnsContextCanceledAndWritesNothing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "note.txt")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := files.NewLocal().Write(ctx, path, []byte("hello"), 0o644)
	require.ErrorIs(t, err, context.Canceled)

	_, err = os.Stat(filepath.Dir(path))
	require.ErrorIs(t, err, fs.ErrNotExist)
}
