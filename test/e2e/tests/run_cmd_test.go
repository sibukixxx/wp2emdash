package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sibukixxx/wp2emdash/internal/domain/media"
	"github.com/sibukixxx/wp2emdash/internal/usecase/reporting"
	"github.com/sibukixxx/wp2emdash/test/e2e"
)

func TestRunCommand_MinimalPresetApply(t *testing.T) {
	t.Parallel()

	cli := e2e.NewCLI(t)
	outDir := filepath.Join(t.TempDir(), "out")

	res := cli.Run(t,
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
	if !strings.Contains(res.Stdout, "warnings: 1") {
		t.Fatalf("stdout missing phase warning count:\n%s", res.Stdout)
	}
	if !strings.Contains(res.Stdout, "warning codes: customization.shortcode_posts") {
		t.Fatalf("stdout missing warning codes:\n%s", res.Stdout)
	}
}

func TestRunCommand_MinimalPresetApply_ShowsPhaseWarnings(t *testing.T) {
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
		"run",
		"--preset", "minimal",
		"--wp-root", cli.FixtureDir,
		"--out", outDir,
		"--apply",
	)

	if !strings.Contains(res.Stdout, "warnings: 2") {
		t.Fatalf("stdout missing phase warning count:\n%s", res.Stdout)
	}
	if !strings.Contains(res.Stdout, "warning codes: content.posts, customization.shortcode_posts") {
		t.Fatalf("stdout missing warning codes:\n%s", res.Stdout)
	}
}

func TestRunCommand_MinimalPresetApply_OverSSH(t *testing.T) {
	t.Parallel()

	cli := e2e.NewCLI(t)
	outDir := filepath.Join(t.TempDir(), "out")

	res := cli.Run(t,
		"run",
		"--preset", "minimal",
		"--ssh", "fake@example.test",
		"--wp-root", cli.FixtureDir,
		"--out", outDir,
		"--apply",
	)

	if !strings.Contains(res.Stdout, "phase: audit") {
		t.Fatalf("stdout missing phase output:\n%s", res.Stdout)
	}

	summary := e2e.DecodeJSONFile[reporting.Bundle](t, filepath.Join(outDir, "summary.json"))
	if summary.Tool != "wp2emdash" {
		t.Fatalf("tool: want wp2emdash, got %q", summary.Tool)
	}

	manifest := e2e.DecodeJSONFile[media.Manifest](t, filepath.Join(outDir, "media-manifest.json"))
	if manifest.BaseDir != filepath.Join(cli.FixtureDir, "wp-content", "uploads") {
		t.Fatalf("base_dir: got %q", manifest.BaseDir)
	}
	if _, err := os.Stat(filepath.Join(outDir, "risk-report.md")); err != nil {
		t.Fatalf("risk-report.md missing: %v", err)
	}
}
