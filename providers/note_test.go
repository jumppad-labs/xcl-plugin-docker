package providers_test

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/jumppad-labs/xcl/entity"
	"github.com/jumppad-labs/xcl/plugins"
	"github.com/jumppad-labs/xcl/types"

	"github.com/jumppad-labs/xcl-plugin-template/client/files"
	"github.com/jumppad-labs/xcl-plugin-template/client/files/mocks"
	"github.com/jumppad-labs/xcl-plugin-template/entities"
	"github.com/jumppad-labs/xcl-plugin-template/providers"
)

func testNote() *entities.Note {
	return &entities.Note{
		ResourceBase: types.ResourceBase{
			Meta: types.Meta{Name: "greeting"},
		},
		Directory: "dir",
		Name:      "name.txt",
		Content:   "hello",
	}
}

func TestCreateWritesTheNoteAndSetsComputedValues(t *testing.T) {
	client := mocks.NewMockFiles(t)
	client.EXPECT().
		Write(mock.Anything, "dir/name.txt", []byte("hello"), fs.FileMode(0o644)).
		Return(files.Info{Checksum: "abc123"}, nil).
		Once()

	provider := providers.NewNoteProvider(client)

	result, err := provider.Create(context.Background(), testNote())

	require.NoError(t, err)
	require.Equal(t, "dir/name.txt", result.Path)
	require.Equal(t, "abc123", result.Checksum)
}

func TestCreateWritesWithTheConfiguredMode(t *testing.T) {
	client := mocks.NewMockFiles(t)
	client.EXPECT().
		Write(mock.Anything, "dir/name.txt", []byte("hello"), fs.FileMode(0o600)).
		Return(files.Info{Checksum: "abc123"}, nil).
		Once()

	provider := providers.NewNoteProvider(client)

	note := testNote()
	note.Mode = "0600"

	_, err := provider.Create(context.Background(), note)

	require.NoError(t, err)
}

func TestCreateReturnsWriteError(t *testing.T) {
	writeErr := errors.New("disk full")

	client := mocks.NewMockFiles(t)
	client.EXPECT().
		Write(mock.Anything, "dir/name.txt", []byte("hello"), fs.FileMode(0o644)).
		Return(files.Info{}, writeErr).
		Once()

	provider := providers.NewNoteProvider(client)

	result, err := provider.Create(context.Background(), testNote())

	require.ErrorIs(t, err, writeErr)
	require.Nil(t, result)
}

func TestCreateRejectsAnInvalidMode(t *testing.T) {
	provider := providers.NewNoteProvider(mocks.NewMockFiles(t))

	note := testNote()
	note.Mode = "rw"

	result, err := provider.Create(context.Background(), note)

	require.Error(t, err)
	require.Nil(t, result)
}

func TestReadFillsPathAndChecksumFromTheFile(t *testing.T) {
	client := mocks.NewMockFiles(t)
	client.EXPECT().
		Stat(mock.Anything, "dir/name.txt").
		Return(files.Info{Checksum: "def456"}, nil).
		Once()

	provider := providers.NewNoteProvider(client)

	old := testNote()
	old.Path = "dir/name.txt"
	old.Checksum = "stale"

	updated := testNote()
	updated.Content = "new content"

	result, err := provider.Read(context.Background(), old, updated)

	require.NoError(t, err)
	require.Equal(t, "dir/name.txt", result.Path)
	require.Equal(t, "def456", result.Checksum)
	require.Equal(t, "dir", result.Directory)
	require.Equal(t, "name.txt", result.Name)
	require.Equal(t, "new content", result.Content)
	require.Equal(t, "", result.Mode)
}

func TestReadReportsNotFoundForAMissingFile(t *testing.T) {
	client := mocks.NewMockFiles(t)
	client.EXPECT().
		Stat(mock.Anything, "dir/name.txt").
		Return(files.Info{}, fmt.Errorf("reading dir/name.txt: %w", fs.ErrNotExist)).
		Once()

	provider := providers.NewNoteProvider(client)

	old := testNote()
	old.Path = "dir/name.txt"

	result, err := provider.Read(context.Background(), old, testNote())

	require.ErrorIs(t, err, plugins.ErrNotFound)
	require.Nil(t, result)
}

