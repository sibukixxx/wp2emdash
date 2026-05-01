package agenthttp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sibukixxx/wp2emdash/internal/domain/media"
)

func ScanMedia(ctx context.Context, endpoint, token string, timeout time.Duration, params MediaScanParams) (media.Manifest, error) {
	if strings.TrimSpace(endpoint) == "" {
		return media.Manifest{}, fmt.Errorf("agent url is required")
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	u, err := url.Parse(endpoint)
	if err != nil {
		return media.Manifest{}, fmt.Errorf("parse agent url: %w", err)
	}
	q := u.Query()
	if params.Dir != "" {
		q.Set("dir", params.Dir)
	}
	if params.Hash {
		q.Set("hash", "1")
	}
	if params.HistogramOnly {
		q.Set("histogram_only", "1")
	}
	if params.MaxFiles > 0 {
		q.Set("max_files", fmt.Sprintf("%d", params.MaxFiles))
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return media.Manifest{}, fmt.Errorf("build agent request: %w", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return media.Manifest{}, fmt.Errorf("request agent media scan: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return media.Manifest{}, fmt.Errorf("agent returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var manifest media.Manifest
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return media.Manifest{}, fmt.Errorf("decode agent media response: %w", err)
	}
	if manifest.Extensions == nil {
		manifest.Extensions = map[string]int{}
	}
	return manifest, nil
}

type MediaScanParams struct {
	Dir           string
	Hash          bool
	MaxFiles      int
	HistogramOnly bool
}
