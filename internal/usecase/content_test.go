package usecase

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/sibukixxx/wp2emdash/internal/domain/contentverify"
)

func TestRunContentVerifyWritesEvidenceAndResolvedMap(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	expectedPath := filepath.Join(dir, "expected.json")
	actualPath := filepath.Join(dir, "actual.json")
	body := contentverify.FingerprintHTML("<p>Hello</p>")
	expected := contentverify.Snapshot{Complete: true, Entries: []contentverify.Entry{{ID: "3", Kind: "post", Slug: "hello", Status: "publish", Title: "Hello", Body: body}}}
	actual := contentverify.Snapshot{Complete: true, Entries: []contentverify.Entry{{ID: "emdash-3", Kind: "posts", Slug: "hello", Status: "published", Title: "Hello", Body: body}}}
	if err := writeContentJSON(expectedPath, expected); err != nil {
		t.Fatal(err)
	}
	if err := writeContentJSON(actualPath, actual); err != nil {
		t.Fatal(err)
	}
	res, err := RunContentVerify(ContentVerifyParams{ExpectedPath: expectedPath, ActualPath: actualPath, OutDir: dir, Version: "test", Write: true})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Report.OK {
		t.Fatalf("report failed: %+v", res.Report.Issues)
	}
	for _, p := range []string{res.JSONPath, res.MarkdownPath, res.ResolvedMapPath} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("missing artifact %s: %v", p, err)
		}
	}
}

func TestRunContentSnapshotRejectsUnknownSource(t *testing.T) {
	t.Parallel()
	_, err := RunContentSnapshot(context.Background(), ContentSnapshotParams{Source: "other"})
	if err == nil {
		t.Fatal("expected error")
	}
}
