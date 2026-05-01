package tests

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/sibukixxx/wp2emdash/internal/usecase"
	"github.com/sibukixxx/wp2emdash/test/e2e"
)

func TestMediaSyncCommand_WritesDryRunReport(t *testing.T) {
	t.Parallel()

	cli := e2e.NewCLI(t)
	outDir := filepath.Join(t.TempDir(), "out")
	uploadsDir := filepath.Join(cli.FixtureDir, "wp-content", "uploads")

	res := cli.Run(t,
		"media", "sync",
		"--dir", uploadsDir,
		"--to", "r2:bucket/uploads",
		"--out", outDir,
		"--checksum",
	)

	if !strings.Contains(res.Stdout, "applied:  false") {
		t.Fatalf("stdout missing applied=false:\n%s", res.Stdout)
	}

	report := e2e.DecodeJSONFile[usecase.MediaSyncResult](t, filepath.Join(outDir, "media-sync.json"))
	if report.Applied {
		t.Fatal("report.Applied: want false, got true")
	}
	if !strings.Contains(report.Command.Command, "rclone copy") {
		t.Fatalf("command = %q", report.Command.Command)
	}
}
