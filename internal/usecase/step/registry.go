// Package step provides a registry for preset step handlers, allowing new
// step kinds to be added without modifying the core run_preset dispatcher.
package step

import (
	"context"
	"fmt"
	"time"

	"github.com/sibukixxx/wp2emdash/internal/domain/preset"
	"github.com/sibukixxx/wp2emdash/internal/domain/source"
)

// Params carries the runtime parameters shared across all step handlers.
type Params struct {
	WPRoot        string
	OutDir        string
	Version       string
	RiskBandsPath string
	AgentAuditURL string
	AgentMediaURL string
	AgentToken    string
	AgentTimeout  time.Duration
	SSHTarget     string
	SSHPort       int
	SSHKey        string
}

// Result carries non-fatal step diagnostics back to the preset runner.
type Result struct {
	Warnings []source.Warning
}

// HandlerFunc executes a single preset step.
type HandlerFunc func(ctx context.Context, s preset.Step, p Params) (Result, error)

// Registry maps step kind strings to their handler functions.
type Registry struct {
	handlers map[string]HandlerFunc
}

// NewRegistry returns an empty Registry.
func NewRegistry() *Registry {
	return &Registry{handlers: make(map[string]HandlerFunc)}
}

// Register associates kind with fn. Calling Register twice for the same kind
// replaces the previous handler.
func (r *Registry) Register(kind string, fn HandlerFunc) {
	r.handlers[kind] = fn
}

// Execute looks up the handler for s.Kind and calls it. Returns an error if
// no handler is registered for the given kind.
func (r *Registry) Execute(ctx context.Context, s preset.Step, p Params) (Result, error) {
	fn, ok := r.handlers[s.Kind]
	if !ok {
		return Result{}, fmt.Errorf("unhandled step kind %q", s.Kind)
	}
	return fn(ctx, s, p)
}
