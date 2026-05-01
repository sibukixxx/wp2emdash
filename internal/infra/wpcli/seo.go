package wpcli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/sibukixxx/wp2emdash/internal/domain/seo"
	"github.com/sibukixxx/wp2emdash/internal/shell"
)

// ExtractMeta collects per-post SEO metadata from the WordPress install.
// Implements source.MetaExtractor.
//
// The strategy:
//  1. List all published post / page entries with wp-cli.
//  2. Pull every relevant SEO postmeta key in a single wp db query.
//  3. Merge them in Go, preferring Yoast > Rank Math > AIOSEO when several
//     SEO plugins coexist (matches WP's own load-order tie-breaking).
func (a *Auditor) ExtractMeta(ctx context.Context) ([]seo.MetaItem, error) {
	site := a.collectSite(ctx)
	posts := a.listPostsForSEO(ctx)
	if len(posts) == 0 {
		return []seo.MetaItem{}, nil
	}
	rawMeta := a.collectSEOMeta(ctx, site.DBPrefix, postIDs(posts))

	items := make([]seo.MetaItem, 0, len(posts))
	for _, p := range posts {
		item := seo.MetaItem{
			PostID:   p.ID,
			PostType: p.PostType,
			Slug:     p.PostName,
			URL:      fallbackURL(p, site.HomeURL),
			Title:    p.PostTitle,
			Source:   "core",
		}
		seo.ApplyPostMeta(&item, rawMeta[p.ID])
		items = append(items, item)
	}
	return items, nil
}

// ExtractRedirects reads .htaccess and known plugin tables.
// Implements source.RedirectExtractor.
func (a *Auditor) ExtractRedirects(ctx context.Context) ([]seo.RedirectRule, error) {
	rules := make([]seo.RedirectRule, 0)

	htaccessPath := a.wpPath(".htaccess")
	body, ok := a.readHtaccess(ctx, htaccessPath)
	if ok {
		rules = append(rules, seo.ParseHtaccessRedirects(strings.NewReader(body))...)
	}

	site := a.collectSite(ctx)
	rules = append(rules, a.queryRedirectionPluginRules(ctx, site.DBPrefix)...)
	rules = append(rules, a.querySRMPluginRules(ctx, site.DBPrefix)...)
	return rules, nil
}

// postSummary mirrors the JSON shape of `wp post list ... --format=json`.
type postSummary struct {
	ID        int    `json:"ID"`
	PostType  string `json:"post_type"`
	PostName  string `json:"post_name"`
	PostTitle string `json:"post_title"`
	URL       string `json:"url"`
}

func (a *Auditor) listPostsForSEO(ctx context.Context) []postSummary {
	raw := a.wp(ctx, "seo.posts_list",
		"post", "list",
		"--post_status=publish",
		"--post_type=post,page",
		"--fields=ID,post_type,post_name,post_title,url",
		"--format=json",
	)
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var rows []postSummary
	if err := json.Unmarshal([]byte(raw), &rows); err != nil {
		a.warnf("seo.posts_list.invalid", "wp post list returned invalid JSON: %v", err)
		return nil
	}
	return rows
}

func postIDs(rows []postSummary) []int {
	ids := make([]int, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
	}
	return ids
}

// fallbackURL prefers the wp-cli "url" field when present, otherwise builds a
// best-effort fallback so downstream consumers always see a usable URL.
func fallbackURL(p postSummary, home string) string {
	if u := strings.TrimSpace(p.URL); u != "" {
		return u
	}
	if home == "" {
		return ""
	}
	if p.PostName != "" {
		return strings.TrimRight(home, "/") + "/" + p.PostName + "/"
	}
	return strings.TrimRight(home, "/") + "/?p=" + strconv.Itoa(p.ID)
}

