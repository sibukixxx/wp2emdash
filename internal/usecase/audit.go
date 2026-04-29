package usecase

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/rokubunnoni-inc/wp2emdash/internal/domain/score"
	"github.com/rokubunnoni-inc/wp2emdash/internal/domain/source"
	"github.com/rokubunnoni-inc/wp2emdash/internal/infra/wpcli"
	"github.com/rokubunnoni-inc/wp2emdash/internal/usecase/reporting"
)

type AuditParams struct {
	WPRoot  string
	OutDir  string
	Write   bool
	Version string
}

type AuditResult struct {
	Bundle      reporting.Bundle
	SummaryPath string
	ReportPath  string
}

// RunAuditFromSource runs the audit pipeline using the given source adapter.
// Use this when you need to inject a custom Auditor (e.g. for testing, or for
// a non-WordPress CMS). RunAudit is the convenience wrapper for the WP case.
func RunAuditFromSource(ctx context.Context, src source.Auditor, params AuditParams) (AuditResult, error) {
	a, err := src.Run(ctx)
	if err != nil {
		return AuditResult{}, err
	}
	s := score.Compute(a)

	bundle := reporting.Bundle{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Tool:        "wp2emdash",
		Version:     params.Version,
		Audit:       a,
		Score:       s,
	}

	result := AuditResult{
		Bundle:      bundle,
		SummaryPath: filepath.Join(params.OutDir, "summary.json"),
		ReportPath:  filepath.Join(params.OutDir, "risk-report.md"),
	}
	if !params.Write {
		return result, nil
	}
	if err := reporting.WriteAll(params.OutDir, bundle); err != nil {
		return AuditResult{}, fmt.Errorf("write audit bundle: %w", err)
	}
	return result, nil
}

// RunAudit is the standard entry point for WordPress site auditing.
// It constructs a WordPress-backed Auditor from params.WPRoot and delegates
// to RunAuditFromSource.
func RunAudit(ctx context.Context, params AuditParams) (AuditResult, error) {
	auditor, err := wpcli.NewAuditor(params.WPRoot)
	if err != nil {
		return AuditResult{}, err
	}
	return RunAuditFromSource(ctx, auditor, params)
}
