package wpcli

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/sibukixxx/wp2emdash/internal/domain/contentverify"
)

type contentRow struct {
	ID              int    `json:"ID"`
	PostType        string `json:"post_type"`
	PostName        string `json:"post_name"`
	PostStatus      string `json:"post_status"`
	PostTitle       string `json:"post_title"`
	PostContent     string `json:"post_content"`
	PostExcerpt     string `json:"post_excerpt"`
	PostAuthor      any    `json:"post_author"`
	PostParent      any    `json:"post_parent"`
	PostDateGMT     string `json:"post_date_gmt"`
	PostModifiedGMT string `json:"post_modified_gmt"`
}

// Snapshot implements source.ContentSnapshotter using WP-CLI. WordPress is
// deliberately read independently of EmDash's importer so shared importer
// bugs cannot make verification pass.
func (a *Auditor) Snapshot(ctx context.Context, mapping contentverify.Map) (contentverify.Snapshot, error) {
	types := []string{"post", "page"}
	for _, cm := range mapping.Collections {
		types = append(types, cm.WordPressPostType)
	}
	types = unique(types)
	var rows []contentRow
	for page := 1; ; page++ {
		out := a.wp(ctx, "content.snapshot", "post", "list",
			"--post_type="+strings.Join(types, ","),
			"--post_status=publish,draft,pending,private,future",
			"--posts_per_page=500", "--paged="+strconv.Itoa(page),
			"--fields=ID,post_type,post_name,post_status,post_title,post_content,post_excerpt,post_author,post_parent,post_date_gmt,post_modified_gmt",
			"--format=json")
		if out == "" {
			return contentverify.Snapshot{}, fmt.Errorf("wordpress content snapshot page %d returned no JSON", page)
		}
		var batch []contentRow
		if err := json.Unmarshal([]byte(out), &batch); err != nil {
			return contentverify.Snapshot{}, fmt.Errorf("decode wordpress content snapshot page %d: %w", page, err)
		}
		rows = append(rows, batch...)
		if len(batch) < 500 {
			break
		}
	}
	s := contentverify.Snapshot{SchemaVersion: 1, GeneratedAt: time.Now().UTC().Format(time.RFC3339), Tool: "wp2emdash", Source: "wordpress", Complete: true, Entries: make([]contentverify.Entry, 0, len(rows))}
	s.SiteURL = a.wp(ctx, "content.snapshot.site_url", "option", "get", "home")
	byID := make(map[int]int, len(rows))
	for _, row := range rows {
		created := wpTime(row.PostDateGMT)
		e := contentverify.Entry{ID: strconv.Itoa(row.ID), Kind: row.PostType, Slug: row.PostName, Status: row.PostStatus, Title: row.PostTitle, ExcerptHash: contentverify.HashString(row.PostExcerpt), Author: fmt.Sprint(row.PostAuthor), ParentID: fmt.Sprint(row.PostParent), CreatedAt: created, UpdatedAt: wpTime(row.PostModifiedGMT), Body: contentverify.FingerprintHTML(row.PostContent)}
		if row.PostStatus == "publish" {
			e.PublishedAt = created
		}
		s.Entries = append(s.Entries, e)
		byID[row.ID] = len(s.Entries) - 1
	}
	if err := a.collectMappedContentFields(ctx, mapping, byID, &s); err != nil {
		return contentverify.Snapshot{}, err
	}
	for _, warning := range a.Warnings() {
		s.Complete = false
		s.Warnings = append(s.Warnings, warning.Code+": "+warning.Message)
	}
	return s, nil
}

func (a *Auditor) collectMappedContentFields(ctx context.Context, mapping contentverify.Map, byID map[int]int, snap *contentverify.Snapshot) error {
	targetByMeta := map[string]string{}
	for _, cm := range mapping.Collections {
		for sourceKey, targetField := range cm.Meta {
			targetByMeta[sourceKey] = targetField
		}
	}
	if len(targetByMeta) == 0 {
		return nil
	}
	keys := make([]string, 0, len(targetByMeta))
	for key := range targetByMeta {
		keys = append(keys, "'"+strings.ReplaceAll(key, "'", "''")+"'")
	}
	sort.Strings(keys)
	prefix := strings.TrimSpace(a.wp(ctx, "content.snapshot.db_prefix", "db", "prefix"))
	if !validDBPrefix.MatchString(prefix) {
		return fmt.Errorf("invalid WordPress database prefix %q", prefix)
	}
	query := fmt.Sprintf("SELECT JSON_OBJECT('post_id', post_id, 'meta_key', meta_key, 'meta_value', meta_value) FROM %spostmeta WHERE meta_key IN (%s) ORDER BY post_id, meta_key", prefix, strings.Join(keys, ","))
	raw := a.wp(ctx, "content.snapshot.meta", "db", "query", query, "--skip-column-names")
	if raw == "" {
		return nil
	}
	for lineNo, line := range strings.Split(raw, "\n") {
		var row struct {
			PostID    int    `json:"post_id"`
			MetaKey   string `json:"meta_key"`
			MetaValue any    `json:"meta_value"`
		}
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			return fmt.Errorf("decode mapped postmeta line %d: %w", lineNo+1, err)
		}
		idx, ok := byID[row.PostID]
		if !ok {
			continue
		}
		if snap.Entries[idx].Fields == nil {
			snap.Entries[idx].Fields = map[string]string{}
		}
		encoded, _ := json.Marshal(row.MetaValue)
		snap.Entries[idx].Fields[targetByMeta[row.MetaKey]] = contentverify.HashString(string(encoded))
	}
	return nil
}

func wpTime(v string) string {
	v = strings.TrimSpace(v)
	if v == "" || strings.HasPrefix(v, "0000-") {
		return ""
	}
	if len(v) == 19 {
		return strings.Replace(v, " ", "T", 1) + "Z"
	}
	return v
}
func unique(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, v := range in {
		if v != "" && !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	sort.Strings(out)
	return out
}
