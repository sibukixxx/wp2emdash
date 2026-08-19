package contentverify

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

func Compare(expected, actual Snapshot, mapping Map) Report {
	report := Report{SchemaVersion: 1, OK: true, ExpectedHash: snapshotHash(expected), ActualHash: snapshotHash(actual)}
	report.Totals.Expected, report.Totals.Actual = len(expected.Entries), len(actual.Entries)
	if !expected.Complete || !actual.Complete {
		addIssue(&report, Issue{Severity: SeverityCritical, Code: "snapshot_incomplete", Message: "one or both snapshots are incomplete"})
	}
	actualByID := map[string]int{}
	actualBySlug := map[string][]int{}
	for i, e := range actual.Entries {
		actualByID[e.Kind+"\x00"+e.ID] = i
		actualBySlug[key(e.Kind, e.Locale, e.Slug)] = append(actualBySlug[key(e.Kind, e.Locale, e.Slug)], i)
	}
	used := map[int]bool{}
	for _, exp := range expected.Entries {
		if _, mapped := collectionFor(exp.Kind, mapping); !mapped {
			report.Totals.Missing++
			addIssue(&report, Issue{Severity: SeverityCritical, Code: "unmapped_post_type", Message: "WordPress post type has no EmDash collection mapping", SourceID: exp.ID, Kind: exp.Kind, Slug: exp.Slug})
			continue
		}
		idx, method, ok := resolve(exp, actual.Entries, actualByID, actualBySlug, mapping)
		if !ok {
			report.Totals.Missing++
			addIssue(&report, Issue{Severity: SeverityCritical, Code: "missing_entry", Message: "expected entry is missing or has no unique mapping", SourceID: exp.ID, Kind: exp.Kind, Slug: exp.Slug})
			continue
		}
		if used[idx] {
			addIssue(&report, Issue{Severity: SeverityCritical, Code: "duplicate_mapping", Message: "multiple source entries map to one target entry", SourceID: exp.ID, TargetID: actual.Entries[idx].ID, Kind: exp.Kind, Slug: exp.Slug})
			continue
		}
		used[idx] = true
		act := actual.Entries[idx]
		report.Totals.Matched++
		report.Matches = append(report.Matches, Match{SourceID: exp.ID, TargetID: act.ID, Kind: exp.Kind, Locale: act.Locale, Method: method})
		if method == "slug" {
			addIssue(&report, issue(SeverityWarning, "fallback_identity_match", "entry was matched by collection, locale, and slug", exp, act, ""))
		}
		compareEntry(&report, exp, act)
	}
	for i, act := range actual.Entries {
		if !used[i] {
			report.Totals.Extra++
			addIssue(&report, Issue{Severity: SeverityWarning, Code: "extra_entry", Message: "target entry has no source match", TargetID: act.ID, Kind: act.Kind, Slug: act.Slug})
		}
	}
	sort.Slice(report.Issues, func(i, j int) bool {
		a, b := report.Issues[i], report.Issues[j]
		return string(a.Severity)+a.Code+a.SourceID < string(b.Severity)+b.Code+b.SourceID
	})
	report.OK = report.Totals.Critical == 0 && report.Totals.Errors == 0
	return report
}

func resolve(exp Entry, actual []Entry, byID map[string]int, bySlug map[string][]int, m Map) (int, string, bool) {
	wid, err := strconv.Atoi(exp.ID)
	if err == nil {
		for _, em := range m.Entries {
			if em.WordPressID == wid {
				if i, ok := byID[em.EmDashCollection+"\x00"+em.EmDashID]; ok {
					return i, "explicit", true
				}
			}
		}
	}
	cm, ok := collectionFor(exp.Kind, m)
	if !ok {
		return 0, "", false
	}
	if cm.TargetSourceIDField != "" {
		want := HashString(exp.ID)
		for i, act := range actual {
			if act.Kind == cm.EmDashCollection && act.Fields[cm.TargetSourceIDField] == want {
				return i, "source_field", true
			}
		}
	}
	if cm.DeterministicIDTemplate != "" {
		id := strings.ReplaceAll(cm.DeterministicIDTemplate, "{id}", exp.ID)
		if i, ok := byID[cm.EmDashCollection+"\x00"+id]; ok {
			return i, "deterministic", true
		}
	}
	locale := exp.Locale
	if locale == "" {
		locale = cm.DefaultLocale
	}
	indices := bySlug[key(cm.EmDashCollection, locale, exp.Slug)]
	if len(indices) == 1 {
		return indices[0], "slug", true
	}
	return 0, "", false
}

func collectionFor(kind string, m Map) (CollectionMap, bool) {
	for _, cm := range m.Collections {
		if cm.WordPressPostType == kind {
			return cm, true
		}
	}
	if kind == "post" {
		return CollectionMap{WordPressPostType: "post", EmDashCollection: "posts"}, true
	}
	if kind == "page" {
		return CollectionMap{WordPressPostType: "page", EmDashCollection: "pages"}, true
	}
	return CollectionMap{}, false
}

