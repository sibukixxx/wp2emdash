package tests

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/sibukixxx/wp2emdash/internal/domain/media"
	"github.com/sibukixxx/wp2emdash/test/e2e"
)

func TestMediaVerifyCommand_ComparesAgainstDirectory(t *testing.T) {
	t.Parallel()

	cli := e2e.NewCLI(t)
	outDir := filepath.Join(t.TempDir(), "out")
	uploadsDir := filepath.Join(cli.FixtureDir, "wp-content", "uploads")

	cli.Run(t,
		"media", "scan",
		"--dir", uploadsDir,
		"--hash",
		"--out", outDir,
	)
	res := cli.Run(t,
		"media", "verify",
		"--from", filepath.Join(outDir, "media-manifest.json"),
		"--dir", uploadsDir,
		"--out", outDir,
	)

	if !strings.Contains(res.Stdout, "status:   OK") {
		t.Fatalf("stdout missing OK status:\n%s", res.Stdout)
	}

	report := e2e.DecodeJSONFile[media.VerifyReport](t, filepath.Join(outDir, "media-verify.json"))
	if !report.OK {
		t.Fatal("report.OK: want true, got false")
	}
}
