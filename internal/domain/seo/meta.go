package seo

import "strings"

// postMetaKeys lists the postmeta keys recognised by ApplyPostMeta.
//
// Order matters: when several SEO plugins are active and set the same logical
// field, the value from the first matching key wins (Yoast > Rank Math >
// AIOSEO). This mirrors how WordPress itself tie-breaks based on plugin load
// order in most multi-SEO-plugin installs.
var postMetaKeys = []string{
	"_yoast_wpseo_title", "_yoast_wpseo_metadesc", "_yoast_wpseo_canonical",
	"_yoast_wpseo_meta-robots-noindex",
	"_yoast_wpseo_opengraph-title", "_yoast_wpseo_opengraph-image",
	"rank_math_title", "rank_math_description", "rank_math_canonical_url",
	"rank_math_facebook_title", "rank_math_facebook_image",
	"_aioseop_title", "_aioseop_description", "_aioseop_custom_link",
}

// PostMetaKeys returns a copy of the recognised postmeta key list.
// Callers (typically infra layers building SQL IN clauses) get a defensive
// copy so they cannot mutate the package's source-of-truth.
func PostMetaKeys() []string {
	out := make([]string, len(postMetaKeys))
	copy(out, postMetaKeys)
	return out
}

// ApplyPostMeta merges raw postmeta key/value pairs into item, applying the
// Yoast > Rank Math > AIOSEO precedence. Empty strings count as missing so a
// lower-priority plugin can still fill in a field that the higher-priority
// plugin left blank.
//
// item.Title is overwritten only when at least one SEO plugin sets a non-empty
// title (so a pre-populated WP core title survives if no SEO plugin supplies
// one).
//
// Source is set to "yoast" / "rank_math" / "aioseo" / "merged" depending on
// which plugins contributed; if no SEO plugin set anything, Source is left
// untouched (so the caller can mark it as "core" or empty).
func ApplyPostMeta(item *MetaItem, raw map[string]string) {
	if item == nil || len(raw) == 0 {
		return
	}
	pick := func(keys ...string) string {
		for _, k := range keys {
			if v := strings.TrimSpace(raw[k]); v != "" {
				return v
			}
		}
		return ""
	}
	if v := pick("_yoast_wpseo_title", "rank_math_title", "_aioseop_title"); v != "" {
		item.Title = v
	}
	if v := pick("_yoast_wpseo_metadesc", "rank_math_description", "_aioseop_description"); v != "" {
		item.Description = v
	}
	if v := pick("_yoast_wpseo_canonical", "rank_math_canonical_url", "_aioseop_custom_link"); v != "" {
		item.Canonical = v
	}
	if v := pick("_yoast_wpseo_opengraph-title", "rank_math_facebook_title"); v != "" {
		item.OGTitle = v
	}
	if v := pick("_yoast_wpseo_opengraph-image", "rank_math_facebook_image"); v != "" {
		item.OGImage = v
	}
	if raw["_yoast_wpseo_meta-robots-noindex"] == "1" {
		item.NoIndex = true
	}

	yoast := hasMeaningfulPrefix(raw, "_yoast_wpseo_")
	rank := hasMeaningfulPrefix(raw, "rank_math_")
	aio := hasMeaningfulPrefix(raw, "_aioseop_")
	switch {
	case yoast && rank:
		item.Source = "merged"
	case yoast:
		item.Source = "yoast"
	case rank:
		item.Source = "rank_math"
	case aio:
		item.Source = "aioseo"
	}
}

func hasMeaningfulPrefix(raw map[string]string, prefix string) bool {
	for k, v := range raw {
		if strings.HasPrefix(k, prefix) && strings.TrimSpace(v) != "" {
			return true
		}
	}
	return false
}
