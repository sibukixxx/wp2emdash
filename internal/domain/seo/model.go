// Package seo defines value types and pure functions for SEO migration data:
// per-post metadata extraction, redirect rules, and URL map comparison.
//
// Everything here is side-effect free and depends only on the standard
// library. Source adapters in infra/* shell out to wp-cli or read files and
// produce these types; usecase/seo.go orchestrates the pipeline.
package seo

// MetaItem is the SEO metadata extracted for a single post / page / CPT entry.
//
// Fields use omitempty so missing values stay out of JSON output and so the
// shape is stable across SEO plugins (Yoast / Rank Math / AIOSEO / core).
type MetaItem struct {
	PostID      int    `json:"post_id"`
	PostType    string `json:"post_type"`
	URL         string `json:"url"`
	Slug        string `json:"slug,omitempty"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Canonical   string `json:"canonical,omitempty"`
	OGTitle     string `json:"og_title,omitempty"`
	OGImage     string `json:"og_image,omitempty"`
	NoIndex     bool   `json:"noindex,omitempty"`
	Source      string `json:"source,omitempty"` // "yoast" | "rank_math" | "aioseo" | "core" | "merged"
}

// MetaSet is the top-level document for `wp2emdash seo extract-meta`.
type MetaSet struct {
	GeneratedAt string     `json:"generated_at"`
	Tool        string     `json:"tool"`
	Version     string     `json:"version"`
	SiteURL     string     `json:"site_url,omitempty"`
	Items       []MetaItem `json:"items"`
}

// RedirectRule represents a single redirect from any source (htaccess or DB).
type RedirectRule struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Code   int    `json:"code,omitempty"`  // 301, 302, 307, 308. 0 = unspecified.
	Match  string `json:"match,omitempty"` // "exact" | "regex"
	Source string `json:"source"`          // "htaccess" | "redirection" | "safe-redirect-manager"
	Note   string `json:"note,omitempty"`  // free-form provenance (e.g. line number)
}

// RedirectSet is the top-level document for `wp2emdash seo extract-redirects`.
type RedirectSet struct {
	GeneratedAt string         `json:"generated_at"`
	Tool        string         `json:"tool"`
	Version     string         `json:"version"`
	Rules       []RedirectRule `json:"rules"`
}

// URLEntry is a single URL plus optional metadata in a URL map.
type URLEntry struct {
	URL    string `json:"url"`
	Title  string `json:"title,omitempty"`
	Status int    `json:"status,omitempty"`
}

// URLMap is the input format for `wp2emdash seo url-map`.
//
// The Source field is free-form ("wordpress", "emdash", a sitemap URL, ...)
// and gets echoed back into the diff output for provenance.
type URLMap struct {
	Source  string     `json:"source,omitempty"`
	Entries []URLEntry `json:"entries"`
}

// URLMapTotals carries the aggregate counts for a diff. Kept as a separate
// struct so the JSON document has a clean "total" key.
type URLMapTotals struct {
	Old       int `json:"old"`
	New       int `json:"new"`
	Matched   int `json:"matched"`
	OnlyInOld int `json:"only_in_old"`
	OnlyInNew int `json:"only_in_new"`
}

// URLMapDiff is the output document for `wp2emdash seo url-map`.
//
// OnlyInOld is the most actionable list: those URLs disappeared from the new
// site and likely need explicit redirects to avoid 404s.
type URLMapDiff struct {
	GeneratedAt string       `json:"generated_at"`
	Tool        string       `json:"tool"`
	Version     string       `json:"version"`
	OldSource   string       `json:"old_source,omitempty"`
	NewSource   string       `json:"new_source,omitempty"`
	Matched     []string     `json:"matched"`
	OnlyInOld   []string     `json:"only_in_old"`
	OnlyInNew   []string     `json:"only_in_new"`
	Total       URLMapTotals `json:"total"`
}