func TestReadReturnsOtherStatErrors(t *testing.T) {
	statErr := errors.New("permission denied")

	client := mocks.NewMockFiles(t)
	client.EXPECT().
		Stat(mock.Anything, "dir/name.txt").
		Return(files.Info{}, statErr).
		Once()

	provider := providers.NewNoteProvider(client)

	old := testNote()
	old.Path = "dir/name.txt"

	result, err := provider.Read(context.Background(), old, testNote())

	require.ErrorIs(t, err, statErr)
	require.NotErrorIs(t, err, plugins.ErrNotFound)
	require.Nil(t, result)
}

func TestChangedReplacesWhenDirectoryChanges(t *testing.T) {
	provider := providers.NewNoteProvider(mocks.NewMockFiles(t))

	changes := []entity.PropertyChange{
		{Path: entity.Path{}.Attribute("directory"), Before: "a", After: "b"},
	}

	change, err := provider.Changed(context.Background(), testNote(), testNote(), changes, nil)

	require.NoError(t, err)
	require.Equal(t, entity.Replace, change)
}

func TestChangedReplacesWhenNameChanges(t *testing.T) {
	provider := providers.NewNoteProvider(mocks.NewMockFiles(t))

	changes := []entity.PropertyChange{
		{Path: entity.Path{}.Attribute("name"), Before: "a.txt", After: "b.txt"},
	}

	change, err := provider.Changed(context.Background(), testNote(), testNote(), changes, nil)

	require.NoError(t, err)
	require.Equal(t, entity.Replace, change)
}

func TestChangedReplacesWhenADependencyIsReplaced(t *testing.T) {
	provider := providers.NewNoteProvider(mocks.NewMockFiles(t))

	dependencies := []entity.DependencyChange{
		{Address: "notes.note.other", Change: entity.Replace},
	}

	change, err := provider.Changed(context.Background(), testNote(), testNote(), nil, dependencies)

	require.NoError(t, err)
	require.Equal(t, entity.Replace, change)
}

func TestChangedUpdatesWhenContentChanges(t *testing.T) {
	provider := providers.NewNoteProvider(mocks.NewMockFiles(t))

	changes := []entity.PropertyChange{
		{Path: entity.Path{}.Attribute("content"), Before: "a", After: "b"},
	}

	change, err := provider.Changed(context.Background(), testNote(), testNote(), changes, nil)

	require.NoError(t, err)
	require.Equal(t, entity.Update, change)
}

func TestChangedUpdatesWhenModeChanges(t *testing.T) {
	provider := providers.NewNoteProvider(mocks.NewMockFiles(t))

	changes := []entity.PropertyChange{
		{Path: entity.Path{}.Attribute("mode"), Before: "0644", After: "0600"},
	}

	change, err := provider.Changed(context.Background(), testNote(), testNote(), changes, nil)

	require.NoError(t, err)
	require.Equal(t, entity.Update, change)
}

func TestChangedDoesNotReplaceForAnUpdatedDependency(t *testing.T) {
	provider := providers.NewNoteProvider(mocks.NewMockFiles(t))

	dependencies := []entity.DependencyChange{
		{Address: "notes.note.other", Change: entity.Update},
	}

	change, err := provider.Changed(context.Background(), testNote(), testNote(), nil, dependencies)

	require.NoError(t, err)
	require.NotEqual(t, entity.Replace, change)
	require.Equal(t, entity.NoChange, change)
}

func TestChangedDefersWhenNothingChanged(t *testing.T) {
	provider := providers.NewNoteProvider(mocks.NewMockFiles(t))

	change, err := provider.Changed(context.Background(), testNote(), testNote(), nil, nil)

	require.NoError(t, err)
	require.Equal(t, entity.NoChange, change)
}

