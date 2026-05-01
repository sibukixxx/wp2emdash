package seo

import (
	"strings"
)

// DiffURLMaps compares two URL maps and returns a structured diff.
//
// URLs are matched after a small normalization pass (fragment removed,
// scheme http/https collapsed, trailing slash collapsed) so cosmetic
// differences between WP and EmDash output do not show up as missing pages.
// Path case is preserved: "/About/" and "/about/" are reported as different.
//
// Within each side the entries are deduplicated. Output ordering follows the
// input order of `old` for Matched and OnlyInOld; OnlyInNew follows `newMap`.
// Output slices are never nil so JSON callers see [] instead of null.
func DiffURLMaps(old, newMap URLMap) URLMapDiff {
	oldKeys, oldOriginal := normalizedSet(old.Entries)
	newKeys, newOriginal := normalizedSet(newMap.Entries)

	matched := make([]string, 0)
	onlyInOld := make([]string, 0)
	onlyInNew := make([]string, 0)

	for _, key := range oldKeys {
		if _, ok := newOriginal[key]; ok {
			matched = append(matched, oldOriginal[key])
		} else {
			onlyInOld = append(onlyInOld, oldOriginal[key])
		}
	}
	for _, key := range newKeys {
		if _, ok := oldOriginal[key]; !ok {
			onlyInNew = append(onlyInNew, newOriginal[key])
		}
	}

	return URLMapDiff{
		OldSource: old.Source,
		NewSource: newMap.Source,
		Matched:   matched,
		OnlyInOld: onlyInOld,
		OnlyInNew: onlyInNew,
		Total: URLMapTotals{
			Old:       len(oldKeys),
			New:       len(newKeys),
			Matched:   len(matched),
			OnlyInOld: len(onlyInOld),
			OnlyInNew: len(onlyInNew),
		},
	}
}

// normalizedSet returns the unique normalized keys in input order plus a
// map from key -> first-seen original URL (so the diff preserves the user's
// preferred spelling on each side).
func normalizedSet(entries []URLEntry) ([]string, map[string]string) {
	keys := make([]string, 0, len(entries))
	original := make(map[string]string, len(entries))
	for _, e := range entries {
		key := normalizeURL(e.URL)
		if key == "" {
			continue
		}
		if _, seen := original[key]; seen {
			continue
		}
		original[key] = e.URL
		keys = append(keys, key)
	}
	return keys, original
}

// normalizeURL collapses cosmetic differences but preserves path case.
// Empty input returns "" (filtered out by callers).
func normalizeURL(raw string) string {
	u := strings.TrimSpace(raw)
	if u == "" {
		return ""
	}
	if i := strings.Index(u, "#"); i >= 0 {
		u = u[:i]
	}
	switch {
	case strings.HasPrefix(u, "https://"):
		u = "//" + strings.TrimPrefix(u, "https://")
	case strings.HasPrefix(u, "http://"):
		u = "//" + strings.TrimPrefix(u, "http://")
	}
	if len(u) > 1 && strings.HasSuffix(u, "/") {
		u = strings.TrimRight(u, "/")
	}
	return u
}
