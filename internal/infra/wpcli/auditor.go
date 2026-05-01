// Package wpcli implements the WordPress-side data collection used by
// `wp2emdash audit`. Everything in here ultimately shells out to wp-cli, so
// the tool can be run on any host that already has WordPress + WP-CLI set up.
package wpcli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/sibukixxx/wp2emdash/internal/domain/audit"
	"github.com/sibukixxx/wp2emdash/internal/domain/source"
	"github.com/sibukixxx/wp2emdash/internal/shell"
)

// Auditor is a stateful collector. WPRoot must be a WordPress install root
// (the directory containing wp-config.php).
type Auditor struct {
	WPRoot   string
	Runner   shell.Runner
	remote   *RemoteConfig
	warnings []source.Warning
	seen     map[string]struct{}
}

// New returns an Auditor pinned to wpRoot. The path is validated lazily when
// Run is called so configuring the auditor from CLI flags is cheap.
func NewAuditor(wpRoot string) (*Auditor, error) {
	abs, err := filepath.Abs(wpRoot)
	if err != nil {
		return nil, fmt.Errorf("wp_root: %w", err)
	}
	return &Auditor{
		WPRoot: abs,
		Runner: shell.Runner{Dir: abs},
		seen:   make(map[string]struct{}),
	}, nil
}

// NewRemoteAuditor returns an Auditor that executes probes over SSH.
func NewRemoteAuditor(cfg RemoteConfig) (*Auditor, error) {
	if strings.TrimSpace(cfg.Target) == "" {
		return nil, fmt.Errorf("ssh target is required")
	}
	if strings.TrimSpace(cfg.WPRoot) == "" {
		return nil, fmt.Errorf("wp_root is required")
	}
	return &Auditor{
		WPRoot: cfg.WPRoot,
		Runner: shell.Runner{},
		remote: &cfg,
		seen:   make(map[string]struct{}),
	}, nil
}

// Run produces a complete Audit. Errors from individual probes are tolerated
// — missing data shows up as zero values so downstream scoring still runs.
func (a *Auditor) Run(ctx context.Context) (audit.Audit, error) {
	if a.isRemote() {
		if !a.remoteFileExists(ctx, "site.wp_config_exists", path.Join(a.WPRoot, "wp-config.php")) {
			return audit.Audit{}, fmt.Errorf("wp-config.php not found in %s on %s", a.WPRoot, a.remote.Target)
		}
	} else {
		if _, err := os.Stat(filepath.Join(a.WPRoot, "wp-config.php")); err != nil {
			return audit.Audit{}, fmt.Errorf("wp-config.php not found in %s", a.WPRoot)
		}
	}

	out := audit.Audit{}
	out.Site = a.collectSite(ctx)
	out.Content = a.collectContent(ctx)
	out.Uploads = a.collectUploads(ctx, out.Site.DBPrefix)
	out.Theme = a.collectTheme(ctx, out.Site.HomeURL)
	out.Plugins = a.collectPlugins(ctx)
	out.Customization = a.collectCustomization(ctx, out.Site.DBPrefix)
	return out, nil
}

func (a *Auditor) Warnings() []source.Warning {
	if len(a.warnings) == 0 {
		return nil
	}
	out := make([]source.Warning, len(a.warnings))
	copy(out, a.warnings)
	return out
}

func (a *Auditor) wp(ctx context.Context, code string, args ...string) string {
	if a.isRemote() {
		return a.remoteWP(ctx, code, args...)
	}
	out, err := a.Runner.Output(ctx, "wp", args...)
	if err != nil {
		a.warnf(code, "wp %s failed: %v", strings.Join(args, " "), err)
		return ""
	}
	return out
}

func (a *Auditor) wpInt(ctx context.Context, code string, args ...string) int {
	out := a.wp(ctx, code, args...)
	out = strings.TrimSpace(out)
	if out == "" {
		return 0
	}
	n, err := strconv.Atoi(out)
	if err != nil {
		a.warnf(code, "wp %s returned non-integer output: %q", strings.Join(args, " "), out)
		return 0
	}
	return n
}

func (a *Auditor) wpDBQuery(ctx context.Context, code, sql string) string {
	return a.wp(ctx, code, "db", "query", sql, "--skip-column-names")
}

func (a *Auditor) wpDBQueryInt(ctx context.Context, code, sql string) int {
	out := strings.TrimSpace(a.wpDBQuery(ctx, code, sql))
	if out == "" {
		return 0
	}
	n, err := strconv.Atoi(out)
	if err != nil {
		a.warnf(code, "wp db query returned non-integer output: %q", out)
		return 0
	}
	return n
}

