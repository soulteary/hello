package animation

import (
	"bytes"
	"fmt"
	"io/fs"
	"strings"
)

type Animation struct {
	Metadata map[string]string
	Frames   [][]byte
}

func LoadFromFile(files fs.FS, path string) (*Animation, error) {
	b, err := fs.ReadFile(files, path)
	if err != nil {
		return nil, err
	}

	return LoadFromBytes(b)
}

func LoadFromBytes(b []byte) (*Animation, error) {
	// Normalize CRLF before parsing so files produced on different platforms,
	// including files with mixed LF/CRLF line endings, behave identically.
	// A bare carriage return is not a supported line ending and is rejected to
	// avoid leaking control characters into a terminal or browser response.
	b = bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n"))
	if bytes.ContainsRune(b, '\r') {
		return nil, fmt.Errorf("invalid animation: unsupported bare carriage return")
	}

	frames := bytes.Split(b, []byte("!--FRAME--!\n"))

	// Split yields N+1 segments for N separators. We need at least one
	// metadata header plus 2 frames (an "animation" with a single frame is
	// just a static picture, not worth the framework around it).
	if len(frames) < 3 {
		return nil, fmt.Errorf("invalid animation: need a metadata header and at least 2 frames, got %d segment(s)", len(frames))
	}

	// The first "frame" is actually the metadata.
	metadata := make(map[string]string)
	for _, line := range bytes.Split(frames[0], []byte{'\n'}) {
		parts := bytes.SplitN(line, []byte{':'}, 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(string(parts[0]))
		if key == "" {
			continue
		}
		metadata[key] = strings.TrimSpace(string(parts[1]))
	}

	for i, frame := range frames[1:] {
		if len(bytes.TrimSpace(frame)) == 0 {
			return nil, fmt.Errorf("invalid animation: frame %d is empty", i+1)
		}
	}

	return &Animation{Metadata: metadata, Frames: frames[1:]}, nil
}
