package usecase

import (
	"context"
	"testing"

	"github.com/sibukixxx/wp2emdash/internal/domain/audit"
	"github.com/sibukixxx/wp2emdash/internal/domain/source"
)

type fakeAuditor struct {
	result   audit.Audit
	warnings []source.Warning
}

func (f fakeAuditor) Run(context.Context) (audit.Audit, error) {
	return f.result, nil
}

func (f fakeAuditor) Warnings() []source.Warning {
	return f.warnings
}

func TestRunAuditFromSourceIncludesWarnings(t *testing.T) {
	t.Parallel()

	src := fakeAuditor{
		result: audit.Audit{},
		warnings: []source.Warning{
			{Code: "content.posts", Message: "probe failed"},
		},
	}

	got, err := RunAuditFromSource(context.Background(), src, AuditParams{
		OutDir:  t.TempDir(),
		Write:   false,
		Version: "test",
	})
	if err != nil {
		t.Fatalf("RunAuditFromSource() error = %v", err)
	}
	if len(got.Bundle.Warnings) != 1 {
		t.Fatalf("warnings: want 1, got %d", len(got.Bundle.Warnings))
	}
	if got.Bundle.Warnings[0].Code != "content.posts" {
		t.Fatalf("warning code: want content.posts, got %q", got.Bundle.Warnings[0].Code)
	}
	if got.Bundle.Score.Level == "" {
		t.Fatal("score level is empty")
	}
	if got.Bundle.Score.Estimate == "" {
		t.Fatal("score estimate is empty")
	}
}
