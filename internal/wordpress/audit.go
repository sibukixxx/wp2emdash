// Package wordpress implements the WordPress-side data collection used by
// `wp2emdash audit`. Everything in here ultimately shells out to wp-cli, so
// the tool can be run on any host that already has WordPress + WP-CLI set up.
package wordpress

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/rokubunnoni-inc/wp2emdash/internal/shell"
)

// Audit is the full structured audit payload produced for a WP install.
// It mirrors the fields in scripts/audit/emdash-migration-audit.sh so the
// scoring rubric stays identical between the bash and Go implementations.
type Audit struct {
	Site          SiteInfo      `json:"site"`
	Content       ContentStats  `json:"content"`
	Uploads       UploadsStats  `json:"uploads"`
	Theme         ThemeStats    `json:"theme"`
	Plugins       PluginsStats  `json:"plugins"`
	Customization CustomStats   `json:"customization"`
}

type SiteInfo struct {
	HomeURL     string `json:"home_url"`
	SiteURL     string `json:"site_url"`
	WPVersion   string `json:"wp_version"`
	PHPVersion  string `json:"php_version"`
	DBPrefix    string `json:"db_prefix"`
	IsMultisite string `json:"is_multisite"`
}

type ContentStats struct {
	Posts            int `json:"posts"`
	Pages            int `json:"pages"`
	Drafts           int `json:"drafts"`
	PrivatePosts    int `json:"private_posts"`
	Categories       int `json:"categories"`
	Tags             int `json:"tags"`
	Users            int `json:"users"`
	ApprovedComments int `json:"approved_comments"`
}

type UploadsStats struct {
	Exists                bool   `json:"exists"`
	Size                  string `json:"size"`
	FileCount             int    `json:"file_count"`
	PostsWithUploadsPaths int    `json:"posts_with_uploads_paths"`
	PostsWithHTTPURLs     int    `json:"posts_with_http_urls"`
}

type ThemeStats struct {
	ActiveTheme           string `json:"active_theme"`
	PHPFiles              int    `json:"php_files"`
	CSSFiles              int    `json:"css_files"`
	JSFiles               int    `json:"js_files"`
	PageTemplates         int    `json:"page_templates"`
	HookLikeOccurrences   int    `json:"hook_like_occurrences"`
	JQueryLikeOccurrences int    `json:"jquery_like_occurrences"`
}

type PluginsStats struct {
	ActiveCount     int  `json:"active_count"`
	HasACF          bool `json:"has_acf"`
	HasWooCommerce  bool `json:"has_woocommerce"`
	HasSEO          bool `json:"has_seo"`
	HasForm         bool `json:"has_form"`
	HasRedirect     bool `json:"has_redirect"`
	HasMember       bool `json:"has_member"`
	HasMultilingual bool `json:"has_multilingual"`
	HasCache        bool `json:"has_cache"`
}

type CustomStats struct {
	CustomPostTypeCount             int `json:"custom_post_type_count"`
	CustomTaxonomyCount             int `json:"custom_taxonomy_count"`
	MUPluginCount                   int `json:"mu_plugin_count"`
	MUPluginHookLikeOccurrences     int `json:"mu_plugin_hook_like_occurrences"`
	ShortcodePostCount              int `json:"shortcode_post_count"`
	SEOMetaCount                    int `json:"seo_meta_count"`
	SerializedMetaCount             int `json:"serialized_meta_count"`
	HtaccessRedirectLikeLines       int `json:"htaccess_redirect_like_lines"`
	CodeRedirectLikeOccurrences     int `json:"code_redirect_like_occurrences"`
	ExternalIntegrationOccurrences  int `json:"external_integration_like_occurrences"`
}

// Auditor is a stateful collector. WPRoot must be a WordPress install root
// (the directory containing wp-config.php).
type Auditor struct {
	WPRoot string
	Runner shell.Runner
}

