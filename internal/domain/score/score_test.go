package score

import (
	"testing"

	"github.com/rokubunnoni-inc/wp2emdash/internal/domain/audit"
)

func TestComputeReturnsSimpleForEmptyAudit(t *testing.T) {
	got := Compute(audit.Audit{})
	if got.Score != 0 {
		t.Fatalf("score: want 0, got %d", got.Score)
	}
	if got.Level != LevelSimple {
		t.Fatalf("level: want %q, got %q", LevelSimple, got.Level)
	}
	if len(got.Reasons) != 0 {
		t.Fatalf("reasons: want 0, got %d", len(got.Reasons))
	}
}

func TestComputeAccumulatesSignals(t *testing.T) {
	a := audit.Audit{}
	a.Content.Posts = 600
	a.Content.Pages = 25
	a.Plugins.ActiveCount = 22
	a.Plugins.HasACF = true
	a.Plugins.HasMember = true
	a.Customization.CustomPostTypeCount = 4

	got := Compute(a)
	// 5 (>100) + 10 (>500) + 5 (pages>20) + 5 (plug>10) + 10 (plug>20)
	// + 15 (acf) + 25 (member) + 10 (cpt>0) + 15 (cpt>=3) = 100
	if got.Score != 100 {
		t.Fatalf("score: want 100, got %d", got.Score)
	}
	if got.Level != LevelHighRisk {
		t.Fatalf("level: want %q, got %q", LevelHighRisk, got.Level)
	}
}

func TestLevelForBoundaries(t *testing.T) {
	cases := []struct {
		score int
		want  Level
	}{
		{0, LevelSimple},
		{20, LevelSimple},
		{21, LevelStandard},
		{50, LevelStandard},
		{51, LevelComplex},
		{90, LevelComplex},
		{91, LevelHighRisk},
		{130, LevelHighRisk},
		{131, LevelRebuild},
		{500, LevelRebuild},
	}
	for _, c := range cases {
		got, _ := LevelFor(c.score)
		if got != c.want {
			t.Errorf("score=%d: want %q, got %q", c.score, c.want, got)
		}
	}
}
