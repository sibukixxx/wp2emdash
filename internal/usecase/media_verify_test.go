package usecase

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/sibukixxx/wp2emdash/internal/domain/media"
)

func TestRunMediaVerifyWritesReport(t *testing.T) {
	t.Parallel()

	outDir := t.TempDir()
	manifestPath := filepath.Join(outDir, "expected.json")
	expected := media.Manifest{
		BaseDir:    outDir,
		TotalFiles: 1,
		TotalBytes: 12,
		Files: []media.File{
			{Path: "hello.txt", Size: 12},
		},
	}
	if err := writeMediaManifest(manifestPath, expected); err != nil {
		t.Fatalf("write expected manifest: %v", err)
	}
	actualPath := filepath.Join(outDir, "actual.json")
	if err := writeMediaManifest(actualPath, expected); err != nil {
		t.Fatalf("write actual manifest: %v", err)
	}

	res, err := RunMediaVerify(context.Background(), MediaVerifyParams{
		FromManifest:   manifestPath,
		ActualManifest: actualPath,
		OutDir:         outDir,
	})
	if err != nil {
		t.Fatalf("RunMediaVerify() error = %v", err)
	}
	if !res.Report.OK {
		t.Fatal("report.OK: want true, got false")
	}
	if res.Path == "" {
		t.Fatal("report path is empty")
	}
}