// New returns an Auditor pinned to wpRoot. The path is validated lazily when
// Run is called so configuring the auditor from CLI flags is cheap.
func New(wpRoot string) (*Auditor, error) {
	abs, err := filepath.Abs(wpRoot)
	if err != nil {
		return nil, fmt.Errorf("wp_root: %w", err)
	}
	return &Auditor{WPRoot: abs, Runner: shell.Runner{Dir: abs}}, nil
}

// Run produces a complete Audit. Errors from individual probes are tolerated
// — missing data shows up as zero values so downstream scoring still runs.
func (a *Auditor) Run(ctx context.Context) (Audit, error) {
	if _, err := os.Stat(filepath.Join(a.WPRoot, "wp-config.php")); err != nil {
		return Audit{}, fmt.Errorf("wp-config.php not found in %s", a.WPRoot)
	}

	out := Audit{}
	out.Site = a.collectSite(ctx)
	out.Content = a.collectContent(ctx)
	out.Uploads = a.collectUploads(ctx, out.Site.DBPrefix)
	out.Theme = a.collectTheme(ctx, out.Site.HomeURL)
	out.Plugins = a.collectPlugins(ctx)
	out.Customization = a.collectCustomization(ctx, out.Site.DBPrefix)
	return out, nil
}

func (a *Auditor) wp(ctx context.Context, args ...string) string {
	out, err := a.Runner.Output(ctx, "wp", args...)
	if err != nil {
		return ""
	}
	return out
}

func (a *Auditor) wpInt(ctx context.Context, args ...string) int {
	out := a.wp(ctx, args...)
	out = strings.TrimSpace(out)
	if out == "" {
		return 0
	}
	n, err := strconv.Atoi(out)
	if err != nil {
		return 0
	}
	return n
}

func (a *Auditor) wpDBQuery(ctx context.Context, sql string) string {
	return a.wp(ctx, "db", "query", sql, "--skip-column-names")
}

func (a *Auditor) wpDBQueryInt(ctx context.Context, sql string) int {
	out := strings.TrimSpace(a.wpDBQuery(ctx, sql))
	if out == "" {
		return 0
	}
	n, err := strconv.Atoi(out)
	if err != nil {
		return 0
	}
	return n
}

func (a *Auditor) collectSite(ctx context.Context) SiteInfo {
	prefix := a.wp(ctx, "db", "prefix")
	if prefix == "" {
		prefix = "wp_"
	}
	return SiteInfo{
		HomeURL:     a.wp(ctx, "option", "get", "home"),
		SiteURL:     a.wp(ctx, "option", "get", "siteurl"),
		WPVersion:   a.wp(ctx, "core", "version"),
		PHPVersion:  a.wp(ctx, "eval", "echo PHP_VERSION;"),
		DBPrefix:    prefix,
		IsMultisite: a.wp(ctx, "eval", `echo is_multisite() ? "yes" : "no";`),
	}
}

func (a *Auditor) collectContent(ctx context.Context) ContentStats {
	return ContentStats{
		Posts:            a.wpInt(ctx, "post", "list", "--post_type=post", "--post_status=publish", "--format=count"),
		Pages:            a.wpInt(ctx, "post", "list", "--post_type=page", "--post_status=publish", "--format=count"),
		Drafts:           a.wpInt(ctx, "post", "list", "--post_status=draft", "--format=count"),
		PrivatePosts:     a.wpInt(ctx, "post", "list", "--post_status=private", "--format=count"),
		Categories:       a.wpInt(ctx, "term", "list", "category", "--format=count"),
		Tags:             a.wpInt(ctx, "term", "list", "post_tag", "--format=count"),
		Users:            a.wpInt(ctx, "user", "list", "--format=count"),
		ApprovedComments: a.wpInt(ctx, "comment", "list", "--status=approve", "--format=count"),
	}
}

