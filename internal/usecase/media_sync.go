package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sibukixxx/wp2emdash/internal/infra/rclone"
	"github.com/sibukixxx/wp2emdash/internal/shell"
)

type MediaSyncParams struct {
	Dir        string
	Dest       string
	OutDir     string
	ReportPath string
	Apply      bool
	Delete     bool
	Checksum   bool
}

type MediaSyncResult struct {
	SourceDir string       `json:"source_dir"`
	Dest      string       `json:"dest"`
	Mode      string       `json:"mode"`
	Applied   bool         `json:"applied"`
	Command   shell.Result `json:"command"`
	Path      string       `json:"-"`
}

func RunMediaSync(ctx context.Context, params MediaSyncParams) (MediaSyncResult, error) {
	return RunMediaSyncWithRunner(ctx, shell.Runner{}, params)
}

func RunMediaSyncWithRunner(ctx context.Context, runner shell.Runner, params MediaSyncParams) (MediaSyncResult, error) {
	mode := "copy"
	if params.Delete {
		mode = "sync"
	}
	res, err := rclone.Run(ctx, shell.Runner{
		Dir:    runner.Dir,
		Env:    runner.Env,
		DryRun: !params.Apply,
	}, rclone.SyncConfig{
		SourceDir: params.Dir,
		Dest:      params.Dest,
	}, rclone.SyncOptions{
		DryRun:   !params.Apply,
		Delete:   params.Delete,
		Checksum: params.Checksum,
	})
	if err != nil {
		return MediaSyncResult{}, fmt.Errorf("rclone %s: %w", mode, err)
	}

	result := MediaSyncResult{
		SourceDir: params.Dir,
		Dest:      params.Dest,
		Mode:      mode,
		Applied:   params.Apply,
		Command:   res,
		Path:      params.ReportPath,
	}
	if result.Path == "" {
		result.Path = filepath.Join(params.OutDir, "media-sync.json")
	}
	if err := writeMediaSyncResult(result.Path, result); err != nil {
		return MediaSyncResult{}, fmt.Errorf("write media sync report: %w", err)
	}
	return result, nil
}

func writeMediaSyncResult(path string, result MediaSyncResult) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}
