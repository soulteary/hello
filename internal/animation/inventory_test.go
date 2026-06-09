package animation

import (
	"os"
	"path/filepath"
	"testing"
)

func Test_NewInventory_HasParrotAndPedro(t *testing.T) {
	inv := NewInventory()
	for _, name := range []string{"parrot", "pedro", "cat", "coffee", "loading"} {
		if _, ok := inv[name]; !ok {
			t.Errorf("expected embedded inventory to contain %q", name)
		}
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
