package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rokubunnoni-inc/wp2emdash/internal/usecase/reporting"
	"github.com/rokubunnoni-inc/wp2emdash/test/e2e"
)

func TestAuditCommand_WritesSummaryAndReport(t *testing.T) {
	t.Parallel()

	cli := e2e.NewCLI(t)
	outDir := filepath.Join(t.TempDir(), "out")

	res := cli.Run(t,
		"audit",
		"--wp-root", cli.FixtureDir,
		"--out", outDir,
	)

	if !strings.Contains(res.Stdout, "Risk score:") {
		t.Fatalf("stdout missing score line:\n%s", res.Stdout)
	}

	summary := e2e.DecodeJSONFile[reporting.Bundle](t, filepath.Join(outDir, "summary.json"))
	if summary.Audit.Site.HomeURL != "https://example.test" {
		t.Fatalf("home_url: want https://example.test, got %q", summary.Audit.Site.HomeURL)
	}
	if summary.Score.Score <= 0 {
		t.Fatalf("score: want > 0, got %d", summary.Score.Score)
	}

	reportBytes, err := os.ReadFile(filepath.Join(outDir, "risk-report.md"))
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	report := string(reportBytes)
	if !strings.Contains(report, "# EmDash Migration Audit Report") {
		t.Fatalf("report heading missing:\n%s", report)
	}
}
