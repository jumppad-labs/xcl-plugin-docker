// Package files is the plugin's only backend layer: the files a note is
// written to.
//
// The note provider never touches the filesystem itself. It holds a Files, a
// narrow interface with only the calls it makes, so its unit tests replace it
// with the strict double Mockery generates into client/files/mocks. Local is
// the real implementation, over the local disk.
//
// When you build a plugin for a real backend, such as an HTTP API or an SDK,
// this is the package you replace: keep the interface to the calls your
// providers make, return this package's own types rather than entity types,
// and keep every backend library import in here.
package files

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Files is the filesystem the note provider uses. Every method takes the
// context of the call it serves and returns an error once it is cancelled.
type Files interface {
	// Write writes content to the file at path with the given mode, creating
	// the file and its parent directories when they do not exist, and returns
	// what was written.
	Write(ctx context.Context, path string, content []byte, mode fs.FileMode) (Info, error)
	// Chmod changes the mode of the file at path.
	Chmod(ctx context.Context, path string, mode fs.FileMode) error
	// Stat reports the file at path. The error wraps fs.ErrNotExist when the
	// file does not exist.
	Stat(ctx context.Context, path string) (Info, error)
	// Remove deletes the file at path. A file that does not exist is not an
	// error.
	Remove(ctx context.Context, path string) error
}

// Info is what the client reports about a file.
type Info struct {
	// Checksum is the lowercase hex sha256 of the file's content.
	Checksum string
	// Mode is the file's permission bits.
	Mode fs.FileMode
}

// Local is the Files implementation over the local disk.
type Local struct{}

var _ Files = (*Local)(nil)

// NewLocal returns a Files over the local disk. It touches nothing until one
// of its methods is called, so a plugin can build it in Init.
func NewLocal() *Local {
	return &Local{}
}

// Write writes content to the file at path with the given mode, creating the
// file and its parent directories when they do not exist.
func (l *Local) Write(ctx context.Context, path string, content []byte, mode fs.FileMode) (Info, error) {
	if err := ctx.Err(); err != nil {
		return Info{}, err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return Info{}, fmt.Errorf("creating the directory for %s: %w", path, err)
	}

	if err := os.WriteFile(path, content, mode); err != nil {
		return Info{}, fmt.Errorf("writing %s: %w", path, err)
	}

	// os.WriteFile only applies the mode when it creates the file, so set it
	// again for a file that already existed
	if err := os.Chmod(path, mode); err != nil {
		return Info{}, fmt.Errorf("setting the mode of %s: %w", path, err)
	}

	return Info{Checksum: checksum(content), Mode: mode.Perm()}, nil
}

// Chmod changes the mode of the file at path.
func (l *Local) Chmod(ctx context.Context, path string, mode fs.FileMode) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if err := os.Chmod(path, mode); err != nil {
		return fmt.Errorf("setting the mode of %s: %w", path, err)
	}

	return nil
}

// Stat reports the checksum and mode of the file at path. The error wraps
// fs.ErrNotExist when the file does not exist.
func (l *Local) Stat(ctx context.Context, path string) (Info, error) {
	if err := ctx.Err(); err != nil {
		return Info{}, err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return Info{}, fmt.Errorf("reading %s: %w", path, err)
	}

	stat, err := os.Stat(path)
	if err != nil {
		return Info{}, fmt.Errorf("inspecting %s: %w", path, err)
	}

	return Info{Checksum: checksum(content), Mode: stat.Mode().Perm()}, nil
}

// Remove deletes the file at path. A file that does not exist is not an
// error, so destroying a note whose file is already gone succeeds.
func (l *Local) Remove(ctx context.Context, path string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("removing %s: %w", path, err)
	}

	return nil
}

// checksum returns the lowercase hex sha256 of content.
func checksum(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}
