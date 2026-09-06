package assessment

import (
	"reflect"
	"testing"
	"time"

	"github.com/sibukixxx/wp2emdash/internal/domain/audit"
	"github.com/sibukixxx/wp2emdash/internal/domain/score"
	"github.com/sibukixxx/wp2emdash/internal/domain/source"
)

func TestBuildPreservesUnavailableAndEvidence(t *testing.T) {
	a := audit.Audit{Plugins: audit.PluginsStats{HasWooCommerce: true}, Content: audit.ContentStats{Posts: 501}}
	w := []source.Warning{{Code: "plugins.active_json", Message: "permission denied"}}
	now := time.Unix(10, 0)
	one := Build(a, w, score.Compute(a), "test", now, now.Add(time.Second))
	two := Build(a, w, score.Compute(a), "test", now, now.Add(time.Second))
	if one.Inventory.Plugins.Status != StatusUnavailable {
		t.Fatalf("status = %q", one.Inventory.Plugins.Status)
	}
	if one.Score.Total == 0 || len(one.Signals) == 0 || len(one.Signals[0].Evidence) == 0 {
		t.Fatal("score must be traceable to evidence")
	}
	if !reflect.DeepEqual(one, two) {
		t.Fatal("same input and clock must be deterministic")
	}
	if one.Score.Breakdown["content"] == 0 {
		t.Fatal("content breakdown missing")
	}
}
