package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sibukixxx/wp2emdash/internal/usecase"
	"github.com/sibukixxx/wp2emdash/test/e2e"
)

func TestDBPlanCommand_WritesPlanArtifacts(t *testing.T) {
	t.Parallel()

	cli := e2e.NewCLI(t)
	outDir := filepath.Join(t.TempDir(), "out")

	cli.Run(t,
		"audit",
		"--wp-root", cli.FixtureDir,
		"--out", outDir,
	)
	res := cli.Run(t,
		"db", "plan",
		"--from", filepath.Join(outDir, "summary.json"),
		"--out", outDir,
		"--preset", "small-production",
	)

	if !strings.Contains(res.Stdout, "strategy:") {
		t.Fatalf("stdout missing strategy:\n%s", res.Stdout)
	}

	plan := e2e.DecodeJSONFile[usecase.DBPlan](t, filepath.Join(outDir, "db-plan.json"))
	if len(plan.Tables) == 0 {
		t.Fatal("tables: want entries, got 0")
	}
	if _, err := os.Stat(filepath.Join(outDir, "db-plan.md")); err != nil {
		t.Fatalf("db-plan.md missing: %v", err)
	}
}
