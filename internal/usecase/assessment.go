package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sibukixxx/wp2emdash/internal/domain/assessment"
)

type AssessmentResult struct {
	Summary assessment.Summary
	Paths   []string
}

// RunAssessment aggregates existing read-only probes. It never invokes a
// migration, sync, deploy, database write, or other destructive operation.
func RunAssessment(ctx context.Context, ctxParams AuditParams) (AssessmentResult, error) {
	started := time.Now()
	ctxParams.Write = false
	auditResult, err := RunAudit(ctx, ctxParams)
	if err != nil {
		return AssessmentResult{}, fmt.Errorf("assessment audit: %w", err)
	}
	finished := time.Now()
	summary := assessment.Build(auditResult.Bundle.Audit, auditResult.Bundle.Warnings, auditResult.Bundle.Score, ctxParams.Version, started, finished)
	result := AssessmentResult{Summary: summary}
	if ctxParams.OutDir == "" {
		return result, nil
	}
	if err := os.MkdirAll(ctxParams.OutDir, 0o755); err != nil {
		return AssessmentResult{}, fmt.Errorf("create assessment directory: %w", err)
	}
	artifacts := []struct {
		name  string
		value any
	}{
		{"summary.json", summary},
		{"inventory.json", struct {
			SchemaVersion string               `json:"schema_version"`
			Inventory     assessment.Inventory `json:"inventory"`
		}{assessment.SchemaVersion, summary.Inventory}},
		{"signals.json", struct {
			SchemaVersion string                    `json:"schema_version"`
			Signals       []assessment.Signal       `json:"signals"`
			Score         assessment.TechnicalScore `json:"technical_score"`
		}{assessment.SchemaVersion, summary.Signals, summary.Score}},
	}
	for _, artifact := range artifacts {
		path := filepath.Join(ctxParams.OutDir, artifact.name)
		if err := writeAssessmentJSON(path, artifact.value); err != nil {
			return AssessmentResult{}, err
		}
		result.Paths = append(result.Paths, path)
	}
	reportPath := filepath.Join(ctxParams.OutDir, "technical-report.md")
	if err := os.WriteFile(reportPath, []byte(renderTechnicalReport(summary)), 0o644); err != nil {
		return AssessmentResult{}, fmt.Errorf("write %s: %w", reportPath, err)
	}
	result.Paths = append(result.Paths, reportPath)
	return result, nil
}

func writeAssessmentJSON(path string, value any) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(value); err != nil {
		return fmt.Errorf("encode %s: %w", path, err)
	}
	return nil
}

func renderTechnicalReport(s assessment.Summary) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# WordPress Migration Technical Assessment\n\n")
	fmt.Fprintf(&b, "- Schema: %s\n- Scope: %s\n- Read-only: %t\n- Duration: %d ms\n- Technical score: %d\n\n", s.SchemaVersion, s.Scope, s.ReadOnly, s.Timing.DurationMS, s.Score.Total)
	fmt.Fprintf(&b, "## Observation status\n\n- Runtime: %s\n- Content: %s\n- Media: %s\n- Theme: %s\n- Plugins: %s\n- Customization: %s\n\n", s.Inventory.Site.Status, s.Inventory.Content.Status, s.Inventory.Media.Status, s.Inventory.Theme.Status, s.Inventory.Plugins.Status, s.Inventory.Customization.Status)
	fmt.Fprintf(&b, "## Technical signals\n\n")
	if len(s.Signals) == 0 {
		fmt.Fprintf(&b, "No scored signals were detected from the available evidence.\n")
	}
	for _, signal := range s.Signals {
		fmt.Fprintf(&b, "- `%s` (%s, %s): %d point(s)\n", signal.Signal, signal.Severity, signal.Confidence, signal.Points)
	}
	fmt.Fprintf(&b, "\n## Scope and non-goals\n\nThis report records technical facts, evidence, signals, and a deterministic reference score. It does not recommend a CMS, estimate work, set pricing, generate a proposal, guarantee SEO rankings, or make a business decision.\n")
	return b.String()
}
