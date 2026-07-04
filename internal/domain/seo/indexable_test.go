package seo_test

import (
	"testing"

	"github.com/sibukixxx/wp2emdash/internal/domain/seo"
)

func TestApplyYoastIndexableFillsFieldsPostMetaLeftEmpty(t *testing.T) {
	item := seo.MetaItem{Title: "(core)", Source: "core"}
	seo.ApplyYoastIndexable(&item, seo.YoastIndexable{
		ObjectID:    1,
		Title:       "Indexable Title",
		Description: "Indexable description",
		Canonical:   "https://example.test/canonical/",
		OGTitle:     "Indexable OG",
		NoIndex:     true,
	})
	if item.Title != "Indexable Title" {
		t.Errorf("Title: indexable should override the core title, got %q", item.Title)
	}
	if item.Description != "Indexable description" {
		t.Errorf("Description got %q", item.Description)
	}
	if item.Canonical != "https://example.test/canonical/" {
		t.Errorf("Canonical got %q", item.Canonical)
	}
	if item.OGTitle != "Indexable OG" {
		t.Errorf("OGTitle got %q", item.OGTitle)
	}
	if !item.NoIndex {
		t.Error("NoIndex should be set from the indexable row")
	}
	if item.Source != "yoast_indexable" {
		t.Errorf("Source: want yoast_indexable, got %q", item.Source)
	}
}

func TestApplyYoastIndexableDoesNotOverridePostMetaValues(t *testing.T) {
	item := seo.MetaItem{
		Title:       "Postmeta Title",
		Description: "Postmeta description",
		Source:      "yoast",
	}
	seo.ApplyYoastIndexable(&item, seo.YoastIndexable{
		ObjectID:    1,
		Title:       "Indexable Title",
		Description: "Indexable description",
		OGImage:     "https://example.test/og.png",
	})
	if item.Title != "Postmeta Title" {
		t.Errorf("Title: postmeta must win over indexable, got %q", item.Title)
	}
	if item.Description != "Postmeta description" {
		t.Errorf("Description: postmeta must win over indexable, got %q", item.Description)
	}
	if item.OGImage != "https://example.test/og.png" {
		t.Errorf("OGImage: indexable should backfill missing fields, got %q", item.OGImage)
	}
	if item.Source != "yoast" {
		t.Errorf("Source: existing plugin source must be kept, got %q", item.Source)
	}
}

func TestApplyYoastIndexableEmptyRowIsNoOp(t *testing.T) {
	item := seo.MetaItem{Title: "(core)", Source: "core"}
	seo.ApplyYoastIndexable(&item, seo.YoastIndexable{ObjectID: 1})
	if item.Source != "core" {
		t.Errorf("Source must stay untouched when nothing contributed, got %q", item.Source)
	}
	if item.Title != "(core)" {
		t.Errorf("Title must stay untouched, got %q", item.Title)
	}
}
