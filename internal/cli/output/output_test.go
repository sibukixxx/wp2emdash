package output_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/rokubunnoni-inc/wp2emdash/internal/cli/output"
)

func TestJSON(t *testing.T) {
	t.Run("encodes value as indented JSON", func(t *testing.T) {
		var buf bytes.Buffer
		if err := output.JSON(&buf, map[string]int{"score": 42}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got := buf.String()
		if !strings.Contains(got, `"score": 42`) {
			t.Errorf("want indented JSON with score:42, got %q", got)
		}
		if !strings.HasSuffix(got, "\n") {
			t.Errorf("JSON output should end with newline, got %q", got)
		}
	})
}

func TestPrintln(t *testing.T) {
	t.Run("writes line with newline", func(t *testing.T) {
		var buf bytes.Buffer
		if err := output.Println(&buf, "hello world"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := buf.String(); got != "hello world\n" {
			t.Errorf("want %q, got %q", "hello world\n", got)
		}
	})
}

func TestPrintf(t *testing.T) {
	tests := []struct {
		name   string
		format string
		args   []any
		want   string
	}{
		{"no args", "plain text", nil, "plain text"},
		{"with args", "score: %d", []any{99}, "score: 99"},
		{"multiple args", "%s=%d", []any{"x", 1}, "x=1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := output.Printf(&buf, tt.format, tt.args...); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := buf.String(); got != tt.want {
				t.Errorf("want %q, got %q", tt.want, got)
			}
		})
	}
}
