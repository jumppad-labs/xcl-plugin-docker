// Package notes is the notes plugin, which provides the block type
// notes "note" and writes each note to a file.
//
// Plugin is importable: an application registers it in-process with
// registry.Local.RegisterPlugin(&notes.Plugin{}), and cmd/notes serves the
// same type as a separate program. The block types live in entities, the
// providers in providers and the backend layer in client.
//
// To make the template your own, rename this package and cmd/notes after
// your plugin, and change the block type registered in Init.
package notes

import (
	"github.com/jumppad-labs/xcl/logger"
	"github.com/jumppad-labs/xcl/plugins"

	"github.com/jumppad-labs/xcl-plugin-template/client/files"
	"github.com/jumppad-labs/xcl-plugin-template/entities"
	"github.com/jumppad-labs/xcl-plugin-template/providers"
)

// Plugin provides the block type notes "note". cmd/notes serves it as a
// separate plugin program.
type Plugin struct {
	plugins.PluginBase
}

var _ plugins.Plugin = (*Plugin)(nil)

// Init builds the files client once, without touching the disk, and registers
// the notes "note" block type with a provider built around it. A plugin with
// several types builds each client once here and shares it between the
// providers that use it.
func (p *Plugin) Init(logger logger.Logger, state plugins.State) error {
	filesClient := files.NewLocal()

	logger.Debug("registering block types", "block_types", "notes.note")

	return plugins.RegisterResourceProvider(
		&p.PluginBase,
		logger,
		state,
		"notes",
		"note",
		&entities.Note{},
		providers.NewNoteProvider(filesClient),
	)
}
