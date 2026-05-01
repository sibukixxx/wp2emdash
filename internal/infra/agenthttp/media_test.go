package agenthttp

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"
	"time"
)

func TestScanMediaBuildsQueryAndAuth(t *testing.T) {
	orig := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = orig })
	http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if got := r.Header.Get("Authorization"); got != "Bearer token" {
			t.Fatalf("Authorization = %q", got)
		}
		q := r.URL.Query()
		if q.Get("dir") != "wp-content/uploads" || q.Get("hash") != "1" || q.Get("max_files") != "200" || q.Get("histogram_only") != "1" {
			t.Fatalf("unexpected query: %s", r.URL.RawQuery)
		}
		body := `{"base_dir":"wp-content/uploads","total_files":3,"total_bytes":12,"extensions":{"txt":1},"files":[{"path":"2024/01/hello.txt","size":12,"ext":"txt","mime":"text/plain"}]}`
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewBufferString(body)),
		}, nil
	})

	got, err := ScanMedia(context.Background(), "https://agent.example.test/media-scan", "token", time.Second, MediaScanParams{
		Dir:           "wp-content/uploads",
		Hash:          true,
		MaxFiles:      200,
		HistogramOnly: true,
	})
	if err != nil {
		t.Fatalf("ScanMedia() error = %v", err)
	}
	if got.TotalFiles != 3 {
		t.Fatalf("total_files = %d", got.TotalFiles)
	}
}
