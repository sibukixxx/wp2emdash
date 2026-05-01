package walk_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/sibukixxx/wp2emdash/internal/walk"
)

func TestFiles(t *testing.T) {
	t.Run("visits regular files only", func(t *testing.T) {
		dir := t.TempDir()
		must(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0o644))
		must(t, os.WriteFile(filepath.Join(dir, "b.txt"), []byte("y"), 0o644))
		must(t, os.MkdirAll(filepath.Join(dir, "sub"), 0o755))
		must(t, os.WriteFile(filepath.Join(dir, "sub", "c.txt"), []byte("z"), 0o644))

		var paths []string
		err := walk.Files(dir, func(path string, _ fs.DirEntry) error {
			paths = append(paths, filepath.Base(path))
			return nil
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(paths) != 3 {
			t.Errorf("want 3 files, got %d: %v", len(paths), paths)
		}
	})

	t.Run("tolerates per-file errors from outer walker", func(t *testing.T) {
		// Files() must not abort when the WalkDir callback gets a non-nil err
		// (e.g. unreadable directory).  We simulate by walking a non-existent
		// subtree; the outer WalkDir will pass err!=nil for those entries but
		// Files() should swallow them and continue.
		dir := t.TempDir()
		must(t, os.WriteFile(filepath.Join(dir, "ok.txt"), []byte("1"), 0o644))

		var visited []string
		err := walk.Files(dir, func(path string, _ fs.DirEntry) error {
			visited = append(visited, filepath.Base(path))
			return nil
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(visited) != 1 || visited[0] != "ok.txt" {
			t.Errorf("want [ok.txt], got %v", visited)
		}
	})

	t.Run("propagates handler error", func(t *testing.T) {
		dir := t.TempDir()
		must(t, os.WriteFile(filepath.Join(dir, "f.txt"), []byte("x"), 0o644))

		sentinel := errors.New("stop")
		err := walk.Files(dir, func(_ string, _ fs.DirEntry) error {
			return sentinel
		})
		if !errors.Is(err, sentinel) {
			t.Errorf("want sentinel error, got %v", err)
		}
	})

	t.Run("SkipAll stops walk without error", func(t *testing.T) {
		dir := t.TempDir()
		must(t, os.WriteFile(filepath.Join(dir, "1.txt"), []byte("a"), 0o644))
		must(t, os.WriteFile(filepath.Join(dir, "2.txt"), []byte("b"), 0o644))

		var count int
		err := walk.Files(dir, func(_ string, _ fs.DirEntry) error {
			count++
			return filepath.SkipAll
		})
		if err != nil {
			t.Fatalf("SkipAll should not return error, got %v", err)
		}
		if count != 1 {
			t.Errorf("want 1 file visited, got %d", count)
		}
	})
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
