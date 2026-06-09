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
	// Handle both Unix (\n) and Windows (\r\n) line endings
	separator := []byte("!--FRAME--!\n")
	if bytes.Contains(b, []byte("!--FRAME--!\r\n")) {
		separator = []byte("!--FRAME--!\r\n")
	}

	frames := bytes.Split(b, separator)

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
		metadata[strings.TrimSpace(string(parts[0]))] = strings.TrimSpace(string(parts[1]))
	}

	for i, frame := range frames[1:] {
		if len(frame) == 0 {
			return nil, fmt.Errorf("invalid animation: frame %d is empty", i)
		}
	}

	return &Animation{metadata, frames[1:]}, nil
}
