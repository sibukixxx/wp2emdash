// Package source defines the interface that migration-source adapters must
// implement. The WordPress adapter lives in infra/wpcli; future adapters for
// other CMSes (Drupal, Joomla, etc.) implement this same interface.
package source

import (
	"context"

	"github.com/sibukixxx/wp2emdash/internal/domain/audit"
)

// Warning captures a best-effort audit probe failure that did not abort the
// whole audit, but may have left some metrics incomplete or zero-valued.
type Warning struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Auditor collects complexity metrics from a migration source site.
// Implementations are responsible for all I/O and external tool calls;
// the domain and scoring layers remain free of side effects.
type Auditor interface {
	Run(ctx context.Context) (audit.Audit, error)
}

// WarningReporter is an optional extension for auditors that can surface
// partial-failure diagnostics alongside a successful Audit result.
type WarningReporter interface {
	Warnings() []Warning
}
