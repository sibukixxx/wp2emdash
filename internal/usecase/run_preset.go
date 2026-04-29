package usecase

import (
	"context"
	"path/filepath"

	"github.com/rokubunnoni-inc/wp2emdash/internal/domain/preset"
	"github.com/rokubunnoni-inc/wp2emdash/internal/usecase/step"
)

// PresetParams holds the runtime configuration for preset execution.
type PresetParams struct {
	WPRoot  string
	OutDir  string
	Version string
}

// defaultRegistry holds the step handlers used by RunPresetStep.
var defaultRegistry = buildRegistry()

func buildRegistry() *step.Registry {
	reg := step.NewRegistry()

	reg.Register("doctor", func(ctx context.Context, _ preset.Step, _ step.Params) error {
		_ = RunDoctor(ctx)
		return nil
	})

	reg.Register("audit", func(ctx context.Context, _ preset.Step, p step.Params) error {
		_, err := RunAudit(ctx, AuditParams{
			WPRoot:  p.WPRoot,
			OutDir:  p.OutDir,
			Write:   true,
			Version: p.Version,
		})
		return err
	})

	uploadsDir := func(p step.Params) string {
		return filepath.Join(p.WPRoot, "wp-content", "uploads")
	}

	reg.Register("media-scan-sample", func(_ context.Context, _ preset.Step, p step.Params) error {
		_, err := RunMediaScan(MediaScanParams{
			Dir:      uploadsDir(p),
			OutDir:   p.OutDir,
			MaxFiles: 200,
		})
		return err
	})

	reg.Register("media-scan", func(_ context.Context, _ preset.Step, p step.Params) error {
		_, err := RunMediaScan(MediaScanParams{
			Dir:    uploadsDir(p),
			OutDir: p.OutDir,
		})
		return err
	})

	reg.Register("media-scan-hash", func(_ context.Context, _ preset.Step, p step.Params) error {
		_, err := RunMediaScan(MediaScanParams{
			Dir:    uploadsDir(p),
			OutDir: p.OutDir,
			Hash:   true,
		})
		return err
	})

	reg.Register("report", func(_ context.Context, _ preset.Step, _ step.Params) error {
		return nil
	})
	reg.Register("todo", func(_ context.Context, _ preset.Step, _ step.Params) error {
		return nil
	})

	return reg
}

// RunPresetStep executes a single preset step using the default step registry.
// New step kinds can be registered into a custom Registry without modifying
// this function; see package step for the registration API.
func RunPresetStep(ctx context.Context, s preset.Step, params PresetParams) error {
	return defaultRegistry.Execute(ctx, s, step.Params{
		WPRoot:  params.WPRoot,
		OutDir:  params.OutDir,
		Version: params.Version,
	})
}
