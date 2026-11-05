package git

import (
	"bytes"
	"fmt"
	"strings"
)

type Ref struct {
	FullName string
	Name     string
	SHA      string
	Upstream string
	Remote   bool
}

// ParseRefs reads NUL-terminated (refname, objectname, upstream) triples.
func ParseRefs(data []byte) ([]Ref, error) {
	var refs []Ref
	for len(data) > 0 {
		data = bytes.TrimLeft(data, "\r\n")
		if len(data) == 0 {
			break
		}
		var fields [3]string
		for i := range fields {
			end := bytes.IndexByte(data, 0)
			if end < 0 {
				return nil, fmt.Errorf("truncated ref record")
			}
			fields[i] = string(data[:end])
			data = data[end+1:]
		}
		ref := Ref{FullName: fields[0], Name: fields[0], SHA: fields[1], Upstream: fields[2]}
		switch {
		case strings.HasPrefix(ref.Name, "refs/heads/"):
			ref.Name = ref.Name[len("refs/heads/"):]
		case strings.HasPrefix(ref.Name, "refs/remotes/"):
			ref.Name = ref.Name[len("refs/remotes/"):]
			ref.Remote = true
		default:
			return nil, fmt.Errorf("unexpected ref %q", ref.Name)
		}
		if ref.Name == "" || len(ref.SHA) != 40 && len(ref.SHA) != 64 {
			return nil, fmt.Errorf("invalid ref %q", ref.Name)
		}
		refs = append(refs, ref)
	}
	return refs, nil
}