// collectSEOMeta returns post_id -> meta_key -> meta_value for the SEO keys.
func (a *Auditor) collectSEOMeta(ctx context.Context, prefix string, ids []int) map[int]map[string]string {
	out := make(map[int]map[string]string)
	if len(ids) == 0 {
		return out
	}
	idList := make([]string, len(ids))
	for i, id := range ids {
		idList[i] = strconv.Itoa(id)
	}
	keys := seo.PostMetaKeys()
	keyList := make([]string, len(keys))
	for i, k := range keys {
		keyList[i] = "'" + strings.ReplaceAll(k, "'", "''") + "'"
	}
	query := fmt.Sprintf(
		"SELECT post_id, meta_key, meta_value FROM %spostmeta WHERE post_id IN (%s) AND meta_key IN (%s)",
		prefix,
		strings.Join(idList, ","),
		strings.Join(keyList, ","),
	)
	raw := a.wp(ctx, "seo.postmeta", "db", "query", query, "--skip-column-names")
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) != 3 {
			continue
		}
		id, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}
		if _, ok := out[id]; !ok {
			out[id] = make(map[string]string, 4)
		}
		out[id][parts[1]] = parts[2]
	}
	return out
}

// readHtaccess returns the .htaccess body. Returns false on miss/error so the
// caller can degrade gracefully (a missing .htaccess is normal).
func (a *Auditor) readHtaccess(ctx context.Context, htaccessPath string) (string, bool) {
	if a.isRemote() {
		out := a.remoteOutput(ctx, "seo.htaccess",
			"if [ -f "+shell.QuotePOSIX(htaccessPath)+" ]; then cat "+shell.QuotePOSIX(htaccessPath)+"; fi")
		if out == "" {
			return "", false
		}
		return out, true
	}
	body, err := os.ReadFile(htaccessPath)
	if err != nil {
		return "", false
	}
	return string(body), true
}

func (a *Auditor) queryRedirectionPluginRules(ctx context.Context, prefix string) []seo.RedirectRule {
	query := fmt.Sprintf(
		"SELECT url, action_data, action_code, regex FROM %sredirection_items WHERE status='enabled'",
		prefix,
	)
	raw := a.wp(ctx, "seo.redirection_items", "db", "query", query, "--skip-column-names")
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var rules []seo.RedirectRule
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) < 3 {
			continue
		}
		code := 302
		if c, err := strconv.Atoi(strings.TrimSpace(parts[2])); err == nil && c > 0 {
			code = c
		}
		match := "exact"
		if len(parts) >= 4 && strings.TrimSpace(parts[3]) == "1" {
			match = "regex"
		}
		rules = append(rules, seo.RedirectRule{
			From:   parts[0],
			To:     parts[1],
			Code:   code,
			Match:  match,
			Source: "redirection",
		})
	}
	return rules
}

func (a *Auditor) querySRMPluginRules(ctx context.Context, prefix string) []seo.RedirectRule {
	// Safe Redirect Manager stores entries as posts with post_type='redirect_rule'.
	query := fmt.Sprintf(`SELECT p.ID, p.post_title, MAX(CASE WHEN pm.meta_key='_redirect_rule_to' THEN pm.meta_value END) AS to_url, MAX(CASE WHEN pm.meta_key='_redirect_rule_status_code' THEN pm.meta_value END) AS code FROM %sposts p LEFT JOIN %spostmeta pm ON pm.post_id=p.ID WHERE p.post_type='redirect_rule' AND p.post_status='publish' GROUP BY p.ID, p.post_title`, prefix, prefix)
	raw := a.wp(ctx, "seo.srm_items", "db", "query", query, "--skip-column-names")
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var rules []seo.RedirectRule
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) < 3 {
			continue
		}
		code := 302
		if len(parts) >= 4 {
			if c, err := strconv.Atoi(strings.TrimSpace(parts[3])); err == nil && c > 0 {
				code = c
			}
		}
		rules = append(rules, seo.RedirectRule{
			From:   strings.TrimSpace(parts[1]),
			To:     strings.TrimSpace(parts[2]),
			Code:   code,
			Match:  "exact",
			Source: "safe-redirect-manager",
		})
	}
	return rules
}
