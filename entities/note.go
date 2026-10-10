// Package entities holds the plugin's block types, and nothing else: no
// behaviour, no providers, no clients and no backend libraries.
//
// An application that only reads the resources a plugin applied imports this
// package alone, for example xcl.FindByType[entities.Note], so keep it free of
// imports from the rest of the plugin. Use plain field types: a nested named
// struct is rebuilt empty when the plugin runs as a separate program.
package entities

import "github.com/jumppad-labs/xcl/types"

// Note is the block type notes "note": a note written to a file.
//
//	notes "note" "greeting" {
//	  directory = "./out"
//	  name      = "greeting.txt"
//	  content   = "Hello from xcl"
//	}
type Note struct {
	types.ResourceBase `xcl:",remain"`

	// Directory is the directory the note is written to. Changing it moves
	// the file, so it needs a replace.
	Directory string `xcl:"directory" json:"directory"`

	// Name is the file name of the note. Changing it moves the file, so it
	// needs a replace.
	Name string `xcl:"name" json:"name"`

	// Content is the text of the note. It is updated in place.
	Content string `xcl:"content" json:"content"`

	// Mode is the file's permissions in octal, such as "0600". It defaults to
	// "0644" and is updated in place.
	Mode string `xcl:"mode,optional" json:"mode,omitempty"`

	// Path is computed: the directory joined with the name.
	Path string `xcl:"path,optional,computed" json:"path,omitempty"`

	// Checksum is computed: the sha256 of the content written to the file.
	Checksum string `xcl:"checksum,optional,computed" json:"checksum,omitempty"`
}
