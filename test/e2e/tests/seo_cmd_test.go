package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sibukixxx/wp2emdash/internal/domain/seo"
	"github.com/sibukixxx/wp2emdash/test/e2e"
)

func TestSEOExtractMetaCommand_WritesItemsAndMergesPluginPrecedence(t *testing.T) {
	t.Parallel()

	cli := e2e.NewCLI(t)
	outDir := filepath.Join(t.TempDir(), "out")

	res := cli.Run(t,
		"seo", "extract-meta",
		"--wp-root", cli.FixtureDir,
		"--out", outDir,
	)

	if !strings.Contains(res.Stdout, "items: 2") {
		t.Fatalf("stdout missing items count:\n%s", res.Stdout)
	}

	set := e2e.DecodeJSONFile[seo.MetaSet](t, filepath.Join(outDir, "seo-meta.json"))
	if len(set.Items) != 2 {
		t.Fatalf("items: want 2, got %d", len(set.Items))
	}
	if set.Items[0].Title != "Hello SEO" {
		t.Errorf("post 1 title should come from Yoast: got %q", set.Items[0].Title)
	}
	if set.Items[0].Source != "yoast" {
		t.Errorf("post 1 source: want yoast, got %q", set.Items[0].Source)
	}
	if !set.Items[0].NoIndex {
		t.Errorf("post 1 NoIndex should be true")
	}
	if set.Items[1].Title != "About Page" {
		t.Errorf("post 2 title should come from Rank Math: got %q", set.Items[1].Title)
	}
	if set.Items[1].Source != "rank_math" {
		t.Errorf("post 2 source: want rank_math, got %q", set.Items[1].Source)
	}
}

func TestSEOExtractRedirectsCommand_MergesHtaccessAndPluginRules(t *testing.T) {
	t.Parallel()

	cli := e2e.NewCLI(t)
	outDir := filepath.Join(t.TempDir(), "out")

	res := cli.Run(t,
		"seo", "extract-redirects",
		"--wp-root", cli.FixtureDir,
		"--out", outDir,
	)

	if !strings.Contains(res.Stdout, "rules:") {
		t.Fatalf("stdout missing rules count:\n%s", res.Stdout)
	}

	set := e2e.DecodeJSONFile[seo.RedirectSet](t, filepath.Join(outDir, "seo-redirects.json"))
	bySource := map[string]int{}
	for _, r := range set.Rules {
		bySource[r.Source]++
	}
	if bySource["htaccess"] != 3 {
		t.Errorf("htaccess rule count: want 3 (Redirect + RedirectMatch + RewriteRule R=301), got %d", bySource["htaccess"])
	}
	if bySource["redirection"] != 2 {
		t.Errorf("redirection rule count: want 2, got %d", bySource["redirection"])
	}
	if bySource["safe-redirect-manager"] != 1 {
		t.Errorf("SRM rule count: want 1, got %d", bySource["safe-redirect-manager"])
	}
}

func TestSEOURLMapCommand_DiffsTwoFiles(t *testing.T) {
	t.Parallel()

	cli := e2e.NewCLI(t)
	dir := t.TempDir()
	oldPath := filepath.Join(dir, "old.txt")
	newPath := filepath.Join(dir, "new.txt")
	if err := os.WriteFile(oldPath, []byte("https://example.com/a/\nhttps://example.com/b/\n"), 0o644); err != nil {
		t.Fatalf("write old: %v", err)
	}
	if err := os.WriteFile(newPath, []byte("https://example.com/a/\nhttps://example.com/c/\n"), 0o644); err != nil {
		t.Fatalf("write new: %v", err)
	}

	res := cli.Run(t,
		"seo", "url-map",
		"--old", oldPath,
		"--new", newPath,
		"--out", dir,
	)

	if !strings.Contains(res.Stdout, "matched:     1") {
		t.Errorf("stdout missing matched count:\n%s", res.Stdout)
	}

	diff := e2e.DecodeJSONFile[seo.URLMapDiff](t, filepath.Join(dir, "seo-url-map.json"))
	if diff.Total.Matched != 1 || diff.Total.OnlyInOld != 1 || diff.Total.OnlyInNew != 1 {
		t.Errorf("totals: %+v", diff.Total)
	}
	if len(diff.OnlyInOld) != 1 || diff.OnlyInOld[0] != "https://example.com/b/" {
		t.Errorf("OnlyInOld: %v", diff.OnlyInOld)
	}
	if len(diff.OnlyInNew) != 1 || diff.OnlyInNew[0] != "https://example.com/c/" {
		t.Errorf("OnlyInNew: %v", diff.OnlyInNew)
	}
}
