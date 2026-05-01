package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sibukixxx/wp2emdash/internal/usecase/reporting"
	"github.com/sibukixxx/wp2emdash/test/e2e"
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

func TestAuditCommand_ShowsWarningsInHumanOutput(t *testing.T) {
	t.Parallel()

	cli := e2e.NewCLI(t)
	cli.ReplaceTool(t, "wp", `#!/bin/sh
set -eu

case "$*" in
  "db prefix")
    printf "wp_"
    ;;
  "option get home")
    printf "https://example.test"
    ;;
  "option get siteurl")
    printf "https://example.test"
    ;;
  "core version")
    printf "6.5.0"
    ;;
  "eval echo PHP_VERSION;")
    printf "8.2.12"
    ;;
  "eval echo is_multisite() ? \"yes\" : \"no\";")
    printf "no"
    ;;
  "post list --post_type=post --post_status=publish --format=count")
    exit 1
    ;;
  "post list --post_type=page --post_status=publish --format=count")
    printf "12"
    ;;
  "post list --post_status=draft --format=count")
    printf "3"
    ;;
  "post list --post_status=private --format=count")
    printf "1"
    ;;
  "term list category --format=count")
    printf "8"
    ;;
  "term list post_tag --format=count")
    printf "15"
    ;;
  "user list --format=count")
    printf "4"
    ;;
  "comment list --status=approve --format=count")
    printf "22"
    ;;
  "theme list --status=active --field=name")
    printf "test-theme"
    ;;
  "plugin list --status=active --format=json")
    printf '[{"name":"advanced-custom-fields","status":"active"},{"name":"redirection","status":"active"}]'
    ;;
  "post-type list --field=name")
    printf "post\npage\nattachment\nlanding_page\n"
    ;;
  "taxonomy list --field=name")
    printf "category\npost_tag\ncampaign\n"
    ;;
  *)
    case "$*" in
      *"SELECT COUNT(*) FROM wp_posts WHERE post_content LIKE '%wp-content/uploads%'"*)
        printf "7"
        ;;
      *"SELECT COUNT(*) FROM wp_posts WHERE post_content LIKE '%http://%'"*)
        printf "2"
        ;;
      *"SELECT COUNT(*) FROM wp_postmeta WHERE meta_key LIKE '%yoast%' OR meta_key LIKE '%rank_math%' OR meta_key LIKE '%aioseo%'"*)
        printf "11"
        ;;
      *"SELECT COUNT(*) FROM wp_postmeta WHERE meta_value LIKE 'a:%' OR meta_value LIKE 'O:%'"*)
        printf "9"
        ;;
      *"SELECT COUNT(*) FROM wp_posts WHERE post_content REGEXP '\\[[a-zA-Z0-9_-]+'"*)
        printf "5"
        ;;
      *)
        exit 1
        ;;
    esac
    ;;
esac
`)

	outDir := filepath.Join(t.TempDir(), "out")
	res := cli.Run(t,
		"audit",
		"--wp-root", cli.FixtureDir,
		"--out", outDir,
	)

	if !strings.Contains(res.Stdout, "Audit warnings: 2") {
		t.Fatalf("stdout missing warning summary:\n%s", res.Stdout)
	}
	if !strings.Contains(res.Stdout, "content.posts") {
		t.Fatalf("stdout missing warning code:\n%s", res.Stdout)
	}
	if !strings.Contains(res.Stdout, "customization.shortcode_posts") {
		t.Fatalf("stdout missing second warning code:\n%s", res.Stdout)
	}

	summary := e2e.DecodeJSONFile[reporting.Bundle](t, filepath.Join(outDir, "summary.json"))
	if len(summary.Warnings) != 2 {
		t.Fatalf("warnings: want 2, got %d", len(summary.Warnings))
	}
}

func TestAuditCommand_OverSSH(t *testing.T) {
	t.Parallel()

	cli := e2e.NewCLI(t)
	outDir := filepath.Join(t.TempDir(), "out")

	res := cli.Run(t,
		"audit",
		"--ssh", "fake@example.test",
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
	if summary.Audit.Theme.ActiveTheme != "test-theme" {
		t.Fatalf("active_theme: want test-theme, got %q", summary.Audit.Theme.ActiveTheme)
	}
}
