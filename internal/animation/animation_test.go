package animation

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"
)

func Test_LoadFromFile(t *testing.T) {
	a, err := LoadFromFile(os.DirFS("assets/animations"), "parrot.animation")
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

func Test_LoadFromFile_Missing(t *testing.T) {
	_, err := LoadFromFile(fstest.MapFS{}, "missing.animation")
	if err == nil {
		t.Fatal("expected a missing-file error")
	}
}

func Test_LoadFromBytes(t *testing.T) {
	t.Run("invalid input", testLoadFromBytesInvalidInput)
	t.Run("valid document", testLoadFromBytesValidDocument)
	t.Run("line endings", testLoadFromBytesLineEndings)
	t.Run("metadata", testLoadFromBytesMetadata)
}

func testLoadFromBytesInvalidInput(t *testing.T) {
	t.Helper()
	tests := []struct {
		name        string
		input       string
		wantMessage string
	}{
		{name: "no frames"},
		{name: "one frame", input: "frame"},
		{name: "empty frame", input: "!--FRAME--!\n"},
		{name: "bare carriage return", input: "description: bad\r!--FRAME--!\nA\n!--FRAME--!\nB", wantMessage: "bare carriage return"},
		{name: "whitespace-only frame", input: "description: bad\n!--FRAME--!\n \t\n!--FRAME--!\nB", wantMessage: "frame 1 is empty"},
		{name: "trailing separator", input: "description: bad\n!--FRAME--!\nA\n!--FRAME--!\n", wantMessage: "frame 2 is empty"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := LoadFromBytes([]byte(tc.input))
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if tc.wantMessage != "" && !strings.Contains(err.Error(), tc.wantMessage) {
				t.Fatalf("error = %v, want message containing %q", err, tc.wantMessage)
			}
		})
	}
}

func testLoadFromBytesValidDocument(t *testing.T) {
	t.Helper()
	a, err := LoadFromBytes([]byte("description: test\n!--FRAME--!\nA\n!--FRAME--!\nB\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Frames) != 2 {
		t.Fatalf("expected 2 frames, got %d", len(a.Frames))
	}
	if a.Metadata["description"] != "test" {
		t.Errorf("description = %q, want test", a.Metadata["description"])
	}
}

func testLoadFromBytesLineEndings(t *testing.T) {
	t.Helper()
	t.Run("CRLF", func(t *testing.T) {
		a, err := LoadFromBytes([]byte("description: windows\r\n!--FRAME--!\r\nA\r\n!--FRAME--!\r\nB\r\n"))
		if err != nil {
			t.Fatal(err)
		}
		if got := string(a.Frames[0]); got != "A\n" {
			t.Errorf("first frame = %q, want %q", got, "A\\n")
		}
	})
	t.Run("mixed", func(t *testing.T) {
		a, err := LoadFromBytes([]byte("description: mixed\r\n!--FRAME--!\nA\r\n!--FRAME--!\r\nB"))
		if err != nil {
			t.Fatal(err)
		}
		if len(a.Frames) != 2 {
			t.Fatalf("frames = %d, want 2", len(a.Frames))
		}
	})
}

func testLoadFromBytesMetadata(t *testing.T) {
	t.Helper()
	a, err := LoadFromBytes([]byte(" description : value:with:colons \n: ignored\nnot metadata\n!--FRAME--!\nA\n!--FRAME--!\nB"))
	if err != nil {
		t.Fatal(err)
	}
	if got := a.Metadata["description"]; got != "value:with:colons" {
		t.Errorf("description = %q", got)
	}
	if _, ok := a.Metadata[""]; ok {
		t.Error("empty metadata key was retained")
	}
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
