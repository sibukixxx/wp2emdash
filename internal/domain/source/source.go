// Package source defines the interface that migration-source adapters must
// implement. The WordPress adapter lives in infra/wpcli; future adapters for
// other CMSes (Drupal, Joomla, etc.) implement this same interface.
package source

import (
	"context"

	"github.com/rokubunnoni-inc/wp2emdash/internal/domain/audit"
)

// Auditor collects complexity metrics from a migration source site.
// Implementations are responsible for all I/O and external tool calls;
// the domain and scoring layers remain free of side effects.
type Auditor interface {
	Run(ctx context.Context) (audit.Audit, error)
}
