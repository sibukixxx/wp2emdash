// Package assessment defines the stable, evidence-first output of a read-only
// WordPress migration assessment.
package assessment

import (
	"strings"
	"time"

	"github.com/sibukixxx/wp2emdash/internal/domain/audit"
	"github.com/sibukixxx/wp2emdash/internal/domain/score"
	"github.com/sibukixxx/wp2emdash/internal/domain/source"
)

const SchemaVersion = "1.0"

type ObservationStatus string

const (
	StatusObserved    ObservationStatus = "observed"
	StatusNotDetected ObservationStatus = "not_detected"
	StatusUnavailable ObservationStatus = "unavailable"
	StatusUnknown     ObservationStatus = "unknown"
	StatusError       ObservationStatus = "error"
)

type Confidence string

const (
	ConfidenceConfirmed Confidence = "confirmed"
	ConfidenceLikely    Confidence = "likely"
	ConfidenceUnknown   Confidence = "unknown"
)

type Evidence struct {
	Type  string `json:"type"`
	Field string `json:"field,omitempty"`
	Value any    `json:"value"`
}

type Signal struct {
	Signal     string     `json:"signal"`
	Category   string     `json:"category"`
	Severity   string     `json:"severity"`
	Confidence Confidence `json:"confidence"`
	Evidence   []Evidence `json:"evidence"`
	Points     int        `json:"points"`
}

type Section[T any] struct {
	Status ObservationStatus `json:"status"`
	Facts  T                 `json:"facts"`
}

type Inventory struct {
	Site          Section[audit.SiteInfo]     `json:"runtime"`
	Content       Section[audit.ContentStats] `json:"content"`
	Media         Section[audit.UploadsStats] `json:"media"`
	Theme         Section[audit.ThemeStats]   `json:"theme"`
	Plugins       Section[audit.PluginsStats] `json:"plugins"`
	Customization Section[audit.CustomStats]  `json:"customization"`
}

type TechnicalScore struct {
	Total     int            `json:"total"`
	Breakdown map[string]int `json:"breakdown"`
}

type Timing struct {
	StartedAt  string           `json:"started_at"`
	FinishedAt string           `json:"finished_at"`
	DurationMS int64            `json:"duration_ms"`
	Phases     map[string]int64 `json:"phases_ms,omitempty"`
}

type Summary struct {
	SchemaVersion string           `json:"schema_version"`
	Tool          string           `json:"tool"`
	ToolVersion   string           `json:"tool_version"`
	Scope         string           `json:"assessment_scope"`
	ReadOnly      bool             `json:"read_only"`
	Inventory     Inventory        `json:"inventory"`
	Signals       []Signal         `json:"signals"`
	Score         TechnicalScore   `json:"technical_score"`
	Warnings      []source.Warning `json:"warnings,omitempty"`
	Timing        Timing           `json:"timing"`
}

func Build(a audit.Audit, warnings []source.Warning, scored score.Result, version string, started, finished time.Time) Summary {
	statuses := map[string]ObservationStatus{}
	for _, section := range []string{"site", "content", "uploads", "theme", "plugins", "customization"} {
		statuses[section] = StatusObserved
	}
	for _, warning := range warnings {
		prefix := strings.SplitN(warning.Code, ".", 2)[0]
		if _, ok := statuses[prefix]; ok {
			statuses[prefix] = StatusUnavailable
		}
	}
	inv := Inventory{
		Site:          Section[audit.SiteInfo]{Status: statuses["site"], Facts: a.Site},
		Content:       Section[audit.ContentStats]{Status: statuses["content"], Facts: a.Content},
		Media:         Section[audit.UploadsStats]{Status: statuses["uploads"], Facts: a.Uploads},
		Theme:         Section[audit.ThemeStats]{Status: statuses["theme"], Facts: a.Theme},
		Plugins:       Section[audit.PluginsStats]{Status: statuses["plugins"], Facts: a.Plugins},
		Customization: Section[audit.CustomStats]{Status: statuses["customization"], Facts: a.Customization},
	}
	signals := signalsFrom(a, scored)
	breakdown := map[string]int{"content": 0, "media": 0, "plugins": 0, "customization": 0, "dynamic_features": 0, "seo": 0}
	for _, signal := range signals {
		breakdown[signal.Category] += signal.Points
	}
	return Summary{
		SchemaVersion: SchemaVersion, Tool: "wp2emdash", ToolVersion: version,
		Scope: "Full Assessment", ReadOnly: true, Inventory: inv, Signals: signals,
		Score: TechnicalScore{Total: scored.Score, Breakdown: breakdown}, Warnings: warnings,
		Timing: Timing{StartedAt: started.UTC().Format(time.RFC3339Nano), FinishedAt: finished.UTC().Format(time.RFC3339Nano), DurationMS: finished.Sub(started).Milliseconds(), Phases: map[string]int64{"audit_ms": finished.Sub(started).Milliseconds()}},
	}
}

func signalsFrom(a audit.Audit, scored score.Result) []Signal {
	result := make([]Signal, 0, len(scored.Reasons)+3)
	category := func(code string) string {
		switch {
		case strings.HasPrefix(code, "posts."), strings.HasPrefix(code, "pages."):
			return "content"
		case strings.HasPrefix(code, "plugin.seo"), strings.HasPrefix(code, "seo."):
			return "seo"
		case strings.HasPrefix(code, "plugin."):
			return "plugins"
		case strings.Contains(code, "external"), strings.Contains(code, "jquery"), strings.Contains(code, "shortcode"), strings.Contains(code, "member"), strings.Contains(code, "woo"), strings.Contains(code, "multilingual"):
			return "dynamic_features"
		default:
			return "customization"
		}
	}
	for _, reason := range scored.Reasons {
		result = append(result, Signal{Signal: reason.Code, Category: category(reason.Code), Severity: severity(reason.Points), Confidence: ConfidenceConfirmed, Evidence: []Evidence{{Type: "observed_fact", Field: reason.Code, Value: reason.Text}}, Points: reason.Points})
	}
	add := func(name, category, field string, value any, when bool) {
		if when {
			result = append(result, Signal{Signal: name, Category: category, Severity: "info", Confidence: ConfidenceConfirmed, Evidence: []Evidence{{Type: "observed_fact", Field: field, Value: value}}})
		}
	}
	add("forms_detected", "dynamic_features", "plugins.has_form", true, a.Plugins.HasForm)
	add("shortcode_dependency_detected", "dynamic_features", "customization.shortcode_post_count", a.Customization.ShortcodePostCount, a.Customization.ShortcodePostCount > 0)
	add("oversized_content_detected", "content", "customization.oversized_content_count", a.Customization.OversizedContentCount, a.Customization.OversizedContentCount > 0)
	return result
}

func severity(points int) string {
	if points >= 25 {
		return "high"
	}
	if points >= 10 {
		return "medium"
	}
	return "low"
}
