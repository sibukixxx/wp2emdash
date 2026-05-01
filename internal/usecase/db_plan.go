package usecase

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sibukixxx/wp2emdash/internal/domain/audit"
	"github.com/sibukixxx/wp2emdash/internal/usecase/reporting"
)

type DBPlanParams struct {
	From   string
	OutDir string
	Preset string
	Write  bool
}

type DBPlanResult struct {
	Plan         DBPlan
	JSONPath     string
	MarkdownPath string
}

type DBPlan struct {
	GeneratedAt  string        `json:"generated_at"`
	Source       string        `json:"source"`
	TargetPreset string        `json:"target_preset"`
	Strategy     string        `json:"strategy"`
	Tables       []DBTablePlan `json:"tables"`
	Notes        []string      `json:"notes"`
	Risks        []string      `json:"risks"`
	NextActions  []string      `json:"next_actions"`
}

type DBTablePlan struct {
	Name   string `json:"name"`
	Action string `json:"action"`
	Reason string `json:"reason"`
}

func RunDBPlan(params DBPlanParams) (DBPlanResult, error) {
	bundle, err := LoadReportBundle(params.From)
	if err != nil {
		return DBPlanResult{}, fmt.Errorf("load summary: %w", err)
	}

	plan := buildDBPlan(bundle, params)
	result := DBPlanResult{
		Plan:         plan,
		JSONPath:     filepath.Join(params.OutDir, "db-plan.json"),
		MarkdownPath: filepath.Join(params.OutDir, "db-plan.md"),
	}
	if !params.Write {
		return result, nil
	}
	if err := writeDBPlan(result); err != nil {
		return DBPlanResult{}, err
	}
	return result, nil
}

func buildDBPlan(bundle reporting.Bundle, params DBPlanParams) DBPlan {
	a := bundle.Audit
	notes := []string{
		fmt.Sprintf("Base table prefix is %s.", a.Site.DBPrefix),
		fmt.Sprintf("Content volume: %d posts, %d pages, %d approved comments.", a.Content.Posts, a.Content.Pages, a.Content.ApprovedComments),
	}
	risks := make([]string, 0, 6)
	actions := []string{
		"Decide whether comments will be migrated, archived, or dropped before import design starts.",
		"List postmeta keys to keep, transform, or omit before writing import code.",
		"Run content export/import on a staging workspace and compare record counts.",
	}

	if a.Site.IsMultisite == "yes" {
		risks = append(risks, "Multisite detected. Table selection and content mapping need tenant-aware handling.")
	}
	if a.Customization.SerializedMetaCount > 0 {
		risks = append(risks, fmt.Sprintf("Serialized meta detected (%d rows). Blind SQL export/import is likely unsafe.", a.Customization.SerializedMetaCount))
	}
	if a.Customization.CustomPostTypeCount > 0 || a.Customization.CustomTaxonomyCount > 0 {
		notes = append(notes, fmt.Sprintf("Custom structures detected: %d custom post types, %d custom taxonomies.", a.Customization.CustomPostTypeCount, a.Customization.CustomTaxonomyCount))
		actions = append(actions, "Define EmDash-side content models for custom post types and taxonomies before import.")
	}
	if a.Plugins.HasSEO || a.Customization.SEOMetaCount > 0 {
		risks = append(risks, fmt.Sprintf("SEO plugin/meta usage detected (%d SEO meta rows). Preserve canonical/meta/redirect ownership explicitly.", a.Customization.SEOMetaCount))
		actions = append(actions, "Map Yoast / Rank Math / AIOSEO fields to EmDash metadata or export them into a sidecar dataset.")
	}
	if a.Plugins.HasACF {
		risks = append(risks, "ACF detected. Field groups and meta schemas should be reviewed before import.")
	}
	if a.Plugins.HasWooCommerce {
		risks = append(risks, "WooCommerce detected. Orders, products, and transactional state should not be treated as a simple page/post migration.")
	}
	if a.Plugins.HasMember {
		risks = append(risks, "Membership plugin detected. User/account migration needs a separate auth plan.")
	}
	if a.Plugins.HasMultilingual {
		risks = append(risks, "Multilingual plugin detected. Locale linkage and URL strategy need explicit mapping.")
	}
	if a.Customization.ShortcodePostCount > 0 {
		risks = append(risks, fmt.Sprintf("Shortcodes detected in %d posts. Exported post bodies may need preprocessing.", a.Customization.ShortcodePostCount))
	}
	if a.Customization.ExternalIntegrationOccurrences > 0 {
		risks = append(risks, fmt.Sprintf("External integration markers detected (%d). Some content may depend on data not stored in WordPress tables alone.", a.Customization.ExternalIntegrationOccurrences))
	}

	strategy := "Export core WordPress content tables, treat plugin/meta tables as review-first, and transform postmeta before final import."
	if len(risks) == 0 {
		strategy = "Export the core WordPress content tables and validate row counts on staging before cutover."
	}

	return DBPlan{
		GeneratedAt:  time.Now().UTC().Format(time.RFC3339),
		Source:       params.From,
		TargetPreset: params.Preset,
		Strategy:     strategy,
		Tables: []DBTablePlan{
			{Name: a.Site.DBPrefix + "posts", Action: "export", Reason: "Primary source for posts, pages, attachments, and custom post types."},
			{Name: a.Site.DBPrefix + "postmeta", Action: "transform", Reason: "Post metadata often contains SEO, ACF, and serialized values that need mapping."},
			{Name: a.Site.DBPrefix + "terms", Action: "export", Reason: "Taxonomy labels for categories, tags, and custom taxonomies."},
			{Name: a.Site.DBPrefix + "term_taxonomy", Action: "export", Reason: "Taxonomy structure and parentage."},
			{Name: a.Site.DBPrefix + "term_relationships", Action: "export", Reason: "Post-to-taxonomy assignments."},
			{Name: a.Site.DBPrefix + "users", Action: actionForUsers(a), Reason: "Needed only when authorship, members, or comments are preserved."},
			{Name: a.Site.DBPrefix + "usermeta", Action: actionForUsermeta(a), Reason: "Review only the keys that must survive the migration."},
			{Name: a.Site.DBPrefix + "options", Action: "review", Reason: "Mostly environment-specific; migrate only selected site settings."},
			{Name: a.Site.DBPrefix + "comments", Action: actionForComments(a), Reason: "Optional depending on whether historic comments stay public."},
			{Name: a.Site.DBPrefix + "commentmeta", Action: actionForComments(a), Reason: "Only relevant when comments are migrated."},
		},
		Notes:       notes,
		Risks:       risks,
		NextActions: dedupeStrings(actions),
	}
}

