package git

import (
	"bytes"
	"fmt"
)

type Hunk struct {
	Patch []byte
}

// ParseHunks returns independently applicable hunks, each with the original
// file headers required by git apply.
func ParseHunks(patch []byte) ([]Hunk, error) {
	lines := bytes.SplitAfter(patch, []byte{'\n'})
	headerEnd := -1
	for i, line := range lines {
		if bytes.HasPrefix(line, []byte("@@ ")) {
			headerEnd = i
			break
		}
	}
	if headerEnd < 0 {
		return nil, fmt.Errorf("diff contains no text hunks")
	}
	prefix := bytes.Join(lines[:headerEnd], nil)
	var hunks []Hunk
	start := headerEnd
	for i := headerEnd + 1; i <= len(lines); i++ {
		if i == len(lines) || bytes.HasPrefix(lines[i], []byte("@@ ")) {
			selected := make([]byte, 0, len(prefix)+len(bytes.Join(lines[start:i], nil)))
			selected = append(selected, prefix...)
			for _, line := range lines[start:i] {
				selected = append(selected, line...)
			}
			hunks = append(hunks, Hunk{Patch: selected})
			start = i
		}
	}
	return hunks, nil
}
