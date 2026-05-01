package agenthttp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sibukixxx/wp2emdash/internal/domain/audit"
	"github.com/sibukixxx/wp2emdash/internal/domain/source"
)

type Auditor struct {
	Endpoint string
	Token    string
	Client   *http.Client

	warnings []source.Warning
}

func NewAuditor(endpoint, token string, timeout time.Duration) (*Auditor, error) {
	if strings.TrimSpace(endpoint) == "" {
		return nil, fmt.Errorf("agent url is required")
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &Auditor{
		Endpoint: endpoint,
		Token:    token,
		Client:   &http.Client{Timeout: timeout},
	}, nil
}

func (a *Auditor) Run(ctx context.Context) (audit.Audit, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.Endpoint, nil)
	if err != nil {
		return audit.Audit{}, fmt.Errorf("build agent request: %w", err)
	}
	if a.Token != "" {
		req.Header.Set("Authorization", "Bearer "+a.Token)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := a.Client.Do(req)
	if err != nil {
		return audit.Audit{}, fmt.Errorf("request agent: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return audit.Audit{}, fmt.Errorf("agent returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var payload auditResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return audit.Audit{}, fmt.Errorf("decode agent response: %w", err)
	}
	a.warnings = append(a.warnings[:0], payload.Warnings...)
	return payload.Audit, nil
}

func (a *Auditor) Warnings() []source.Warning {
	if len(a.warnings) == 0 {
		return nil
	}
	out := make([]source.Warning, len(a.warnings))
	copy(out, a.warnings)
	return out
}
