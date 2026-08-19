package tests

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/sibukixxx/wp2emdash/internal/domain/contentverify"
	"github.com/sibukixxx/wp2emdash/test/e2e"
)

func TestContentSnapshotAndVerifyCommands(t *testing.T) {
	cli := e2e.NewCLI(t)
	out := filepath.Join(t.TempDir(), "out")
	cli.ReplaceTool(t, "emdash", `#!/bin/sh
set -eu
case "$*" in
  "content list posts --limit 100 --url https://new.example.test --json") printf '{"items":[{"id":"emdash-1","slug":"hello","locale":"","status":"published","title":"Hello"}]}' ;;
  "content list pages --limit 100 --url https://new.example.test --json") printf '{"items":[]}' ;;
  "content get posts emdash-1 --raw --published --url https://new.example.test --json") printf '{"id":"emdash-1","slug":"hello","locale":"","status":"published","createdAt":"2024-01-02T03:04:05Z","updatedAt":"2024-01-02T03:04:05Z","publishedAt":"2024-01-02T03:04:05Z","data":{"title":"Hello","content":[{"_type":"block","style":"normal","children":[{"_type":"span","text":"Hello world"}]}]}}' ;;
  *) exit 1 ;;
esac
`)
	cli.Run(t, "content", "snapshot", "wordpress", "--wp-root", cli.FixtureDir, "--out", out)
	cli.Run(t, "content", "snapshot", "emdash", "--url", "https://new.example.test", "--out", out)
	res := cli.Run(t, "content", "verify", "--out", out)
	if !strings.Contains(res.Stdout, "gate:     PASS") {
		t.Fatalf("missing PASS gate:\n%s", res.Stdout)
	}
	report := e2e.DecodeJSONFile[contentverify.Report](t, filepath.Join(out, "content-verify.json"))
	if !report.OK || report.Totals.Matched != 1 {
		t.Fatalf("unexpected report: %+v", report)
	}
	resolved := e2e.DecodeJSONFile[contentverify.Map](t, filepath.Join(out, "content-resolved-map.json"))
	if len(resolved.Entries) != 1 || resolved.Entries[0].EmDashID != "emdash-1" {
		t.Fatalf("resolved ledger missing: %+v", resolved)
	}
}