func TestUpdateRewritesOnlyForAContentChange(t *testing.T) {
	client := mocks.NewMockFiles(t)
	client.EXPECT().
		Write(mock.Anything, "dir/name.txt", []byte("new content"), fs.FileMode(0o644)).
		Return(files.Info{Checksum: "newsum"}, nil).
		Once()

	provider := providers.NewNoteProvider(client)

	note := testNote()
	note.Path = "dir/name.txt"
	note.Checksum = "oldsum"
	note.Content = "new content"

	changes := []entity.PropertyChange{
		{Path: entity.Path{}.Attribute("content"), Before: "hello", After: "new content"},
	}

	result, err := provider.Update(context.Background(), note, changes, nil)

	require.NoError(t, err)
	require.Equal(t, "newsum", result.Checksum)
	require.Equal(t, "dir/name.txt", result.Path)
}

func TestUpdateChangesOnlyTheModeForAModeChange(t *testing.T) {
	client := mocks.NewMockFiles(t)
	client.EXPECT().
		Chmod(mock.Anything, "dir/name.txt", fs.FileMode(0o600)).
		Return(nil).
		Once()

	provider := providers.NewNoteProvider(client)

	note := testNote()
	note.Path = "dir/name.txt"
	note.Checksum = "sum"
	note.Mode = "0600"

	changes := []entity.PropertyChange{
		{Path: entity.Path{}.Attribute("mode"), Before: "0644", After: "0600"},
	}

	result, err := provider.Update(context.Background(), note, changes, nil)

	require.NoError(t, err)
	require.Equal(t, "sum", result.Checksum)
}

func TestUpdateMakesNoCallsWithoutContentOrModeChanges(t *testing.T) {
	provider := providers.NewNoteProvider(mocks.NewMockFiles(t))

	note := testNote()
	note.Path = "dir/name.txt"
	note.Checksum = "sum"

	result, err := provider.Update(context.Background(), note, nil, nil)

	require.NoError(t, err)
	require.Equal(t, "sum", result.Checksum)
}

func TestUpdateReturnsWriteError(t *testing.T) {
	writeErr := errors.New("disk full")

	client := mocks.NewMockFiles(t)
	client.EXPECT().
		Write(mock.Anything, "dir/name.txt", []byte("hello"), fs.FileMode(0o644)).
		Return(files.Info{}, writeErr).
		Once()

	provider := providers.NewNoteProvider(client)

	note := testNote()
	note.Path = "dir/name.txt"

	changes := []entity.PropertyChange{
		{Path: entity.Path{}.Attribute("content"), Before: "a", After: "hello"},
	}

	result, err := provider.Update(context.Background(), note, changes, nil)

	require.ErrorIs(t, err, writeErr)
	require.Nil(t, result)
}

func TestUpdateReturnsChmodError(t *testing.T) {
	chmodErr := errors.New("operation not permitted")

	client := mocks.NewMockFiles(t)
	client.EXPECT().
		Chmod(mock.Anything, "dir/name.txt", fs.FileMode(0o600)).
		Return(chmodErr).
		Once()

	provider := providers.NewNoteProvider(client)

	note := testNote()
	note.Path = "dir/name.txt"
	note.Mode = "0600"

	changes := []entity.PropertyChange{
		{Path: entity.Path{}.Attribute("mode"), Before: "0644", After: "0600"},
	}

	result, err := provider.Update(context.Background(), note, changes, nil)

	require.ErrorIs(t, err, chmodErr)
	require.Nil(t, result)
}

func TestDestroyRemovesTheFile(t *testing.T) {
	client := mocks.NewMockFiles(t)
	client.EXPECT().
		Remove(mock.Anything, "dir/name.txt").
		Return(nil).
		Once()

	provider := providers.NewNoteProvider(client)

	note := testNote()
	note.Path = "dir/name.txt"

	err := provider.Destroy(context.Background(), note, false)

	require.NoError(t, err)
}

func TestDestroyReturnsRemoveError(t *testing.T) {
	removeErr := errors.New("busy")

	client := mocks.NewMockFiles(t)
	client.EXPECT().
		Remove(mock.Anything, "dir/name.txt").
		Return(removeErr).
		Once()

	provider := providers.NewNoteProvider(client)

	note := testNote()
	note.Path = "dir/name.txt"

	err := provider.Destroy(context.Background(), note, false)

	require.ErrorIs(t, err, removeErr)
}
