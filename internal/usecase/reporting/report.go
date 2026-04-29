// Package report renders an Audit + Score into JSON / Markdown artifacts
// that downstream commands (and humans selling migrations) consume.
package reporting

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/rokubunnoni-inc/wp2emdash/internal/domain/audit"
	"github.com/rokubunnoni-inc/wp2emdash/internal/domain/score"
)

// Bundle is the umbrella struct written to summary.json. Keeping this in one
// type means downstream tools can consume a single JSON document instead of
// joining several files together.
type Bundle struct {
	GeneratedAt string         `json:"generated_at"`
	Tool        string         `json:"tool"`
	Version     string         `json:"version"`
	Audit       audit.Audit   `json:"audit"`
	Score       score.Result   `json:"score"`
}

// WriteAll writes summary.json + risk-report.md to outDir.
func WriteAll(outDir string, b Bundle) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", outDir, err)
	}
	if err := writeJSON(filepath.Join(outDir, "summary.json"), b); err != nil {
		return err
	}
	if err := writeMarkdown(filepath.Join(outDir, "risk-report.md"), b); err != nil {
		return err
	}
	return nil
}

func ReadBundle(path string) (Bundle, error) {
	f, err := os.Open(path)
	if err != nil {
		return Bundle{}, err
	}
	defer f.Close()

	var bundle Bundle
	if err := json.NewDecoder(f).Decode(&bundle); err != nil {
		return Bundle{}, err
	}
	return bundle, nil
}

func writeJSON(path string, v any) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func writeMarkdown(path string, b Bundle) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return RenderMarkdown(f, b)
}

// RenderMarkdown emits the human-facing report. Kept exported so the CLI
// can preview it on stdout without touching the filesystem.
func RenderMarkdown(w io.Writer, b Bundle) error {
	a := b.Audit
	s := b.Score

	var sb strings.Builder
	wln := func(format string, args ...any) {
		sb.WriteString(fmt.Sprintf(format, args...))
		sb.WriteByte('\n')
	}

	wln("# EmDash Migration Audit Report")
	wln("")
	wln("- Generated: %s", b.GeneratedAt)
	wln("- Tool: %s %s", b.Tool, b.Version)
	wln("- URL: %s", a.Site.HomeURL)
	wln("- WordPress: %s", a.Site.WPVersion)
	wln("- PHP: %s", a.Site.PHPVersion)
	wln("- Active theme: %s", a.Theme.ActiveTheme)
	wln("- **Risk score: %d (%s)**", s.Score, s.Level)
	wln("- Rough estimate: %s", s.Estimate)
	wln("")
	wln("## Content")
	wln("")
	wln("- Posts: %d", a.Content.Posts)
	wln("- Pages: %d", a.Content.Pages)
	wln("- Drafts: %d", a.Content.Drafts)
	wln("- Private posts: %d", a.Content.PrivatePosts)
	wln("- Categories: %d", a.Content.Categories)
	wln("- Tags: %d", a.Content.Tags)
	wln("- Users: %d", a.Content.Users)
	wln("- Approved comments: %d", a.Content.ApprovedComments)
	wln("")
	wln("## Uploads")
	wln("")
	wln("- Exists: %t", a.Uploads.Exists)
	wln("- Size: %s", a.Uploads.Size)
	wln("- File count: %d", a.Uploads.FileCount)
	wln("- Posts referencing wp-content/uploads: %d", a.Uploads.PostsWithUploadsPaths)
	wln("- Posts with http:// URLs: %d", a.Uploads.PostsWithHTTPURLs)
	wln("")
	wln("## Plugins")
	wln("")
	wln("- Active plugins: %d", a.Plugins.ActiveCount)
	wln("- ACF / custom-fields: %t", a.Plugins.HasACF)
	wln("- WooCommerce: %t", a.Plugins.HasWooCommerce)
	wln("- SEO: %t", a.Plugins.HasSEO)
	wln("- Form: %t", a.Plugins.HasForm)
	wln("- Redirect: %t", a.Plugins.HasRedirect)
	wln("- Member: %t", a.Plugins.HasMember)
	wln("- Multilingual: %t", a.Plugins.HasMultilingual)
	wln("- Cache: %t", a.Plugins.HasCache)
	wln("")
	wln("## Theme")
	wln("")
	wln("- PHP files: %d", a.Theme.PHPFiles)
	wln("- CSS files: %d", a.Theme.CSSFiles)
	wln("- JS files: %d", a.Theme.JSFiles)
	wln("- Page templates: %d", a.Theme.PageTemplates)
	wln("- Hook-like occurrences: %d", a.Theme.HookLikeOccurrences)
	wln("- jQuery-like occurrences: %d", a.Theme.JQueryLikeOccurrences)
	wln("")
	wln("## Customization")
	wln("")
	wln("- Custom post types: %d", a.Customization.CustomPostTypeCount)
	wln("- Custom taxonomies: %d", a.Customization.CustomTaxonomyCount)
	wln("- mu-plugin files: %d", a.Customization.MUPluginCount)
	wln("- mu-plugin hook occurrences: %d", a.Customization.MUPluginHookLikeOccurrences)
	wln("- Shortcode posts: %d", a.Customization.ShortcodePostCount)
	wln("- SEO meta count: %d", a.Customization.SEOMetaCount)
	wln("- Serialized meta count: %d", a.Customization.SerializedMetaCount)
	wln("- .htaccess redirect/rewrite lines: %d", a.Customization.HtaccessRedirectLikeLines)
	wln("- Code redirect occurrences: %d", a.Customization.CodeRedirectLikeOccurrences)
	wln("- External integration occurrences: %d", a.Customization.ExternalIntegrationOccurrences)
	wln("")
	wln("## Risk reasons")
	wln("")
	if len(s.Reasons) == 0 {
		wln("- No major risk items detected by this audit.")
	} else {
		for _, r := range s.Reasons {
			wln("- +%d (`%s`): %s", r.Points, r.Code, r.Text)
		}
	}
	wln("")
	wln("## Recommended next actions")
	wln("")
	wln("1. Decide on migration scope and choose the matching `wp2emdash preset`.")
	wln("2. Inspect `summary.json` for the raw metrics.")
	wln("3. For Complex+ levels, run `wp2emdash media scan --hash` and store the manifest.")
	wln("4. Decide whether old `/wp-content/uploads/` URLs must be preserved (affects routing).")
	wln("5. Decide which postmeta keys to migrate vs. drop (Yoast / Rank Math / ACF).")

	_, err := io.WriteString(w, sb.String())
	return err
}