func (a *Auditor) collectSite(ctx context.Context) audit.SiteInfo {
	prefix := a.wp(ctx, "site.db_prefix", "db", "prefix")
	if prefix == "" {
		a.warnf("site.db_prefix.defaulted", "wp db prefix probe failed; defaulting to wp_")
		prefix = "wp_"
	}
	return audit.SiteInfo{
		HomeURL:     a.wp(ctx, "site.home_url", "option", "get", "home"),
		SiteURL:     a.wp(ctx, "site.site_url", "option", "get", "siteurl"),
		WPVersion:   a.wp(ctx, "site.wp_version", "core", "version"),
		PHPVersion:  a.wp(ctx, "site.php_version", "eval", "echo PHP_VERSION;"),
		DBPrefix:    prefix,
		IsMultisite: a.wp(ctx, "site.is_multisite", "eval", `echo is_multisite() ? "yes" : "no";`),
	}
}

func (a *Auditor) collectContent(ctx context.Context) audit.ContentStats {
	return audit.ContentStats{
		Posts:            a.wpInt(ctx, "content.posts", "post", "list", "--post_type=post", "--post_status=publish", "--format=count"),
		Pages:            a.wpInt(ctx, "content.pages", "post", "list", "--post_type=page", "--post_status=publish", "--format=count"),
		Drafts:           a.wpInt(ctx, "content.drafts", "post", "list", "--post_status=draft", "--format=count"),
		PrivatePosts:     a.wpInt(ctx, "content.private_posts", "post", "list", "--post_status=private", "--format=count"),
		Categories:       a.wpInt(ctx, "content.categories", "term", "list", "category", "--format=count"),
		Tags:             a.wpInt(ctx, "content.tags", "term", "list", "post_tag", "--format=count"),
		Users:            a.wpInt(ctx, "content.users", "user", "list", "--format=count"),
		ApprovedComments: a.wpInt(ctx, "content.approved_comments", "comment", "list", "--status=approve", "--format=count"),
	}
}

func (a *Auditor) collectUploads(ctx context.Context, prefix string) audit.UploadsStats {
	stats := audit.UploadsStats{}
	uploadsDir := a.wpPath("wp-content", "uploads")
	if a.isRemote() {
		size, count, exists := a.remoteDirSizeAndCount(ctx, "uploads.dir_stats", uploadsDir)
		if exists {
			stats.Exists = true
			stats.Size = humanSize(size)
			stats.FileCount = count
		}
	} else if info, err := os.Stat(uploadsDir); err == nil && info.IsDir() {
		stats.Exists = true
		size, count := dirSizeAndCount(uploadsDir)
		stats.Size = humanSize(size)
		stats.FileCount = count
	}
	stats.PostsWithUploadsPaths = a.wpDBQueryInt(ctx, "uploads.paths_in_posts", fmt.Sprintf(
		"SELECT COUNT(*) FROM %sposts WHERE post_content LIKE '%%wp-content/uploads%%'", prefix))
	stats.PostsWithHTTPURLs = a.wpDBQueryInt(ctx, "uploads.http_urls_in_posts", fmt.Sprintf(
		"SELECT COUNT(*) FROM %sposts WHERE post_content LIKE '%%http://%%'", prefix))
	return stats
}

