package riskbands

import (
	"testing"

	"github.com/sibukixxx/wp2emdash/internal/domain/score"
)

func TestLoadDefaultAndClassify(t *testing.T) {
	t.Parallel()

	policy, err := Load("")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	level, estimate, err := policy.Classify(91)
	if err != nil {
		t.Fatalf("Classify() error = %v", err)
	}
	if level != score.LevelHighRisk {
		t.Fatalf("level = %q", level)
	}
	if estimate == "" {
		t.Fatal("estimate is empty")
	}
}
