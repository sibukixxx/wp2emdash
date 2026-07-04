package seo

import "strings"

// YoastIndexable is one row of the {prefix}yoast_indexable table — the
// normalized store Yoast SEO uses since v14 (2020). Modern Yoast installs
// often leave the legacy _yoast_wpseo_* postmeta stale or empty, so a
// postmeta-only extraction silently drops editor overrides. Note the
// similarly named {prefix}yoast_seo_meta table holds only internal link
// counts and is NOT a metadata source.
type YoastIndexable struct {
	ObjectID    int
	Title       string
	Description string
	Canonical   string
	OGTitle     string
	OGImage     string
	NoIndex     bool
}

// ApplyYoastIndexable merges an indexable row into item. Explicit postmeta
// values stay the highest-precedence source: the indexable row only fills
// fields that are still empty after ApplyPostMeta. The one exception is
// Title — a WP core title placed by the caller (Source empty or "core") is
// overridden, because the indexable row records an actual editor override.
//
// When the row contributed at least one field and no SEO plugin postmeta was
// found, Source becomes "yoast_indexable"; an existing plugin Source is kept
// as-is since indexable data is Yoast-family anyway.
func ApplyYoastIndexable(item *MetaItem, row YoastIndexable) {
	if item == nil {
		return
	}
	contributed := false
	fill := func(dst *string, v string) {
		v = strings.TrimSpace(v)
		if *dst == "" && v != "" {
			*dst = v
			contributed = true
		}
	}

	if t := strings.TrimSpace(row.Title); t != "" && !sourceIsSEOPlugin(item.Source) && item.Title != t {
		item.Title = t
		contributed = true
	}
	fill(&item.Description, row.Description)
	fill(&item.Canonical, row.Canonical)
	fill(&item.OGTitle, row.OGTitle)
	fill(&item.OGImage, row.OGImage)
	if row.NoIndex && !item.NoIndex {
		item.NoIndex = true
		contributed = true
	}

	if contributed && !sourceIsSEOPlugin(item.Source) {
		item.Source = "yoast_indexable"
	}
}

// sourceIsSEOPlugin reports whether s already names an SEO plugin source set
// by ApplyPostMeta (as opposed to "", "core", or "yoast_indexable").
func sourceIsSEOPlugin(s string) bool {
	switch s {
	case "yoast", "rank_math", "aioseo", "merged":
		return true
	}
	return false
}
