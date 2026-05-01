package usecase_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sibukixxx/wp2emdash/internal/usecase"
)

func TestRunSEOURLMapAcceptsJSONAndPlainTextInputs(t *testing.T) {
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "old.json")
	if err := os.WriteFile(jsonPath, []byte(`{"source":"wp","entries":[{"url":"https://example.com/a/"},{"url":"https://example.com/b/"}]}`), 0o644); err != nil {
		t.Fatalf("write old.json: %v", err)
	}
	textPath := filepath.Join(dir, "new.txt")
	if err := os.WriteFile(textPath, []byte("# emdash export\nhttps://example.com/a/\nhttps://example.com/c/\n"), 0o644); err != nil {
		t.Fatalf("write new.txt: %v", err)
	}

	res, err := usecase.RunSEOURLMap(usecase.SEOURLMapParams{
		OldPath: jsonPath,
		NewPath: textPath,
		OutDir:  dir,
		Write:   true,
		Version: "test",
	})
	if err != nil {
		t.Fatalf("RunSEOURLMap: %v", err)
	}

	if got, want := len(res.Diff.Matched), 1; got != want {
		t.Errorf("Matched count: got %d want %d (%v)", got, want, res.Diff.Matched)
	}
	if got, want := len(res.Diff.OnlyInOld), 1; got != want {
		t.Errorf("OnlyInOld count: got %d want %d (%v)", got, want, res.Diff.OnlyInOld)
	}
	if got, want := len(res.Diff.OnlyInNew), 1; got != want {
		t.Errorf("OnlyInNew count: got %d want %d (%v)", got, want, res.Diff.OnlyInNew)
	}
	if res.Diff.OldSource != "wp" {
		t.Errorf("OldSource should pass through from JSON input, got %q", res.Diff.OldSource)
	}
	if res.Diff.NewSource != "new" {
		t.Errorf("NewSource should default to file stem for text input, got %q", res.Diff.NewSource)
	}
	if res.Diff.GeneratedAt == "" || res.Diff.Tool != "wp2emdash" {
		t.Errorf("envelope fields not populated: %+v", res.Diff)
	}
	if _, err := os.Stat(res.Path); err != nil {
		t.Errorf("expected output file at %s: %v", res.Path, err)
	}
}

func TestRunSEOURLMapRequiresBothPaths(t *testing.T) {
	_, err := usecase.RunSEOURLMap(usecase.SEOURLMapParams{OldPath: "a.txt"})
	if err == nil {
		t.Fatalf("expected error when --new is missing")
	}
}

func TestRunSEOURLMapDoesNotWriteWhenWriteIsFalse(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.txt")
	b := filepath.Join(dir, "b.txt")
	if err := os.WriteFile(a, []byte("https://example.com/\n"), 0o644); err != nil {
		t.Fatalf("write a: %v", err)
	}
	if err := os.WriteFile(b, []byte("https://example.com/\n"), 0o644); err != nil {
		t.Fatalf("write b: %v", err)
	}
	res, err := usecase.RunSEOURLMap(usecase.SEOURLMapParams{
		OldPath: a, NewPath: b, OutDir: dir, Write: false, Version: "test",
	})
	if err != nil {
		t.Fatalf("RunSEOURLMap: %v", err)
	}
	if _, err := os.Stat(res.Path); !os.IsNotExist(err) {
		t.Errorf("expected output file NOT to exist, stat err=%v", err)
	}
}