func actionForUsers(a audit.Audit) string {
	if a.Content.Users > 0 || a.Plugins.HasMember {
		return "review"
	}
	return "skip"
}

func actionForUsermeta(a audit.Audit) string {
	if a.Content.Users > 0 || a.Plugins.HasMember {
		return "review"
	}
	return "skip"
}

func actionForComments(a audit.Audit) string {
	if a.Content.ApprovedComments > 0 {
		return "review"
	}
	return "skip"
}

func writeDBPlan(result DBPlanResult) error {
	if err := os.MkdirAll(filepath.Dir(result.JSONPath), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(result.JSONPath), err)
	}
	if err := writeDBPlanJSON(result.JSONPath, result.Plan); err != nil {
		return err
	}
	if err := writeDBPlanMarkdown(result.MarkdownPath, result.Plan); err != nil {
		return err
	}
	return nil
}

func writeDBPlanJSON(path string, plan DBPlan) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(plan)
}

func writeDBPlanMarkdown(path string, plan DBPlan) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()

	var sb strings.Builder
	sb.WriteString("# DB Migration Plan\n\n")
	sb.WriteString(fmt.Sprintf("- Generated: %s\n", plan.GeneratedAt))
	sb.WriteString(fmt.Sprintf("- Source summary: %s\n", plan.Source))
	sb.WriteString(fmt.Sprintf("- Target preset: %s\n", plan.TargetPreset))
	sb.WriteString(fmt.Sprintf("- Strategy: %s\n\n", plan.Strategy))
	sb.WriteString("## Tables\n\n")
	for _, table := range plan.Tables {
		sb.WriteString(fmt.Sprintf("- `%s`: %s — %s\n", table.Name, table.Action, table.Reason))
	}
	if len(plan.Notes) > 0 {
		sb.WriteString("\n## Notes\n\n")
		for _, note := range plan.Notes {
			sb.WriteString(fmt.Sprintf("- %s\n", note))
		}
	}
	if len(plan.Risks) > 0 {
		sb.WriteString("\n## Risks\n\n")
		for _, risk := range plan.Risks {
			sb.WriteString(fmt.Sprintf("- %s\n", risk))
		}
	}
	if len(plan.NextActions) > 0 {
		sb.WriteString("\n## Next actions\n\n")
		for i, action := range plan.NextActions {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, action))
		}
	}

	if _, err := f.WriteString(sb.String()); err != nil {
		return fmt.Errorf("write db plan markdown: %w", err)
	}
	return nil
}

func dedupeStrings(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