func (a *Auditor) collectUploads(ctx context.Context, prefix string) UploadsStats {
	stats := UploadsStats{}
	uploadsDir := filepath.Join(a.WPRoot, "wp-content", "uploads")
	if info, err := os.Stat(uploadsDir); err == nil && info.IsDir() {
		stats.Exists = true
		size, count := dirSizeAndCount(uploadsDir)
		stats.Size = humanSize(size)
		stats.FileCount = count
	}
	stats.PostsWithUploadsPaths = a.wpDBQueryInt(ctx, fmt.Sprintf(
		"SELECT COUNT(*) FROM %sposts WHERE post_content LIKE '%%wp-content/uploads%%'", prefix))
	stats.PostsWithHTTPURLs = a.wpDBQueryInt(ctx, fmt.Sprintf(
		"SELECT COUNT(*) FROM %sposts WHERE post_content LIKE '%%http://%%'", prefix))
	return stats
}

func (a *Auditor) collectTheme(ctx context.Context, _ string) ThemeStats {
	stats := ThemeStats{ActiveTheme: a.wp(ctx, "theme", "list", "--status=active", "--field=name")}
	if stats.ActiveTheme == "" {
		return stats
	}
	themeDir := filepath.Join(a.WPRoot, "wp-content", "themes", stats.ActiveTheme)
	if _, err := os.Stat(themeDir); err != nil {
		return stats
	}
	stats.PHPFiles = countFilesByExt(themeDir, ".php")
	stats.CSSFiles = countFilesByExt(themeDir, ".css")
	stats.JSFiles = countFilesByExt(themeDir, ".js")
	stats.PageTemplates = grepCount(themeDir, "Template Name:")
	stats.HookLikeOccurrences = grepCount(themeDir,
		"add_action", "add_filter", "register_post_type", "register_taxonomy",
		"add_shortcode", "register_rest_route", "add_meta_box",
		"wp_schedule_event", "wp_remote_")
	stats.JQueryLikeOccurrences = grepCount(themeDir,
		"jquery", "admin-ajax.php", "slick", "swiper", "owlCarousel")
	return stats
}

// pluginListJSONRow matches the shape of `wp plugin list --format=json`.
type pluginListJSONRow struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

func (a *Auditor) collectPlugins(ctx context.Context) PluginsStats {
	stats := PluginsStats{}
	raw := a.wp(ctx, "plugin", "list", "--status=active", "--format=json")
	if raw == "" {
		return stats
	}
	var rows []pluginListJSONRow
	if err := json.Unmarshal([]byte(raw), &rows); err != nil {
		return stats
	}
	stats.ActiveCount = len(rows)

	// match plugin slugs against well-known categories
	matchAny := func(name string, patterns []string) bool {
		lc := strings.ToLower(name)
		for _, p := range patterns {
			if strings.Contains(lc, p) {
				return true
			}
		}
		return false
	}
	for _, p := range rows {
		switch {
		case matchAny(p.Name, []string{"advanced-custom-fields", "acf"}):
			stats.HasACF = true
		case matchAny(p.Name, []string{"woocommerce"}):
			stats.HasWooCommerce = true
		case matchAny(p.Name, []string{"wordpress-seo", "seo-by-rank-math", "all-in-one-seo", "aioseo"}):
			stats.HasSEO = true
		case matchAny(p.Name, []string{"contact-form-7", "mw-wp-form", "wpforms", "ninja-forms", "gravityforms"}):
			stats.HasForm = true
		case matchAny(p.Name, []string{"redirection", "safe-redirect-manager"}):
			stats.HasRedirect = true
		case matchAny(p.Name, []string{"ultimate-member", "paid-memberships-pro", "memberpress", "simple-membership"}):
			stats.HasMember = true
		case matchAny(p.Name, []string{"wpml", "polylang", "translatepress", "multilingualpress"}):
			stats.HasMultilingual = true
		case matchAny(p.Name, []string{"autoptimize", "wp-rocket", "w3-total-cache", "litespeed-cache", "wp-super-cache"}):
			stats.HasCache = true
		}
	}
	return stats
}

