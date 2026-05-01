package seo_test

import (
	"reflect"
	"testing"

	"github.com/sibukixxx/wp2emdash/internal/domain/seo"
)

func TestApplyPostMetaPreferYoastWhenAllPluginsSetTitle(t *testing.T) {
	item := seo.MetaItem{Title: "(core)"}
	raw := map[string]string{
		"_yoast_wpseo_title": "(yoast)",
		"rank_math_title":    "(rank math)",
		"_aioseop_title":     "(aioseo)",
	}
	seo.ApplyPostMeta(&item, raw)
	if item.Title != "(yoast)" {
		t.Errorf("Yoast should win for title, got %q", item.Title)
	}
}

func TestApplyPostMetaFallsThroughToNextPluginWhenHigherPrecedenceIsEmpty(t *testing.T) {
	item := seo.MetaItem{}
	raw := map[string]string{
		"_yoast_wpseo_title":    "",
		"rank_math_title":       "(rank math)",
		"_yoast_wpseo_metadesc": "",
		"rank_math_description": "rank desc",
	}
	seo.ApplyPostMeta(&item, raw)
	if item.Title != "(rank math)" {
		t.Errorf("expected fallthrough to rank math, got %q", item.Title)
	}
	if item.Description != "rank desc" {
		t.Errorf("expected description from rank math, got %q", item.Description)
	}
}

func TestApplyPostMetaSetsNoIndexWhenYoastFlagIsOne(t *testing.T) {
	tests := []struct {
		name string
		val  string
		want bool
	}{
		{"explicit noindex", "1", true},
		{"explicit follow (0)", "0", false},
		{"missing key", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := seo.MetaItem{}
			raw := map[string]string{}
			if tt.val != "" {
				raw["_yoast_wpseo_meta-robots-noindex"] = tt.val
			}
			seo.ApplyPostMeta(&item, raw)
			if item.NoIndex != tt.want {
				t.Errorf("NoIndex got %v want %v", item.NoIndex, tt.want)
			}
		})
	}
}

func TestApplyPostMetaTagsSourceCorrectly(t *testing.T) {
	tests := []struct {
		name string
		raw  map[string]string
		want string
	}{
		{
			name: "yoast only",
			raw:  map[string]string{"_yoast_wpseo_title": "x"},
			want: "yoast",
		},
		{
			name: "rank math only",
			raw:  map[string]string{"rank_math_title": "x"},
			want: "rank_math",
		},
		{
			name: "aioseo only",
			raw:  map[string]string{"_aioseop_title": "x"},
			want: "aioseo",
		},
		{
			name: "yoast + rank math => merged",
			raw: map[string]string{
				"_yoast_wpseo_title": "x",
				"rank_math_title":    "y",
			},
			want: "merged",
		},
		{
			name: "no SEO meta at all => empty source",
			raw:  map[string]string{},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := seo.MetaItem{}
			seo.ApplyPostMeta(&item, tt.raw)
			if item.Source != tt.want {
				t.Errorf("Source got %q want %q", item.Source, tt.want)
			}
		})
	}
}

func TestSEOPostMetaKeysReturnsKnownKeys(t *testing.T) {
	keys := seo.PostMetaKeys()
	wantSubset := []string{
		"_yoast_wpseo_title",
		"rank_math_description",
		"_aioseop_title",
	}
	for _, k := range wantSubset {
		if !contains(keys, k) {
			t.Errorf("PostMetaKeys missing %q; got %v", k, keys)
		}
	}
	// Returned slice should be a copy so mutations don't poison the package.
	keys[0] = "MUTATED"
	again := seo.PostMetaKeys()
	if reflect.DeepEqual(keys, again) {
		t.Errorf("PostMetaKeys should return a defensive copy")
	}
}

func contains(haystack []string, needle string) bool {
	for _, h := range haystack {
		if h == needle {
			return true
		}
	}
	return false
}
