package usecase

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/rokubunnoni-inc/wp2emdash/internal/domain/score"
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

func RunAudit(ctx context.Context, params AuditParams) (AuditResult, error) {
	auditor, err := wpcli.NewAuditor(params.WPRoot)
	if err != nil {
		return AuditResult{}, err
	}

	a, err := auditor.Run(ctx)
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
