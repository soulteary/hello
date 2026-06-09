package main

import (
	"os"
	"testing"
)

func Test_LoadFromFile(t *testing.T) {
	a, err := LoadFromFile(os.DirFS("animations"), "parrot.animation")
	if err != nil {
		t.Fatal(err)
	}

	if len(a.Frames) < 2 {
		for _, frame := range a.Frames {
			t.Log(string(frame))
		}
		t.Fatalf("expected at least 2 frames, got %d", len(a.Frames))
	}
}

func Test_LoadFromBytes(t *testing.T) {
	t.Run("invalid: no frames", func(t *testing.T) {
		_, err := LoadFromBytes([]byte{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("invalid: one frame", func(t *testing.T) {
		_, err := LoadFromBytes([]byte("frame"))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("invalid: empty frame", func(t *testing.T) {
		_, err := LoadFromBytes([]byte("!--FRAME--!\n"))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("valid", func(t *testing.T) {
		a, err := LoadFromBytes([]byte("description: test\n!--FRAME--!\nA\n!--FRAME--!\nB\n"))
		if err != nil {
			t.Fatal(err)
		}

		if len(a.Frames) != 2 {
			t.Fatalf("expected 2 frames, got %d", len(a.Frames))
		}
	})
}

// FuzzLoadFromBytes ensures the parser never panics on arbitrary input and
// that, when it succeeds, the result satisfies the documented invariants
// (at least two non-empty frames).
func FuzzLoadFromBytes(f *testing.F) {
	f.Add([]byte(""))
	f.Add([]byte("frame"))
	f.Add([]byte("!--FRAME--!\n"))
	f.Add([]byte("description: x\n!--FRAME--!\nA\n!--FRAME--!\nB\n"))
	f.Add([]byte("description: x\r\n!--FRAME--!\r\nA\r\n!--FRAME--!\r\nB\r\n"))
	f.Add([]byte(": no key\nkey only\n!--FRAME--!\nA\n!--FRAME--!\nB"))

	f.Fuzz(func(t *testing.T, data []byte) {
		a, err := LoadFromBytes(data)
		if err != nil {
			return
		}
		if len(a.Frames) < 2 {
			t.Fatalf("parser returned %d frames, want >= 2", len(a.Frames))
		}
		for i, frame := range a.Frames {
			if len(frame) == 0 {
				t.Fatalf("parser returned empty frame at index %d", i)
			}
		}
	})
}
