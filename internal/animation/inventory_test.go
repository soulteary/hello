package animation

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

type failingFS struct{}

func (failingFS) Open(string) (fs.File, error) {
	return nil, errors.New("read failed")
}

func Test_NewInventory_HasParrotAndPedro(t *testing.T) {
	inv := NewInventory()
	for _, name := range []string{"parrot", "pedro", "cat", "coffee", "loading"} {
		anim, ok := inv[name]
		if !ok {
			t.Errorf("expected embedded inventory to contain %q", name)
			continue
		}
		if len(anim.Frames) < 2 {
			t.Errorf("animation %q has %d frames, want at least 2", name, len(anim.Frames))
		}
		if strings.TrimSpace(anim.Metadata["description"]) == "" {
			t.Errorf("animation %q has no description metadata", name)
		}
	}
}

func Test_MustLoadInventory_PanicsOnSetupErrors(t *testing.T) {
	tests := []struct {
		name          string
		filesystem    fs.FS
		filesystemErr error
	}{
		{name: "sub-filesystem error", filesystem: fstest.MapFS{}, filesystemErr: errors.New("sub failed")},
		{name: "inventory load error", filesystem: failingFS{}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("mustLoadInventory did not panic")
				}
			}()
			mustLoadInventory(tc.filesystem, tc.filesystemErr)
		})
	}
}

func Test_LoadFromFS_IgnoresNonAnimationFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	valid := "description: ok\n!--FRAME--!\nA\n!--FRAME--!\nB\n"
	if err := os.WriteFile(filepath.Join(dir, "x.animation"), []byte(valid), 0o644); err != nil {
		t.Fatal(err)
	}

	inv := Inventory{}
	if err := inv.LoadFromFS(os.DirFS(dir)); err != nil {
		t.Fatal(err)
	}
	if _, ok := inv["x"]; !ok {
		t.Errorf("expected 'x' to be loaded")
	}
	if _, ok := inv["readme"]; ok {
		t.Errorf("expected 'readme.txt' to be ignored")
	}
	if len(inv) != 1 {
		t.Errorf("expected exactly 1 animation, got %d", len(inv))
	}
}

func Test_LoadFromFS_RejectsNilInventory(t *testing.T) {
	var inv Inventory
	err := inv.LoadFromFS(fstest.MapFS{})
	if err == nil || !strings.Contains(err.Error(), "inventory is nil") {
		t.Fatalf("error = %v, want nil-inventory error", err)
	}
}

func Test_LoadFromFS_ReportsDirectoryError(t *testing.T) {
	inv := Inventory{}
	err := inv.LoadFromFS(failingFS{})
	if err == nil || !strings.Contains(err.Error(), "read animation directory") {
		t.Fatalf("error = %v, want directory-read error", err)
	}
}

func Test_LoadFromFS_IsAtomicOnParseFailure(t *testing.T) {
	inv := Inventory{"existing": {Frames: [][]byte{[]byte("A"), []byte("B")}}}
	files := fstest.MapFS{
		"a.animation":        {Data: []byte("description: valid\n!--FRAME--!\nA\n!--FRAME--!\nB")},
		"b.animation":        {Data: []byte("not valid")},
		"nested/c.animation": {Data: []byte("description: nested\n!--FRAME--!\nA\n!--FRAME--!\nB")},
	}
	err := inv.LoadFromFS(files)
	if err == nil || !strings.Contains(err.Error(), "load b.animation") {
		t.Fatalf("error = %v, want b.animation parse error", err)
	}
	if len(inv) != 1 {
		t.Fatalf("inventory was partially mutated: %#v", inv)
	}
	if _, ok := inv["existing"]; !ok {
		t.Fatal("existing inventory entry was removed")
	}
}

func Test_LoadFromFS_MergesAfterSuccessfulParse(t *testing.T) {
	inv := Inventory{"existing": {Frames: [][]byte{[]byte("A"), []byte("B")}}}
	files := fstest.MapFS{
		"new.animation":            {Data: []byte("description: valid\n!--FRAME--!\nA\n!--FRAME--!\nB")},
		"folder/ignored.animation": {Data: []byte("description: nested\n!--FRAME--!\nA\n!--FRAME--!\nB")},
	}
	if err := inv.LoadFromFS(files); err != nil {
		t.Fatal(err)
	}
	if _, ok := inv["new"]; !ok {
		t.Fatal("new animation was not loaded")
	}
	if _, ok := inv["folder/ignored"]; ok {
		t.Fatal("nested animation should not be loaded from the root inventory")
	}
	if _, ok := inv["existing"]; !ok {
		t.Fatal("existing inventory entry was removed")
	}
}
