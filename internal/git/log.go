package git

import (
	"bytes"
	"fmt"
	"strings"
)

type Commit struct {
	SHA     string
	Parents []string
	Author  string
	Date    string
	Subject string
}

// ParseLog reads records from git log -z --format with five NUL-terminated fields.
func ParseLog(data []byte) ([]Commit, error) {
	var commits []Commit
	for len(data) > 0 {
		data = bytes.TrimLeft(data, "\r\n\x00")
		if len(data) == 0 {
			break
		}
		var fields [5]string
		for i := range fields {
			end := bytes.IndexByte(data, 0)
			if end < 0 {
				return nil, fmt.Errorf("truncated commit record")
			}
			fields[i] = string(data[:end])
			data = data[end+1:]
		}
		if len(fields[0]) != 40 && len(fields[0]) != 64 {
			return nil, fmt.Errorf("invalid commit SHA %q", fields[0])
		}
		commits = append(commits, Commit{SHA: fields[0], Parents: strings.Fields(fields[1]), Author: fields[2], Date: fields[3], Subject: fields[4]})
	}
	return commits, nil
}
