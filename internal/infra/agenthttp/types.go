package agenthttp

import (
	"github.com/sibukixxx/wp2emdash/internal/domain/audit"
	"github.com/sibukixxx/wp2emdash/internal/domain/source"
)

type auditResponse struct {
	Audit    audit.Audit      `json:"audit"`
	Warnings []source.Warning `json:"warnings,omitempty"`
}
