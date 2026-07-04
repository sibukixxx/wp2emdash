package wpcli

import "testing"

func TestParseYoastIndexableLines(t *testing.T) {
	raw := `{"object_id": 1, "title": "Hello", "description": null, "canonical": null, "og_title": null, "og_image": null, "noindex": 1}
{"object_id": 2, "title": null, "description": "line1\\nline2", "canonical": "https://example.test/c/", "og_title": "OG", "og_image": null, "noindex": null}
`
	rows, bad := parseYoastIndexableLines(raw)
	if bad != 0 {
		t.Fatalf("bad lines: want 0, got %d", bad)
	}
	if len(rows) != 2 {
		t.Fatalf("rows: want 2, got %d", len(rows))
	}
	r1 := rows[1]
	if r1.Title != "Hello" || !r1.NoIndex {
		t.Errorf("row 1 mismatch: %+v", r1)
	}
	if r1.Description != "" {
		t.Errorf("row 1 description should be empty for NULL, got %q", r1.Description)
	}
	r2 := rows[2]
	// The mysql batch mode escapes the backslash of the JSON \n escape as
	// \\n; after decoding, the JSON parser restores the real newline.
	if r2.Description != "line1\nline2" {
		t.Errorf("row 2 description: want embedded newline restored, got %q", r2.Description)
	}
	if r2.Canonical != "https://example.test/c/" || r2.OGTitle != "OG" || r2.NoIndex {
		t.Errorf("row 2 mismatch: %+v", r2)
	}
}

func TestParseYoastIndexableLinesCountsInvalidLines(t *testing.T) {
	raw := "not-json\n{\"object_id\": 3, \"title\": \"T\"}\n{\"title\": \"missing id\"}\n"
	rows, bad := parseYoastIndexableLines(raw)
	if bad != 2 {
		t.Fatalf("bad lines: want 2, got %d", bad)
	}
	if len(rows) != 1 || rows[3].Title != "T" {
		t.Fatalf("rows mismatch: %+v", rows)
	}
}

func TestParseYoastIndexableLinesEmptyInput(t *testing.T) {
	rows, bad := parseYoastIndexableLines("")
	if bad != 0 || len(rows) != 0 {
		t.Fatalf("want empty result, got rows=%v bad=%d", rows, bad)
	}
}

func TestDecodeMySQLBatchEscapes(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"no escapes", `{"a":1}`, `{"a":1}`},
		{"escaped backslash", `a\\nb`, `a\nb`},
		{"batch newline", `a\nb`, "a\nb"},
		{"batch tab", `a\tb`, "a\tb"},
		{"unknown escape kept", `a\qb`, `a\qb`},
		{"trailing backslash kept", `ab\`, `ab\`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := decodeMySQLBatchEscapes(tt.in); got != tt.want {
				t.Errorf("got %q want %q", got, tt.want)
			}
		})
	}
}
