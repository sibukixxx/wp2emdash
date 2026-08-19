// Package contentverify defines privacy-preserving content snapshots and the
// pure comparison rules used to prove a WordPress import arrived intact.
package contentverify

type Snapshot struct {
	SchemaVersion int      `json:"schema_version"`
	GeneratedAt   string   `json:"generated_at"`
	Tool          string   `json:"tool"`
	Version       string   `json:"version"`
	Source        string   `json:"source"`
	SiteURL       string   `json:"site_url,omitempty"`
	Complete      bool     `json:"complete"`
	Warnings      []string `json:"warnings,omitempty"`
	Entries       []Entry  `json:"entries"`
}

type Entry struct {
	ID          string            `json:"id"`
	Kind        string            `json:"kind"`
	Slug        string            `json:"slug"`
	Locale      string            `json:"locale,omitempty"`
	Status      string            `json:"status"`
	Title       string            `json:"title,omitempty"`
	ExcerptHash string            `json:"excerpt_sha256,omitempty"`
	Author      string            `json:"author,omitempty"`
	ParentID    string            `json:"parent_id,omitempty"`
	CreatedAt   string            `json:"created_at,omitempty"`
	UpdatedAt   string            `json:"updated_at,omitempty"`
	PublishedAt string            `json:"published_at,omitempty"`
	Body        Fingerprint       `json:"body"`
	Taxonomies  []Relationship    `json:"taxonomies,omitempty"`
	Fields      map[string]string `json:"field_hashes,omitempty"`
}

type Relationship struct {
	Taxonomy string `json:"taxonomy"`
	Slug     string `json:"slug"`
}

type Heading struct {
	Level int    `json:"level"`
	Hash  string `json:"sha256"`
}

type Fingerprint struct {
	Format        string    `json:"format"`
	RawSHA256     string    `json:"raw_sha256"`
	TextSHA256    string    `json:"text_sha256"`
	TextLength    int       `json:"text_length"`
	Headings      []Heading `json:"headings,omitempty"`
	Links         []string  `json:"links,omitempty"`
	Images        []string  `json:"images,omitempty"`
	ImageCount    int       `json:"image_count,omitempty"`
	Lists         int       `json:"lists,omitempty"`
	Quotes        int       `json:"quotes,omitempty"`
	CodeBlocks    int       `json:"code_blocks,omitempty"`
	Embeds        int       `json:"embeds,omitempty"`
	UnknownBlocks int       `json:"unknown_blocks,omitempty"`
}

type Map struct {
	Version     int               `json:"version"`
	Collections []CollectionMap   `json:"collections,omitempty"`
	Entries     []ExplicitMapping `json:"entries,omitempty"`
}

type CollectionMap struct {
	WordPressPostType       string            `json:"wordpress_post_type"`
	EmDashCollection        string            `json:"emdash_collection"`
	DefaultLocale           string            `json:"default_locale,omitempty"`
	TargetSourceIDField     string            `json:"target_source_id_field,omitempty"`
	DeterministicIDTemplate string            `json:"deterministic_id_template,omitempty"`
	Fields                  map[string]string `json:"fields,omitempty"`
	Meta                    map[string]string `json:"meta,omitempty"`
}

type ExplicitMapping struct {
	WordPressID      int    `json:"wordpress_id"`
	EmDashCollection string `json:"emdash_collection"`
	EmDashID         string `json:"emdash_id"`
	Locale           string `json:"locale,omitempty"`
}

type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityError    Severity = "error"
	SeverityWarning  Severity = "warning"
)

type Issue struct {
	Severity Severity `json:"severity"`
	Code     string   `json:"code"`
	Message  string   `json:"message"`
	SourceID string   `json:"source_id,omitempty"`
	TargetID string   `json:"target_id,omitempty"`
	Kind     string   `json:"kind,omitempty"`
	Slug     string   `json:"slug,omitempty"`
	Field    string   `json:"field,omitempty"`
}

type Match struct {
	SourceID string `json:"source_id"`
	TargetID string `json:"target_id"`
	Kind     string `json:"kind"`
	Locale   string `json:"locale,omitempty"`
	Method   string `json:"method"`
}

type Totals struct {
	Expected int `json:"expected"`
	Actual   int `json:"actual"`
	Matched  int `json:"matched"`
	Missing  int `json:"missing"`
	Extra    int `json:"extra"`
	Critical int `json:"critical"`
	Errors   int `json:"errors"`
	Warnings int `json:"warnings"`
}

type Report struct {
	SchemaVersion int     `json:"schema_version"`
	GeneratedAt   string  `json:"generated_at"`
	Tool          string  `json:"tool"`
	Version       string  `json:"version"`
	OK            bool    `json:"ok"`
	ExpectedHash  string  `json:"expected_snapshot_sha256"`
	ActualHash    string  `json:"actual_snapshot_sha256"`
	Totals        Totals  `json:"totals"`
	Matches       []Match `json:"matches"`
	Issues        []Issue `json:"issues,omitempty"`
	Policy        Policy  `json:"policy"`
}

type Policy struct {
	Version           int                 `json:"version"`
	AllowCritical     int                 `json:"allow_critical"`
	AllowErrors       int                 `json:"allow_errors"`
	SeverityOverrides map[string]Severity `json:"severity_overrides,omitempty"`
}
