package animation

import (
	"embed"
	"errors"
	"fmt"
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
	if i == nil {
		return errors.New("inventory is nil")
	}

	files, err := fs.ReadDir(filesystem, ".")
	if err != nil {
		return fmt.Errorf("read animation directory: %w", err)
	}

	// Parse into a temporary map first. A malformed file must not leave the
	// caller with a partially updated inventory.
	loaded := make(Inventory)
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".animation") {
			continue
		}
		animation, err := LoadFromFile(filesystem, file.Name())
		if err != nil {
			return fmt.Errorf("load %s: %w", file.Name(), err)
		}
		loaded[strings.TrimSuffix(file.Name(), ".animation")] = *animation
	}

	for name, animation := range loaded {
		i[name] = animation
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