func (a *Auditor) collectCustomization(ctx context.Context, prefix string) CustomStats {
	stats := CustomStats{}

	// Custom post types / taxonomies (excluding core ones).
	cptOut := a.wp(ctx, "post-type", "list", "--field=name")
	stats.CustomPostTypeCount = countNonCore(cptOut, []string{
		"post", "page", "attachment", "revision", "nav_menu_item",
		"custom_css", "customize_changeset", "oembed_cache", "user_request",
		"wp_block", "wp_template", "wp_template_part", "wp_global_styles", "wp_navigation",
	})
	taxOut := a.wp(ctx, "taxonomy", "list", "--field=name")
	stats.CustomTaxonomyCount = countNonCore(taxOut, []string{
		"category", "post_tag", "nav_menu", "link_category", "post_format", "wp_theme",
	})

	// mu-plugins.
	muDir := filepath.Join(a.WPRoot, "wp-content", "mu-plugins")
	if info, err := os.Stat(muDir); err == nil && info.IsDir() {
		stats.MUPluginCount = countFilesByExt(muDir, ".php")
		stats.MUPluginHookLikeOccurrences = grepCount(muDir,
			"add_action", "add_filter", "register_post_type", "register_taxonomy",
			"wp_remote_", "wp_redirect", "register_rest_route")
	}

	// SQL-driven counts.
	stats.SEOMetaCount = a.wpDBQueryInt(ctx, fmt.Sprintf(
		"SELECT COUNT(*) FROM %spostmeta WHERE meta_key LIKE '%%yoast%%' OR meta_key LIKE '%%rank_math%%' OR meta_key LIKE '%%aioseo%%'", prefix))
	stats.SerializedMetaCount = a.wpDBQueryInt(ctx, fmt.Sprintf(
		"SELECT COUNT(*) FROM %spostmeta WHERE meta_value LIKE 'a:%%' OR meta_value LIKE 'O:%%'", prefix))
	stats.ShortcodePostCount = a.wpDBQueryInt(ctx, fmt.Sprintf(
		`SELECT COUNT(*) FROM %sposts WHERE post_content REGEXP '\\[[a-zA-Z0-9_-]+'`, prefix))

	// .htaccess / theme code redirects / external integrations.
	htaccessPath := filepath.Join(a.WPRoot, ".htaccess")
	if _, err := os.Stat(htaccessPath); err == nil {
		stats.HtaccessRedirectLikeLines = countLinesMatching(htaccessPath, []string{"redirect", "rewrite"})
	}

	roots := []string{
		filepath.Join(a.WPRoot, "wp-content", "themes"),
		filepath.Join(a.WPRoot, "wp-content", "plugins"),
		filepath.Join(a.WPRoot, "wp-content", "mu-plugins"),
	}
	stats.CodeRedirectLikeOccurrences = grepCountInRoots(roots,
		"wp_redirect", "header('Location", `header("Location`)
	stats.ExternalIntegrationOccurrences = grepCountInRoots(roots,
		"wp_remote_get", "wp_remote_post", "curl_init", "admin-ajax.php",
		"register_rest_route", "webhook", "stripe", "line", "slack", "mailchimp")

	return stats
}

// countNonCore counts whitespace-separated entries in `out` that are not in
// the `core` list. Used for CPT / taxonomy detection.
func countNonCore(out string, core []string) int {
	if out == "" {
		return 0
	}
	skip := make(map[string]struct{}, len(core))
	for _, c := range core {
		skip[c] = struct{}{}
	}
	count := 0
	for _, name := range strings.Fields(out) {
		if _, ok := skip[name]; ok {
			continue
		}
		count++
	}
	return count
}

// ErrWPCLIMissing is reported when `wp` cannot be located at all.
var ErrWPCLIMissing = errors.New("wp-cli not found in PATH")
