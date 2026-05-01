package rclone

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sibukixxx/wp2emdash/internal/shell"
)

func TestRunBuildsDryRunCopyCommand(t *testing.T) {
	t.Parallel()

	res, err := Run(context.Background(), shell.Runner{DryRun: true}, SyncConfig{
		SourceDir: "/src",
		Dest:      "remote:bucket/uploads",
	}, SyncOptions{
		DryRun:   true,
		Checksum: true,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !strings.Contains(res.Command, "rclone copy /src remote:bucket/uploads --checksum --dry-run") {
		t.Fatalf("command = %q", res.Command)
	}
}

func TestRunExecutesSyncMode(t *testing.T) {
	binDir := t.TempDir()
	rclonePath := filepath.Join(binDir, "rclone")
	if err := os.WriteFile(rclonePath, []byte("#!/bin/sh\nprintf '%s' \"$*\"\n"), 0o755); err != nil {
		t.Fatalf("write rclone stub: %v", err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	res, err := Run(context.Background(), shell.Runner{}, SyncConfig{
		SourceDir: "/src",
		Dest:      "remote:bucket/uploads",
	}, SyncOptions{
		Delete: true,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if strings.TrimSpace(res.Stdout) != "sync /src remote:bucket/uploads" {
		t.Fatalf("stdout = %q", res.Stdout)
	}
}
