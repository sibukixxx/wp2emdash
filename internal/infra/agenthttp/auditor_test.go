package agenthttp

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestAuditorRunIncludesWarningsAndAuth(t *testing.T) {
	t.Parallel()

	auditor, err := NewAuditor("https://agent.example.test/audit", "secret", time.Second)
	if err != nil {
		t.Fatalf("NewAuditor() error = %v", err)
	}
	auditor.Client = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if got := r.Header.Get("Authorization"); got != "Bearer secret" {
				t.Fatalf("Authorization = %q", got)
			}
			body := `{
				"audit": {
					"site": {"home_url":"https://example.test","site_url":"https://example.test","wp_version":"6.5.0","php_version":"8.2.12","db_prefix":"wp_","is_multisite":"no"},
					"content": {"posts": 120},
					"uploads": {"exists": true, "size":"12KB", "file_count":3},
					"theme": {"active_theme":"test-theme"},
					"plugins": {"active_count":2},
					"customization": {"shortcode_post_count":5}
				},
				"warnings": [{"code":"content.posts","message":"sample"}]
			}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(bytes.NewBufferString(body)),
			}, nil
		}),
	}

	got, err := auditor.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got.Site.HomeURL != "https://example.test" {
		t.Fatalf("home_url = %q", got.Site.HomeURL)
	}
	if len(auditor.Warnings()) != 1 {
		t.Fatalf("warnings = %d", len(auditor.Warnings()))
	}
}

func TestAuditorRunFailsOnNon200(t *testing.T) {
	t.Parallel()

	auditor, err := NewAuditor("https://agent.example.test/audit", "", time.Second)
	if err != nil {
		t.Fatalf("NewAuditor() error = %v", err)
	}
	auditor.Client = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusUnauthorized,
				Status:     "401 Unauthorized",
				Body:       io.NopCloser(bytes.NewBufferString("denied")),
			}, nil
		}),
	}

	if _, err := auditor.Run(context.Background()); err == nil {
		t.Fatal("Run() error = nil, want non-nil")
	}
}