func compareEntry(r *Report, exp, act Entry) {
	checks := []struct{ code, field, a, b string }{
		{"slug_mismatch", "slug", exp.Slug, act.Slug}, {"status_mismatch", "status", canonicalStatus(exp.Status), canonicalStatus(act.Status)},
		{"title_mismatch", "title", normalizeText(exp.Title), normalizeText(act.Title)}, {"excerpt_mismatch", "excerpt", exp.ExcerptHash, act.ExcerptHash}, {"published_at_mismatch", "published_at", normalizeTime(exp.PublishedAt), normalizeTime(act.PublishedAt)},
		{"body_text_mismatch", "content", exp.Body.TextSHA256, act.Body.TextSHA256},
	}
	for _, c := range checks {
		if c.a != c.b {
			addIssue(r, issue(SeverityError, c.code, c.field+" differs", exp, act, c.field))
		}
	}
	if exp.Body.RawSHA256 != act.Body.RawSHA256 && exp.Body.TextSHA256 == act.Body.TextSHA256 {
		addIssue(r, issue(SeverityWarning, "body_raw_mismatch", "raw body differs but visible text matches", exp, act, "content"))
	}
	if !equalJSON(exp.Body.Headings, act.Body.Headings) {
		addIssue(r, issue(SeverityError, "heading_structure_mismatch", "heading structure differs", exp, act, "content"))
	}
	if !equalStrings(exp.Body.Links, act.Body.Links) {
		addIssue(r, issue(SeverityError, "links_mismatch", "link targets differ", exp, act, "content"))
	}
	if exp.Body.ImageCount != act.Body.ImageCount {
		addIssue(r, issue(SeverityError, "image_count_mismatch", "inline image counts differ", exp, act, "content"))
	}
	if len(exp.Body.Images) > 0 && len(act.Body.Images) > 0 && !equalStrings(exp.Body.Images, act.Body.Images) {
		addIssue(r, issue(SeverityError, "images_mismatch", "image references differ", exp, act, "content"))
	}
	if exp.Body.Lists != act.Body.Lists || exp.Body.Quotes != act.Body.Quotes || exp.Body.CodeBlocks != act.Body.CodeBlocks || exp.Body.Embeds != act.Body.Embeds {
		addIssue(r, issue(SeverityError, "body_structure_mismatch", "list, quote, code, or embed structure differs", exp, act, "content"))
	}
	if exp.Body.UnknownBlocks != act.Body.UnknownBlocks {
		addIssue(r, issue(SeverityWarning, "unknown_blocks_mismatch", "unknown block counts differ", exp, act, "content"))
	}
	if !equalJSON(exp.Taxonomies, act.Taxonomies) {
		addIssue(r, issue(SeverityError, "taxonomy_mismatch", "taxonomy relationships differ", exp, act, "taxonomies"))
	}
	if exp.Author != "" && act.Author != "" && exp.Author != act.Author {
		addIssue(r, issue(SeverityError, "author_mismatch", "author relationship differs", exp, act, "author"))
	}
	for field, hash := range exp.Fields {
		if act.Fields[field] != hash {
			addIssue(r, issue(SeverityError, "field_mismatch", "mapped field is missing or differs", exp, act, field))
		}
	}
	if exp.UpdatedAt != "" && act.UpdatedAt != "" && normalizeTime(exp.UpdatedAt) != normalizeTime(act.UpdatedAt) {
		addIssue(r, issue(SeverityWarning, "updated_at_mismatch", "updated timestamp differs", exp, act, "updated_at"))
	}
}

func issue(s Severity, code, message string, exp, act Entry, field string) Issue {
	return Issue{Severity: s, Code: code, Message: message, SourceID: exp.ID, TargetID: act.ID, Kind: exp.Kind, Slug: exp.Slug, Field: field}
}
func addIssue(r *Report, i Issue) {
	r.Issues = append(r.Issues, i)
	switch i.Severity {
	case SeverityCritical:
		r.Totals.Critical++
	case SeverityError:
		r.Totals.Errors++
	case SeverityWarning:
		r.Totals.Warnings++
	}
}
func key(kind, locale, slug string) string { return kind + "\x00" + locale + "\x00" + slug }
func canonicalStatus(s string) string {
	switch s {
	case "publish":
		return "published"
	case "pending", "private":
		return "draft"
	case "future":
		return "scheduled"
	case "trash":
		return "archived"
	}
	return s
}
func normalizeTime(s string) string {
	if len(s) >= 19 {
		return s[:19]
	}
	return s
}
func equalStrings(a, b []string) bool {
	aa, bb := append([]string(nil), a...), append([]string(nil), b...)
	sort.Strings(aa)
	sort.Strings(bb)
	return equalJSON(aa, bb)
}
func equalJSON(a, b any) bool {
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return string(x) == string(y)
}
func snapshotHash(s Snapshot) string {
	s.GeneratedAt = ""
	b, _ := json.Marshal(s)
	return HashString(string(b))
}

func ValidateMap(m Map) error {
	if m.Version != 0 && m.Version != 1 {
		return fmt.Errorf("unsupported content map version %d", m.Version)
	}
	return nil
}

func ApplyPolicy(report *Report, policy Policy) {
	if policy.Version == 0 {
		policy.Version = 1
	}
	report.Policy = policy
	report.Totals.Critical, report.Totals.Errors, report.Totals.Warnings = 0, 0, 0
	for i := range report.Issues {
		if severity, ok := policy.SeverityOverrides[report.Issues[i].Code]; ok {
			report.Issues[i].Severity = severity
		}
		switch report.Issues[i].Severity {
		case SeverityCritical:
			report.Totals.Critical++
		case SeverityError:
			report.Totals.Errors++
		case SeverityWarning:
			report.Totals.Warnings++
		}
	}
	report.OK = report.Totals.Critical <= policy.AllowCritical && report.Totals.Errors <= policy.AllowErrors
}
