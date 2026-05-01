package score

import (
	"testing"

	"github.com/sibukixxx/wp2emdash/internal/domain/audit"
)

func TestComputeReturnsSimpleForEmptyAudit(t *testing.T) {
	got := Compute(audit.Audit{})
	if got.Score != 0 {
		t.Fatalf("score: want 0, got %d", got.Score)
	}
	if got.Level != "" {
		t.Fatalf("level: want empty, got %q", got.Level)
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
	if got.Level != "" {
		t.Fatalf("level: want empty, got %q", got.Level)
	}
}
