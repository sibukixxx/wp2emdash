package wpcli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sibukixxx/wp2emdash/internal/domain/seo"
)

// collectYoastIndexables returns object_id -> indexable row for posts that
// carry at least one editor override in the {prefix}yoast_indexable table
// (Yoast 14+). A missing table (no Yoast, or Yoast < 14) is normal and
// returns an empty map without emitting a warning.
//
// Rows are fetched as one JSON_OBJECT per line instead of raw TSV because
// title / description are free text that may embed tabs and newlines.
func (a *Auditor) collectYoastIndexables(ctx context.Context, prefix string) map[int]seo.YoastIndexable {
	table := prefix + "yoast_indexable"
	exists := a.wpDBQueryInt(ctx, "seo.yoast_indexable.exists", fmt.Sprintf(
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = '%s'", table))
	if exists == 0 {
		return map[int]seo.YoastIndexable{}
	}

	// Only rows where an editor actually overrode something: all-NULL rows
	// would just restate the defaults the target CMS computes anyway.
	query := fmt.Sprintf(
		"SELECT JSON_OBJECT("+
			"'object_id', object_id, 'title', title, 'description', description, "+
			"'canonical', canonical, 'og_title', open_graph_title, 'og_image', open_graph_image, "+
			"'noindex', is_robots_noindex) "+
			"FROM %s WHERE object_type = 'post' AND ("+
			"title IS NOT NULL OR description IS NOT NULL OR canonical IS NOT NULL OR "+
			"open_graph_title IS NOT NULL OR open_graph_image IS NOT NULL OR is_robots_noindex = 1)",
		table)
	raw := a.wp(ctx, "seo.yoast_indexable", "db", "query", query, "--skip-column-names")
	rows, bad := parseYoastIndexableLines(raw)
	if bad > 0 {
		a.warnf("seo.yoast_indexable.invalid", "%d yoast_indexable rows returned invalid JSON and were skipped", bad)
	}
	return rows
}

// yoastIndexableRow mirrors the JSON_OBJECT emitted by collectYoastIndexables.
// Nullable columns come through as JSON null, hence the pointers.
type yoastIndexableRow struct {
	ObjectID    int     `json:"object_id"`
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Canonical   *string `json:"canonical"`
	OGTitle     *string `json:"og_title"`
	OGImage     *string `json:"og_image"`
	NoIndex     *int    `json:"noindex"`
}

// parseYoastIndexableLines decodes one JSON object per line into domain rows
// and reports how many lines failed to parse.
func parseYoastIndexableLines(raw string) (map[int]seo.YoastIndexable, int) {
	out := make(map[int]seo.YoastIndexable)
	bad := 0
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(decodeMySQLBatchEscapes(strings.TrimRight(line, "\r")))
		if line == "" {
			continue
		}
		var row yoastIndexableRow
		if err := json.Unmarshal([]byte(line), &row); err != nil || row.ObjectID == 0 {
			bad++
			continue
		}
		out[row.ObjectID] = seo.YoastIndexable{
			ObjectID:    row.ObjectID,
			Title:       deref(row.Title),
			Description: deref(row.Description),
			Canonical:   deref(row.Canonical),
			OGTitle:     deref(row.OGTitle),
			OGImage:     deref(row.OGImage),
			NoIndex:     row.NoIndex != nil && *row.NoIndex == 1,
		}
	}
	return out, bad
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// decodeMySQLBatchEscapes reverses the escaping the mysql client applies in
// batch (non-interactive) mode: NUL, tab, newline, and backslash inside
// column values are emitted as \0, \t, \n, and \\. JSON_OBJECT output never
// contains raw control characters, so in practice only \\ occurs, but the
// full set is decoded for safety.
func decodeMySQLBatchEscapes(s string) string {
	if !strings.ContainsRune(s, '\\') {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c != '\\' || i+1 == len(s) {
			b.WriteByte(c)
			continue
		}
		i++
		switch s[i] {
		case '\\':
			b.WriteByte('\\')
		case 'n':
			b.WriteByte('\n')
		case 't':
			b.WriteByte('\t')
		case '0':
			b.WriteByte(0)
		default:
			// Not a batch escape; keep both bytes untouched.
			b.WriteByte('\\')
			b.WriteByte(s[i])
		}
	}
	return b.String()
}
