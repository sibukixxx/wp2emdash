package cli

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/rokubunnoni-inc/wp2emdash/internal/media"
)

// writeMediaManifest is shared between `media scan` and `run --preset` so the
// on-disk format stays identical.
func writeMediaManifest(path string, m media.Manifest) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(m)
}
