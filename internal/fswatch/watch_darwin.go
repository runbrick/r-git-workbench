//go:build darwin

package fswatch

import "context"

// Watch leaves manual refresh available until a macOS event backend is added.
func Watch(context.Context, []string, func(), func(error)) error { return nil }
