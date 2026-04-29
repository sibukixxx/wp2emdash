// Package media implements `wp2emdash media scan` — produces a manifest of
// wp-content/uploads (or any directory) so it can be diff'd, hashed, and
// fed into rclone / wrangler r2 / aws-cli for the actual transfer.
package media

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
)

type File struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256,omitempty"`
	MIME   string `json:"mime,omitempty"`
	Ext    string `json:"ext"`
}

// Manifest is what the JSON output of `media scan` parses to.
type Manifest struct {
	BaseDir    string         `json:"base_dir"`
	TotalFiles int            `json:"total_files"`
	TotalBytes int64          `json:"total_bytes"`
	Extensions map[string]int `json:"extensions"`
	Files      []File         `json:"files"`
}

// Options controls scan behavior.
type Options struct {
	Hash      bool // compute SHA-256 per file (slow on large trees)
	MaxFiles  int  // stop after this many files (0 = no limit) — used by sample mode
	WithFiles bool // include the per-file array (false => extension histogram only)
}

// Scan walks dir and returns a Manifest. It tolerates per-file errors so a
// broken symlink or unreadable file doesn't abort the entire scan.
func Scan(dir string, opt Options) (Manifest, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return Manifest{}, err
	}
	manifest := Manifest{
		BaseDir:    abs,
		Extensions: map[string]int{},
	}

	walkErr := filepath.WalkDir(abs, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
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
			f := File{
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
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