func (a *Auditor) collectTheme(ctx context.Context, _ string) audit.ThemeStats {
	stats := audit.ThemeStats{ActiveTheme: a.wp(ctx, "theme.active", "theme", "list", "--status=active", "--field=name")}
	if stats.ActiveTheme == "" {
		return stats
	}
	themeDir := a.wpPath("wp-content", "themes", stats.ActiveTheme)
	if a.isRemote() {
		if !a.remoteDirExists(ctx, "theme.dir_exists", themeDir) {
			return stats
		}
		stats.PHPFiles = a.remoteCountFilesByExt(ctx, "theme.php_files", themeDir, ".php")
		stats.CSSFiles = a.remoteCountFilesByExt(ctx, "theme.css_files", themeDir, ".css")
		stats.JSFiles = a.remoteCountFilesByExt(ctx, "theme.js_files", themeDir, ".js")
		stats.PageTemplates = a.remoteGrepCount(ctx, "theme.page_templates", themeDir, "Template Name:")
		stats.HookLikeOccurrences = a.remoteGrepCount(ctx, "theme.hooks", themeDir,
			"add_action", "add_filter", "register_post_type", "register_taxonomy",
			"add_shortcode", "register_rest_route", "add_meta_box",
			"wp_schedule_event", "wp_remote_")
		stats.JQueryLikeOccurrences = a.remoteGrepCount(ctx, "theme.jquery", themeDir,
			"jquery", "admin-ajax.php", "slick", "swiper", "owlCarousel")
		return stats
	}
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

func (a *Auditor) collectPlugins(ctx context.Context) audit.PluginsStats {
	stats := audit.PluginsStats{}
	raw := a.wp(ctx, "plugins.active_json", "plugin", "list", "--status=active", "--format=json")
	if raw == "" {
		return stats
	}
	var rows []pluginListJSONRow
	if err := json.Unmarshal([]byte(raw), &rows); err != nil {
		a.warnf("plugins.active_json.invalid", "wp plugin list returned invalid JSON: %v", err)
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

func (a *Auditor) collectCustomization(ctx context.Context, prefix string) audit.CustomStats {
	stats := audit.CustomStats{}

	// Custom post types / taxonomies (excluding core ones).
	cptOut := a.wp(ctx, "customization.post_types", "post-type", "list", "--field=name")
	stats.CustomPostTypeCount = countNonCore(cptOut, []string{
		"post", "page", "attachment", "revision", "nav_menu_item",
		"custom_css", "customize_changeset", "oembed_cache", "user_request",
		"wp_block", "wp_template", "wp_template_part", "wp_global_styles", "wp_navigation",
	})
	taxOut := a.wp(ctx, "customization.taxonomies", "taxonomy", "list", "--field=name")
	stats.CustomTaxonomyCount = countNonCore(taxOut, []string{
		"category", "post_tag", "nav_menu", "link_category", "post_format", "wp_theme",
	})

	// mu-plugins.
	muDir := a.wpPath("wp-content", "mu-plugins")
	if a.isRemote() {
		if a.remoteDirExists(ctx, "customization.mu_plugins.exists", muDir) {
			stats.MUPluginCount = a.remoteCountFilesByExt(ctx, "customization.mu_plugins.count", muDir, ".php")
			stats.MUPluginHookLikeOccurrences = a.remoteGrepCount(ctx, "customization.mu_plugins.hooks", muDir,
				"add_action", "add_filter", "register_post_type", "register_taxonomy",
				"wp_remote_", "wp_redirect", "register_rest_route")
		}
	} else if info, err := os.Stat(muDir); err == nil && info.IsDir() {
		stats.MUPluginCount = countFilesByExt(muDir, ".php")
		stats.MUPluginHookLikeOccurrences = grepCount(muDir,
			"add_action", "add_filter", "register_post_type", "register_taxonomy",
			"wp_remote_", "wp_redirect", "register_rest_route")
	}

	// SQL-driven counts.
	stats.SEOMetaCount = a.wpDBQueryInt(ctx, "customization.seo_meta", fmt.Sprintf(
		"SELECT COUNT(*) FROM %spostmeta WHERE meta_key LIKE '%%yoast%%' OR meta_key LIKE '%%rank_math%%' OR meta_key LIKE '%%aioseo%%'", prefix))
	stats.SerializedMetaCount = a.wpDBQueryInt(ctx, "customization.serialized_meta", fmt.Sprintf(
		"SELECT COUNT(*) FROM %spostmeta WHERE meta_value LIKE 'a:%%' OR meta_value LIKE 'O:%%'", prefix))
	stats.ShortcodePostCount = a.wpDBQueryInt(ctx, "customization.shortcode_posts", fmt.Sprintf(
		`SELECT COUNT(*) FROM %sposts WHERE post_content REGEXP '\\[[a-zA-Z0-9_-]+'`, prefix))

	// .htaccess / theme code redirects / external integrations.
	htaccessPath := a.wpPath(".htaccess")
	if a.isRemote() {
		stats.HtaccessRedirectLikeLines = a.remoteCountLinesMatching(ctx, "customization.htaccess", htaccessPath, []string{"redirect", "rewrite"})
	} else if _, err := os.Stat(htaccessPath); err == nil {
		stats.HtaccessRedirectLikeLines = countLinesMatching(htaccessPath, []string{"redirect", "rewrite"})
	}

	roots := []string{
		a.wpPath("wp-content", "themes"),
		a.wpPath("wp-content", "plugins"),
		a.wpPath("wp-content", "mu-plugins"),
	}
	if a.isRemote() {
		stats.CodeRedirectLikeOccurrences = a.remoteGrepCountInRoots(ctx, "customization.code_redirects", roots,
			"wp_redirect", "header('Location", `header("Location`)
		stats.ExternalIntegrationOccurrences = a.remoteGrepCountInRoots(ctx, "customization.external_integrations", roots,
			"wp_remote_get", "wp_remote_post", "curl_init", "admin-ajax.php",
			"register_rest_route", "webhook", "stripe", "line", "slack", "mailchimp")
		return stats
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

func (a *Auditor) warnf(code, format string, args ...any) {
	if a.seen == nil {
		a.seen = make(map[string]struct{})
	}
	if _, ok := a.seen[code]; ok {
		return
	}
	a.seen[code] = struct{}{}
	a.warnings = append(a.warnings, source.Warning{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
	})
}

func (a *Auditor) wpPath(elem ...string) string {
	if a.isRemote() {
		return a.remotePath(elem...)
	}
	parts := make([]string, 0, len(elem)+1)
	parts = append(parts, a.WPRoot)
	parts = append(parts, elem...)
	return filepath.Join(parts...)
}
