package tests

import (
	"path/filepath"
	"testing"

	"github.com/rokubunnoni-inc/wp2emdash/internal/domain/media"
	"github.com/rokubunnoni-inc/wp2emdash/test/e2e"
)

func TestMediaScanCommand_WritesManifest(t *testing.T) {
	t.Parallel()

	cli := e2e.NewCLI(t)
	outDir := filepath.Join(t.TempDir(), "out")

	cli.Run(t,
		"media", "scan",
		"--dir", filepath.Join(cli.FixtureDir, "wp-content", "uploads"),
		"--out", outDir,
	)

	manifest := e2e.DecodeJSONFile[media.Manifest](t, filepath.Join(outDir, "media-manifest.json"))
	if manifest.TotalFiles == 0 {
		t.Fatalf("total_files: want > 0, got 0")
	}
	if len(manifest.Files) == 0 {
		t.Fatal("files: want entries, got 0")
	}
	if manifest.Files[0].Path == "" {
		t.Fatal("first file path is empty")
	}
}
