package rclone

import (
	"context"
	"fmt"

	"github.com/sibukixxx/wp2emdash/internal/shell"
)

type SyncConfig struct {
	SourceDir string
	Dest      string
}

type SyncOptions struct {
	DryRun   bool
	Delete   bool
	Checksum bool
}

func Run(ctx context.Context, runner shell.Runner, cfg SyncConfig, opt SyncOptions) (shell.Result, error) {
	if cfg.SourceDir == "" {
		return shell.Result{}, fmt.Errorf("source dir is required")
	}
	if cfg.Dest == "" {
		return shell.Result{}, fmt.Errorf("destination is required")
	}

	mode := "copy"
	if opt.Delete {
		mode = "sync"
	}
	args := []string{mode, cfg.SourceDir, cfg.Dest}
	if opt.Checksum {
		args = append(args, "--checksum")
	}
	if opt.DryRun {
		args = append(args, "--dry-run")
	}
	return runner.Run(ctx, "rclone", args...)
}
