package usecase

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sibukixxx/wp2emdash/internal/usecase/reporting"
)

func TestRunDBPlan_WritesPlanArtifacts(t *testing.T) {
	t.Parallel()

	outDir := t.TempDir()
	summaryPath := filepath.Join(outDir, "summary.json")
	bundle := reporting.Bundle{
		Audit: reporting.Bundle{}.Audit,
	}
	bundle.Audit.Site.DBPrefix = "wp_"
	bundle.Audit.Content.Posts = 120
	bundle.Audit.Content.Pages = 12
	bundle.Audit.Content.ApprovedComments = 22
	bundle.Audit.Content.Users = 4
	bundle.Audit.Plugins.HasSEO = true
	bundle.Audit.Plugins.HasACF = true
	bundle.Audit.Customization.CustomPostTypeCount = 1
	bundle.Audit.Customization.SerializedMetaCount = 9
	bundle.Audit.Customization.SEOMetaCount = 11
	bundle.Audit.Customization.ShortcodePostCount = 5
	bundle.Audit.Customization.OversizedContentCount = 3
	bundle.Audit.Uploads.Exists = true
	bundle.Audit.Uploads.PostsWithUploadsPaths = 7
	bundle.Audit.Uploads.PostsWithHTTPURLs = 2
	if err := reporting.WriteAll(outDir, bundle); err != nil {
		t.Fatalf("write summary bundle: %v", err)
	}

	res, err := RunDBPlan(DBPlanParams{
		From:   summaryPath,
		OutDir: outDir,
		Preset: "small-production",
		Write:  true,
	})
	if err != nil {
		t.Fatalf("RunDBPlan: %v", err)
	}
	if len(res.Plan.Tables) == 0 {
		t.Fatal("tables: want entries, got 0")
	}
	if len(res.Plan.Risks) == 0 {
		t.Fatal("risks: want entries, got 0")
	}
	if !containsSubstring(res.Plan.Risks, "SQLITE_TOOBIG") {
		t.Errorf("risks missing D1 statement-size warning for oversized posts: %v", res.Plan.Risks)
	}
	if len(res.Plan.TargetNotes) == 0 {
		t.Fatal("target notes: want entries, got 0")
	}
	if !containsSubstring(res.Plan.TargetNotes, "passkey") {
		t.Errorf("target notes missing auth-table warning: %v", res.Plan.TargetNotes)
	}
	if !containsSubstring(res.Plan.TargetNotes, "media completeness") {
		t.Errorf("target notes missing media completeness gate: %v", res.Plan.TargetNotes)
	}
	if !containsSubstring(res.Plan.TargetNotes, "7 posts embed wp-content/uploads") {
		t.Errorf("target notes missing uploads rewrite note: %v", res.Plan.TargetNotes)
	}

	md, err := os.ReadFile(filepath.Join(outDir, "db-plan.md"))
	if err != nil {
		t.Fatalf("read markdown: %v", err)
	}
	if !strings.Contains(string(md), "# DB Migration Plan") {
		t.Fatalf("markdown missing heading:\n%s", string(md))
	}
	if !strings.Contains(string(md), "## Target notes (EmDash / Cloudflare)") {
		t.Fatalf("markdown missing target notes section:\n%s", string(md))
	}
}

func containsSubstring(items []string, sub string) bool {
	for _, s := range items {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
