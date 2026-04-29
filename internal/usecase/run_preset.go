package usecase

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/rokubunnoni-inc/wp2emdash/internal/domain/preset"
)

type PresetParams struct {
	WPRoot  string
	OutDir  string
	Version string
}

func RunPresetStep(ctx context.Context, step preset.Step, params PresetParams) error {
	switch step.Kind {
	case "doctor":
		_ = RunDoctor(ctx)
		return nil
	case "audit":
		_, err := RunAudit(ctx, AuditParams{
			WPRoot:  params.WPRoot,
			OutDir:  params.OutDir,
			Write:   true,
			Version: params.Version,
		})
		return err
	case "media-scan-sample":
		_, err := RunMediaScan(MediaScanParams{
			Dir:      filepath.Join(params.WPRoot, "wp-content", "uploads"),
			OutDir:   params.OutDir,
			MaxFiles: 200,
		})
		return err
	case "media-scan":
		_, err := RunMediaScan(MediaScanParams{
			Dir:    filepath.Join(params.WPRoot, "wp-content", "uploads"),
			OutDir: params.OutDir,
		})
		return err
	case "media-scan-hash":
		_, err := RunMediaScan(MediaScanParams{
			Dir:    filepath.Join(params.WPRoot, "wp-content", "uploads"),
			OutDir: params.OutDir,
			Hash:   true,
		})
		return err
	case "report", "todo":
		return nil
	default:
		return fmt.Errorf("unhandled step kind %q", step.Kind)
	}
}
