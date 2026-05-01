package usecase

import (
	"context"
	"strings"
	"testing"

	"github.com/sibukixxx/wp2emdash/internal/shell"
)

func TestRunMediaSyncWithRunnerWritesDryRunReport(t *testing.T) {
	t.Parallel()

	res, err := RunMediaSyncWithRunner(context.Background(), shell.Runner{}, MediaSyncParams{
		Dir:      "/src",
		Dest:     "remote:bucket/uploads",
		OutDir:   t.TempDir(),
		Apply:    false,
		Checksum: true,
	})
	if err != nil {
		t.Fatalf("RunMediaSyncWithRunner() error = %v", err)
	}
	if res.Applied {
		t.Fatal("Applied: want false, got true")
	}
	if !strings.Contains(res.Command.Command, "rclone copy /src remote:bucket/uploads --checksum --dry-run") {
		t.Fatalf("command = %q", res.Command.Command)
	}
}
