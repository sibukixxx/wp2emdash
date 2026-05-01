package seo_test

import (
	"reflect"
	"testing"

	"github.com/sibukixxx/wp2emdash/internal/domain/seo"
)

func TestDiffURLMapsClassifiesEntriesIntoMatchedOnlyOldAndOnlyNew(t *testing.T) {
	old := seo.URLMap{
		Source: "wordpress",
		Entries: []seo.URLEntry{
			{URL: "https://example.com/about/"},
			{URL: "https://example.com/blog/2024/01/launch/"},
			{URL: "https://example.com/legacy/"},
		},
	}
	newMap := seo.URLMap{
		Source: "emdash",
		Entries: []seo.URLEntry{
			{URL: "https://example.com/about/"},
			{URL: "https://example.com/blog/2024/01/launch/"},
			{URL: "https://example.com/new-feature/"},
		},
	}

	diff := seo.DiffURLMaps(old, newMap)

	wantMatched := []string{
		"https://example.com/about/",
		"https://example.com/blog/2024/01/launch/",
	}
	wantOnlyInOld := []string{"https://example.com/legacy/"}
	wantOnlyInNew := []string{"https://example.com/new-feature/"}
	if !reflect.DeepEqual(diff.Matched, wantMatched) {
		t.Errorf("Matched mismatch: got %v want %v", diff.Matched, wantMatched)
	}
	if !reflect.DeepEqual(diff.OnlyInOld, wantOnlyInOld) {
		t.Errorf("OnlyInOld mismatch: got %v want %v", diff.OnlyInOld, wantOnlyInOld)
	}
	if !reflect.DeepEqual(diff.OnlyInNew, wantOnlyInNew) {
		t.Errorf("OnlyInNew mismatch: got %v want %v", diff.OnlyInNew, wantOnlyInNew)
	}

	wantTotals := seo.URLMapTotals{
		Old:       3,
		New:       3,
		Matched:   2,
		OnlyInOld: 1,
		OnlyInNew: 1,
	}
	if diff.Total != wantTotals {
		t.Errorf("Total mismatch: got %+v want %+v", diff.Total, wantTotals)
	}
	if diff.OldSource != "wordpress" || diff.NewSource != "emdash" {
		t.Errorf("source labels mismatch: old=%q new=%q", diff.OldSource, diff.NewSource)
	}
}

func TestDiffURLMapsNormalizesURLsForComparison(t *testing.T) {
	tests := []struct {
		name string
		old  []string
		new  []string
		want struct {
			matched   []string
			onlyInOld []string
			onlyInNew []string
		}
	}{
		{
			name: "trailing slash differences are matched",
			old:  []string{"https://example.com/about"},
			new:  []string{"https://example.com/about/"},
			want: struct {
				matched   []string
				onlyInOld []string
				onlyInNew []string
			}{
				matched:   []string{"https://example.com/about"},
				onlyInOld: []string{},
				onlyInNew: []string{},
			},
		},
		{
			name: "fragment is ignored",
			old:  []string{"https://example.com/page#top"},
			new:  []string{"https://example.com/page"},
			want: struct {
				matched   []string
				onlyInOld []string
				onlyInNew []string
			}{
				matched:   []string{"https://example.com/page#top"},
				onlyInOld: []string{},
				onlyInNew: []string{},
			},
		},
		{
			name: "scheme differences are matched (http vs https)",
			old:  []string{"http://example.com/a/"},
			new:  []string{"https://example.com/a/"},
			want: struct {
				matched   []string
				onlyInOld []string
				onlyInNew []string
			}{
				matched:   []string{"http://example.com/a/"},
				onlyInOld: []string{},
				onlyInNew: []string{},
			},
		},
		{
			name: "case sensitive in path",
			old:  []string{"https://example.com/About/"},
			new:  []string{"https://example.com/about/"},
			want: struct {
				matched   []string
				onlyInOld []string
				onlyInNew []string
			}{
				matched:   []string{},
				onlyInOld: []string{"https://example.com/About/"},
				onlyInNew: []string{"https://example.com/about/"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldMap := seo.URLMap{Entries: toEntries(tt.old)}
			newMap := seo.URLMap{Entries: toEntries(tt.new)}
			diff := seo.DiffURLMaps(oldMap, newMap)
			if !equalStrings(diff.Matched, tt.want.matched) {
				t.Errorf("Matched mismatch: got %v want %v", diff.Matched, tt.want.matched)
			}
			if !equalStrings(diff.OnlyInOld, tt.want.onlyInOld) {
				t.Errorf("OnlyInOld mismatch: got %v want %v", diff.OnlyInOld, tt.want.onlyInOld)
			}
			if !equalStrings(diff.OnlyInNew, tt.want.onlyInNew) {
				t.Errorf("OnlyInNew mismatch: got %v want %v", diff.OnlyInNew, tt.want.onlyInNew)
			}
		})
	}
}

func TestDiffURLMapsDeduplicatesWithinASide(t *testing.T) {
	old := seo.URLMap{Entries: toEntries([]string{
		"https://example.com/a/",
		"https://example.com/a/",
		"https://example.com/b/",
	})}
	newMap := seo.URLMap{Entries: toEntries([]string{
		"https://example.com/a/",
	})}
	diff := seo.DiffURLMaps(old, newMap)
	if diff.Total.Old != 2 {
		t.Errorf("Old totals should dedupe: got %d want 2", diff.Total.Old)
	}
	if !equalStrings(diff.Matched, []string{"https://example.com/a/"}) {
		t.Errorf("Matched should not contain duplicates: %v", diff.Matched)
	}
	if !equalStrings(diff.OnlyInOld, []string{"https://example.com/b/"}) {
		t.Errorf("OnlyInOld mismatch: %v", diff.OnlyInOld)
	}
}

func TestDiffURLMapsHandlesEmptyInputsWithoutNilSlices(t *testing.T) {
	diff := seo.DiffURLMaps(seo.URLMap{}, seo.URLMap{})
	if diff.Matched == nil || diff.OnlyInOld == nil || diff.OnlyInNew == nil {
		t.Errorf("empty input should produce empty (non-nil) slices to keep JSON shape stable, got %+v", diff)
	}
	if len(diff.Matched) != 0 || len(diff.OnlyInOld) != 0 || len(diff.OnlyInNew) != 0 {
		t.Errorf("expected all empty: %+v", diff)
	}
}

func toEntries(urls []string) []seo.URLEntry {
	out := make([]seo.URLEntry, len(urls))
	for i, u := range urls {
		out[i] = seo.URLEntry{URL: u}
	}
	return out
}

func equalStrings(a, b []string) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	return reflect.DeepEqual(a, b)
}
