package usecase

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/sibukixxx/wp2emdash/internal/shell"
)

func TestRunDoctorWithRunnerUsesRunnerPath(t *testing.T) {
	t.Parallel()

	toolDir := t.TempDir()
	for _, name := range []string{"wp", "wrangler", "git"} {
		path := filepath.Join(toolDir, name)
		if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	report := RunDoctorWithRunner(context.Background(), shell.Runner{
		Env: []string{"PATH=" + toolDir},
	})

	if !report.OK {
		t.Fatal("report.OK: want true, got false")
	}
	for _, check := range report.Checks[:3] {
		if !check.Found {
			t.Fatalf("required tool %s not found", check.Name)
		}
	}
}
