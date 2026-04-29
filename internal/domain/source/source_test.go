package source_test

import (
	"testing"

	"github.com/rokubunnoni-inc/wp2emdash/internal/domain/source"
	"github.com/rokubunnoni-inc/wp2emdash/internal/infra/wpcli"
)

// TestWPCLIAuditorImplementsAuditor is a compile-time interface check.
// If wpcli.Auditor stops satisfying source.Auditor, this test fails to build.
func TestWPCLIAuditorImplementsAuditor(t *testing.T) {
	var _ source.Auditor = (*wpcli.Auditor)(nil)
}
