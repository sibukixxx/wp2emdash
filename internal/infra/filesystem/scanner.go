// Package filesystem implements `wp2emdash media scan` — produces a manifest of
// wp-content/uploads (or any directory) so it can be diff'd, hashed, and
// fed into rclone / wrangler r2 / aws-cli for the actual transfer.
package filesystem

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/fs"
	"mime"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rokubunnoni-inc/wp2emdash/internal/domain/media"
	"github.com/rokubunnoni-inc/wp2emdash/internal/walk"
)

// ScanOptions controls scan behavior.
type ScanOptions struct {
	Hash      bool // compute SHA-256 per file (slow on large trees)
	MaxFiles  int  // stop after this many files (0 = no limit) — used by sample mode
	WithFiles bool // include the per-file array (false => extension histogram only)
}

// Scan walks dir and returns a Manifest. It tolerates per-file errors so a
// broken symlink or unreadable file doesn't abort the entire scan.
func Scan(dir string, opt ScanOptions) (media.Manifest, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return media.Manifest{}, err
	}
	manifest := media.Manifest{
		BaseDir:    abs,
		Extensions: map[string]int{},
	}

	walkErr := walk.Files(abs, func(path string, d fs.DirEntry) error {
		info, ierr := d.Info()
		if ierr != nil {
			return nil
		}
		rel, _ := filepath.Rel(abs, path)
		ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(d.Name()), "."))

		manifest.TotalFiles++
		manifest.TotalBytes += info.Size()
		manifest.Extensions[ext]++

		if opt.WithFiles {
			f := media.File{
				Path: rel,
				Size: info.Size(),
				Ext:  ext,
				MIME: mime.TypeByExtension("." + ext),
			}
			if opt.Hash {
				if sum, herr := hashFile(path); herr == nil {
					f.SHA256 = sum
				}
			}
			manifest.Files = append(manifest.Files, f)
			if opt.MaxFiles > 0 && len(manifest.Files) >= opt.MaxFiles {
				return filepath.SkipAll
			}
		}
		return nil
	})

	// Stable sort for reproducible output across runs.
	sort.SliceStable(manifest.Files, func(i, j int) bool {
		return manifest.Files[i].Path < manifest.Files[j].Path
	})

	return manifest, walkErr
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = f.Close()
	}()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
