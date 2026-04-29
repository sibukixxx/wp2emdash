// Package walk provides a file-traversal helper used throughout wp2emdash.
package walk

import (
	"io/fs"
	"path/filepath"
)

// FileHandler is called for every regular file found under a root directory.
type FileHandler func(path string, d fs.DirEntry) error

// Files walks root and calls fn for each regular file. Per-entry errors from
// the underlying WalkDir (e.g. unreadable directories) are swallowed so the
// scan is best-effort. Errors returned by fn are propagated to the caller;
// returning filepath.SkipAll stops the walk without an error.
func Files(root string, fn FileHandler) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // tolerate per-entry errors (broken symlinks, permissions)
		}
		if d.IsDir() {
			return nil
		}
		return fn(path, d)
	})
}
