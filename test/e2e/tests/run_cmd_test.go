package tests

import (
	"path/filepath"
	"testing"

	"github.com/rokubunnoni-inc/wp2emdash/internal/usecase/reporting"
	"github.com/rokubunnoni-inc/wp2emdash/test/e2e"
)

func TestRunCommand_MinimalPresetApply(t *testing.T) {
	t.Parallel()

	cli := e2e.NewCLI(t)
	outDir := filepath.Join(t.TempDir(), "out")

	cli.Run(t,
		"run",
		"--preset", "minimal",
		"--wp-root", cli.FixtureDir,
		"--out", outDir,
		"--apply",
	)

	summary := e2e.DecodeJSONFile[reporting.Bundle](t, filepath.Join(outDir, "summary.json"))
	if summary.Tool != "wp2emdash" {
		t.Fatalf("tool: want wp2emdash, got %q", summary.Tool)
	}
}
