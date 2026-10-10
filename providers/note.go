// Package providers holds the plugin's resource providers, one file per
// resource type.
//
// Each provider is built by a constructor that takes the clients it uses, so
// the plugin passes it the real clients and the unit tests pass strict
// doubles. Methods follow the lifecycle order: Init, Create, Read, Changed,
// Update, Destroy, Functions.
package providers

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strconv"

	"github.com/jumppad-labs/xcl/entity"
	"github.com/jumppad-labs/xcl/logger"
	"github.com/jumppad-labs/xcl/plugins"

	"github.com/jumppad-labs/xcl-plugin-template/client/files"
	"github.com/jumppad-labs/xcl-plugin-template/entities"
)

// ReplaceSettings are the note settings that cannot change in place: changing
// either moves the file, so the old note is removed and a new one written.
// Edit this list to change what needs a replace; every other setting is
// updated in place.
var ReplaceSettings = []entity.Path{
	entity.Path{}.Attribute("directory"),
	entity.Path{}.Attribute("name"),
}

// NoteProvider writes, reads, updates and removes the file of a notes "note"
// block through the files client it was given.
type NoteProvider struct {
	plugins.DefaultChanged[*entities.Note]

	files files.Files
}

var _ plugins.ResourceProvider[*entities.Note] = (*NoteProvider)(nil)

// NewNoteProvider returns the provider for notes "note" blocks, which works on
// files only through filesClient.
func NewNoteProvider(filesClient files.Files) *NoteProvider {
	return &NoteProvider{files: filesClient}
}

// Init logs that the provider is ready, it needs nothing else.
func (p *NoteProvider) Init(state plugins.State, functions plugins.ProviderFunctions, log logger.Logger) error {
	log.Debug("provider ready")
	return nil
}

// Create writes the note's file and records its path and checksum.
func (p *NoteProvider) Create(ctx context.Context, note *entities.Note) (*entities.Note, error) {
	mode, err := fileMode(note.Mode)
	if err != nil {
		return nil, err
	}

	path := filepath.Join(note.Directory, note.Name)

	info, err := p.files.Write(ctx, path, []byte(note.Content), mode)
	if err != nil {
		return nil, fmt.Errorf("unable to write note %s: %w", note.Meta.Name, err)
	}

	note.Path = path
	note.Checksum = info.Checksum
	plugins.Logger(ctx).Info("created note", "path", note.Path, "checksum", note.Checksum)

	return note, nil
}

// Read reports the real file: its path and the checksum of what it holds now.
// It returns plugins.ErrNotFound when the file is gone, so xcl writes it
// again. It never changes the configured settings.
func (p *NoteProvider) Read(ctx context.Context, old *entities.Note, new *entities.Note) (*entities.Note, error) {
	info, err := p.files.Stat(ctx, old.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, plugins.ErrNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("unable to read note %s: %w", old.Meta.Name, err)
	}

	new.Path = old.Path
	new.Checksum = info.Checksum

	return new, nil
}

// Changed decides from what it is told. It answers replace when a setting in
// ReplaceSettings changed or a dependency is being replaced, update when any
// other setting changed, and leaves the rest to DefaultChanged.
func (p *NoteProvider) Changed(ctx context.Context, old *entities.Note, new *entities.Note, changes []entity.PropertyChange, dependencies []entity.DependencyChange) (entity.Change, error) {
	for _, change := range changes {
		for _, setting := range ReplaceSettings {
			if change.Within(setting) {
				return entity.Replace, nil
			}
		}
	}

	// A note is rebuilt when anything it depends on is replaced. A resource
	// that is built on another type would narrow this to that type, for
	// example by checking the dependency's Address.
	for _, dependency := range dependencies {
		if dependency.Change == entity.Replace {
			return entity.Replace, nil
		}
	}

	if len(changes) > 0 {
		return entity.Update, nil
	}

	return p.DefaultChanged.Changed(ctx, old, new, changes, dependencies)
}

// Update acts only on the changes it is given. A content change rewrites the
// file, a mode change alone only changes its mode. It never looks the file up
// to find out what it was.
func (p *NoteProvider) Update(ctx context.Context, note *entities.Note, changes []entity.PropertyChange, dependencies []entity.DependencyChange) (*entities.Note, error) {
	contentChanged := false
	modeChanged := false

	for _, change := range changes {
		if change.Within(contentSetting) {
			contentChanged = true
		}

		if change.Within(modeSetting) {
			modeChanged = true
		}
	}

	if !contentChanged && !modeChanged {
		return note, nil
	}

	mode, err := fileMode(note.Mode)
	if err != nil {
		return nil, err
	}

	// writing the file sets its mode too, so a content change needs no
	// separate mode change
	if contentChanged {
		info, err := p.files.Write(ctx, note.Path, []byte(note.Content), mode)
		if err != nil {
			return nil, fmt.Errorf("unable to write note %s: %w", note.Meta.Name, err)
		}

		note.Checksum = info.Checksum
		plugins.Logger(ctx).Info("rewrote note", "path", note.Path, "checksum", note.Checksum)

		return note, nil
	}

	if err := p.files.Chmod(ctx, note.Path, mode); err != nil {
		return nil, fmt.Errorf("unable to change the mode of note %s: %w", note.Meta.Name, err)
	}

	plugins.Logger(ctx).Info("changed note mode", "path", note.Path, "mode", note.Mode)

	return note, nil
}

// Destroy removes the note's file. A file that is already gone counts as
// removed.
func (p *NoteProvider) Destroy(ctx context.Context, note *entities.Note, force bool) error {
	if err := p.files.Remove(ctx, note.Path); err != nil {
		return fmt.Errorf("unable to remove note %s: %w", note.Meta.Name, err)
	}

	plugins.Logger(ctx).Info("destroyed note", "path", note.Path)

	return nil
}

// Functions returns nil, the provider offers no functions.
func (p *NoteProvider) Functions() plugins.ProviderFunctions {
	return nil
}

// fileMode returns the permissions an octal mode string names, defaultMode
// when it is empty.
func fileMode(mode string) (fs.FileMode, error) {
	if mode == "" {
		return defaultMode, nil
	}

	parsed, err := strconv.ParseUint(mode, 8, 32)
	if err != nil || parsed > 0o777 {
		return 0, fmt.Errorf("mode %q is not an octal file mode such as \"0644\"", mode)
	}

	return fs.FileMode(parsed), nil
}

// defaultMode is the permissions of a note that sets no mode.
const defaultMode fs.FileMode = 0o644

// contentSetting and modeSetting are the settings Update acts on.
var (
	contentSetting = entity.Path{}.Attribute("content")
	modeSetting    = entity.Path{}.Attribute("mode")
)
