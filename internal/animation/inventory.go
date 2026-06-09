package animation

import (
	"embed"
	"io/fs"
	"strings"
)

//go:embed assets/animations/*
var animations embed.FS

// Inventory is the in-memory catalog of animations keyed by their base name
// (the file name without the `.animation` suffix).
type Inventory map[string]Animation

// LoadFromFS populates the inventory from any fs.FS rooted at a directory of
// `*.animation` files. Files without the suffix are silently ignored.
func (i Inventory) LoadFromFS(filesystem fs.FS) error {
	files, err := fs.ReadDir(filesystem, ".")
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".animation") {
			continue
		}
		animation, err := LoadFromFile(filesystem, file.Name())
		if err != nil {
			return err
		}
		i[strings.TrimSuffix(file.Name(), ".animation")] = *animation
	}

	return nil
}

// NewInventory returns an inventory pre-populated from the embedded
// animations directory. It panics if the embedded data is malformed because
// that indicates a build-time bug, not a runtime condition.
func NewInventory() Inventory {
	i := make(Inventory)
	sub, err := fs.Sub(animations, "assets/animations")
	if err != nil {
		panic(err)
	}
	if err := i.LoadFromFS(sub); err != nil {
		panic(err)
	}
	return i
}
