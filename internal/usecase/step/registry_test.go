package step_test

import (
	"context"
	"errors"
	"testing"

	"github.com/rokubunnoni-inc/wp2emdash/internal/domain/preset"
	"github.com/rokubunnoni-inc/wp2emdash/internal/usecase/step"
)

func TestRegisterAndExecute(t *testing.T) {
	t.Run("executes a registered handler", func(t *testing.T) {
		reg := step.NewRegistry()
		called := false
		reg.Register("do-thing", func(_ context.Context, _ preset.Step, _ step.Params) error {
			called = true
			return nil
		})

		s := preset.Step{Kind: "do-thing", Summary: "test"}
		if err := reg.Execute(context.Background(), s, step.Params{}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !called {
			t.Error("handler was not called")
		}
	})

	t.Run("returns error for unknown step kind", func(t *testing.T) {
		reg := step.NewRegistry()
		s := preset.Step{Kind: "nonexistent", Summary: "nope"}
		err := reg.Execute(context.Background(), s, step.Params{})
		if err == nil {
			t.Fatal("expected error for unknown step kind")
		}
	})

	t.Run("propagates handler error", func(t *testing.T) {
		reg := step.NewRegistry()
		sentinel := errors.New("handler failure")
		reg.Register("fail", func(_ context.Context, _ preset.Step, _ step.Params) error {
			return sentinel
		})

		s := preset.Step{Kind: "fail", Summary: "should fail"}
		err := reg.Execute(context.Background(), s, step.Params{})
		if !errors.Is(err, sentinel) {
			t.Errorf("want sentinel error, got %v", err)
		}
	})
}
