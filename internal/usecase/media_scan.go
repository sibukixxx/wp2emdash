package usecase

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rokubunnoni-inc/wp2emdash/internal/domain/media"
	"github.com/rokubunnoni-inc/wp2emdash/internal/infra/filesystem"
)

type MediaScanParams struct {
	Dir           string
	OutDir        string
	ManifestPath  string
	Hash          bool
	MaxFiles      int
	HistogramOnly bool
}

type MediaScanResult struct {
	Manifest media.Manifest
	Path     string
}

func RunMediaScan(params MediaScanParams) (MediaScanResult, error) {
	manifest, err := filesystem.Scan(params.Dir, filesystem.ScanOptions{
		Hash:      params.Hash,
		MaxFiles:  params.MaxFiles,
		WithFiles: !params.HistogramOnly,
	})
	if err != nil {
		return MediaScanResult{}, err
	}

	dest := params.ManifestPath
	if dest == "" {
		dest = filepath.Join(params.OutDir, "media-manifest.json")
	}
	if err := writeMediaManifest(dest, manifest); err != nil {
		return MediaScanResult{}, fmt.Errorf("write media manifest: %w", err)
	}
	return MediaScanResult{Manifest: manifest, Path: dest}, nil
}

func writeMediaManifest(path string, manifest media.Manifest) error {
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
	return enc.Encode(manifest)
}
