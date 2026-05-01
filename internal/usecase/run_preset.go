package usecase

import (
	"context"
	"path"
	"time"

	"github.com/sibukixxx/wp2emdash/internal/domain/preset"
	"github.com/sibukixxx/wp2emdash/internal/usecase/step"
)

// PresetParams holds the runtime configuration for preset execution.
type PresetParams struct {
	WPRoot        string
	OutDir        string
	Version       string
	RiskBandsPath string
	AgentAuditURL string
	AgentMediaURL string
	AgentToken    string
	AgentTimeout  time.Duration
	SSHTarget     string
	SSHPort       int
	SSHKey        string
}

// defaultRegistry holds the step handlers used by RunPresetStep.
var defaultRegistry = buildRegistry()

func buildRegistry() *step.Registry {
	reg := step.NewRegistry()

	reg.Register("doctor", func(ctx context.Context, _ preset.Step, _ step.Params) (step.Result, error) {
		_ = RunDoctor(ctx)
		return step.Result{}, nil
	})

	reg.Register("audit", func(ctx context.Context, _ preset.Step, p step.Params) (step.Result, error) {
		res, err := RunAudit(ctx, AuditParams{
			WPRoot:        p.WPRoot,
			OutDir:        p.OutDir,
			Write:         true,
			Version:       p.Version,
			RiskBandsPath: p.RiskBandsPath,
			AgentURL:      p.AgentAuditURL,
			AgentToken:    p.AgentToken,
			AgentTimeout:  p.AgentTimeout,
			SSHTarget:     p.SSHTarget,
			SSHPort:       p.SSHPort,
			SSHKey:        p.SSHKey,
		})
		if err != nil {
			return step.Result{}, err
		}
		return step.Result{Warnings: res.Bundle.Warnings}, nil
	})

	uploadsDir := func(p step.Params) string {
		return path.Join(p.WPRoot, "wp-content", "uploads")
	}

	reg.Register("media-scan-sample", func(_ context.Context, _ preset.Step, p step.Params) (step.Result, error) {
		_, err := RunMediaScan(MediaScanParams{
			Dir:          uploadsDir(p),
			OutDir:       p.OutDir,
			MaxFiles:     200,
			AgentURL:     p.AgentMediaURL,
			AgentToken:   p.AgentToken,
			AgentTimeout: p.AgentTimeout,
			SSHTarget:    p.SSHTarget,
			SSHPort:      p.SSHPort,
			SSHKey:       p.SSHKey,
		})
		return step.Result{}, err
	})

	reg.Register("media-scan", func(_ context.Context, _ preset.Step, p step.Params) (step.Result, error) {
		_, err := RunMediaScan(MediaScanParams{
			Dir:          uploadsDir(p),
			OutDir:       p.OutDir,
			AgentURL:     p.AgentMediaURL,
			AgentToken:   p.AgentToken,
			AgentTimeout: p.AgentTimeout,
			SSHTarget:    p.SSHTarget,
			SSHPort:      p.SSHPort,
			SSHKey:       p.SSHKey,
		})
		return step.Result{}, err
	})

	reg.Register("media-scan-hash", func(_ context.Context, _ preset.Step, p step.Params) (step.Result, error) {
		_, err := RunMediaScan(MediaScanParams{
			Dir:          uploadsDir(p),
			OutDir:       p.OutDir,
			Hash:         true,
			AgentURL:     p.AgentMediaURL,
			AgentToken:   p.AgentToken,
			AgentTimeout: p.AgentTimeout,
			SSHTarget:    p.SSHTarget,
			SSHPort:      p.SSHPort,
			SSHKey:       p.SSHKey,
		})
		return step.Result{}, err
	})

	reg.Register("report", func(_ context.Context, _ preset.Step, _ step.Params) (step.Result, error) {
		return step.Result{}, nil
	})
	reg.Register("todo", func(_ context.Context, _ preset.Step, _ step.Params) (step.Result, error) {
		return step.Result{}, nil
	})

	return reg
}

// RunPresetStep executes a single preset step using the default step registry.
// New step kinds can be registered into a custom Registry without modifying
// this function; see package step for the registration API.
func RunPresetStep(ctx context.Context, s preset.Step, params PresetParams) (step.Result, error) {
	return defaultRegistry.Execute(ctx, s, step.Params{
		WPRoot:        params.WPRoot,
		OutDir:        params.OutDir,
		Version:       params.Version,
		RiskBandsPath: params.RiskBandsPath,
		AgentAuditURL: params.AgentAuditURL,
		AgentMediaURL: params.AgentMediaURL,
		AgentToken:    params.AgentToken,
		AgentTimeout:  params.AgentTimeout,
		SSHTarget:     params.SSHTarget,
		SSHPort:       params.SSHPort,
		SSHKey:        params.SSHKey,
	})
}
