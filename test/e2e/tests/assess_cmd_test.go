package tests

import (
	"os"
	"path/filepath"
	"testing"

	e2e "github.com/sibukixxx/wp2emdash/test/e2e"
)

type assessmentSummary struct {
	SchemaVersion string `json:"schema_version"`
	ReadOnly      bool   `json:"read_only"`
	Score         struct {
		Total     int            `json:"total"`
		Breakdown map[string]int `json:"breakdown"`
	} `json:"technical_score"`
}

func TestAssessWritesVersionedReadOnlyArtifacts(t *testing.T) {
	cli := e2e.NewCLI(t)
	out := t.TempDir()
	res := cli.Run(t, "assess", "--wp-root", cli.FixtureDir, "--out", out, "--json")
	if res.Stderr != "" {
		t.Fatalf("stderr = %q", res.Stderr)
	}
	for _, name := range []string{"summary.json", "inventory.json", "signals.json", "technical-report.md"} {
		if _, err := os.Stat(filepath.Join(out, name)); err != nil {
			t.Fatalf("artifact %s: %v", name, err)
		}
	}
	summary := e2e.DecodeJSONFile[assessmentSummary](t, filepath.Join(out, "summary.json"))
	if summary.SchemaVersion != "1.0" || !summary.ReadOnly {
		t.Fatalf("unexpected assessment metadata: %#v", summary)
	}
	total := 0
	for _, points := range summary.Score.Breakdown {
		total += points
	}
	if total != summary.Score.Total {
		t.Fatalf("breakdown total %d != score %d", total, summary.Score.Total)
	}
}
