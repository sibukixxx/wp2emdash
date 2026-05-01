package usecase

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sibukixxx/wp2emdash/internal/usecase/reporting"
)

func TestWriteReportWritesMarkdownOnly(t *testing.T) {
	t.Parallel()

	outDir := t.TempDir()
	summaryPath := filepath.Join(outDir, "summary.json")
	summaryBefore := []byte(`{"keep":"me"}`)
	if err := os.WriteFile(summaryPath, summaryBefore, 0o644); err != nil {
		t.Fatalf("write summary: %v", err)
	}

	bundle := reporting.Bundle{
		GeneratedAt: "2026-05-01T00:00:00Z",
		Tool:        "wp2emdash",
		Version:     "test",
	}

	if err := WriteReport(outDir, bundle); err != nil {
		t.Fatalf("WriteReport() error = %v", err)
	}

	summaryAfter, err := os.ReadFile(summaryPath)
	if err != nil {
		t.Fatalf("read summary: %v", err)
	}
	if string(summaryAfter) != string(summaryBefore) {
		t.Fatalf("summary.json was rewritten: got %q", string(summaryAfter))
	}

	reportBytes, err := os.ReadFile(filepath.Join(outDir, "risk-report.md"))
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	if !strings.Contains(string(reportBytes), "# EmDash Migration Audit Report") {
		t.Fatalf("report heading missing:\n%s", string(reportBytes))
	}
}
