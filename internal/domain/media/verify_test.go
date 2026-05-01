package media

import "testing"

func TestCompareReportsMissingExtraAndHashMismatches(t *testing.T) {
	t.Parallel()

	expected := Manifest{
		BaseDir:    "/expected",
		TotalFiles: 2,
		TotalBytes: 30,
		Files: []File{
			{Path: "a.txt", Size: 10, SHA256: "aaa"},
			{Path: "b.txt", Size: 20, SHA256: "bbb"},
		},
	}
	actual := Manifest{
		BaseDir:    "/actual",
		TotalFiles: 2,
		TotalBytes: 32,
		Files: []File{
			{Path: "a.txt", Size: 11, SHA256: "ccc"},
			{Path: "c.txt", Size: 21, SHA256: "ddd"},
		},
	}

	report := Compare(expected, actual, true)
	if report.OK {
		t.Fatal("report.OK: want false, got true")
	}
	if report.MissingFiles != 1 {
		t.Fatalf("MissingFiles: want 1, got %d", report.MissingFiles)
	}
	if report.ExtraFiles != 1 {
		t.Fatalf("ExtraFiles: want 1, got %d", report.ExtraFiles)
	}
	if report.SizeMismatches != 1 {
		t.Fatalf("SizeMismatches: want 1, got %d", report.SizeMismatches)
	}
	if report.HashMismatches != 1 {
		t.Fatalf("HashMismatches: want 1, got %d", report.HashMismatches)
	}
}

func TestCompareReportsOKForMatchingFiles(t *testing.T) {
	t.Parallel()

	manifest := Manifest{
		BaseDir:    "/same",
		TotalFiles: 1,
		TotalBytes: 10,
		Files: []File{
			{Path: "hello.txt", Size: 10, SHA256: "hash"},
		},
	}

	report := Compare(manifest, manifest, true)
	if !report.OK {
		t.Fatal("report.OK: want true, got false")
	}
	if report.MatchedFiles != 1 {
		t.Fatalf("MatchedFiles: want 1, got %d", report.MatchedFiles)
	}
}
