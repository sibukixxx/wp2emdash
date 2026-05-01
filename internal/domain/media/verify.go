package media

type VerifyIssue struct {
	Path           string `json:"path,omitempty"`
	Code           string `json:"code"`
	Message        string `json:"message"`
	ExpectedSize   int64  `json:"expected_size,omitempty"`
	ActualSize     int64  `json:"actual_size,omitempty"`
	ExpectedSHA256 string `json:"expected_sha256,omitempty"`
	ActualSHA256   string `json:"actual_sha256,omitempty"`
}

type VerifyReport struct {
	OK              bool          `json:"ok"`
	ExpectedBaseDir string        `json:"expected_base_dir"`
	ActualBaseDir   string        `json:"actual_base_dir"`
	ExpectedFiles   int           `json:"expected_files"`
	ActualFiles     int           `json:"actual_files"`
	ExpectedBytes   int64         `json:"expected_bytes"`
	ActualBytes     int64         `json:"actual_bytes"`
	MatchedFiles    int           `json:"matched_files"`
	MissingFiles    int           `json:"missing_files"`
	ExtraFiles      int           `json:"extra_files"`
	SizeMismatches  int           `json:"size_mismatches"`
	HashMismatches  int           `json:"hash_mismatches"`
	Issues          []VerifyIssue `json:"issues,omitempty"`
}

func Compare(expected, actual Manifest, checkHash bool) VerifyReport {
	report := VerifyReport{
		OK:              true,
		ExpectedBaseDir: expected.BaseDir,
		ActualBaseDir:   actual.BaseDir,
		ExpectedFiles:   expected.TotalFiles,
		ActualFiles:     actual.TotalFiles,
		ExpectedBytes:   expected.TotalBytes,
		ActualBytes:     actual.TotalBytes,
	}

	expectedByPath := make(map[string]File, len(expected.Files))
	for _, f := range expected.Files {
		expectedByPath[f.Path] = f
	}
	actualByPath := make(map[string]File, len(actual.Files))
	for _, f := range actual.Files {
		actualByPath[f.Path] = f
	}

	for path, exp := range expectedByPath {
		act, ok := actualByPath[path]
		if !ok {
			report.MissingFiles++
			report.Issues = append(report.Issues, VerifyIssue{
				Path:    path,
				Code:    "missing",
				Message: "file is missing on the verification target",
			})
			report.OK = false
			continue
		}

		matched := true
		if exp.Size != act.Size {
			report.SizeMismatches++
			report.Issues = append(report.Issues, VerifyIssue{
				Path:         path,
				Code:         "size_mismatch",
				Message:      "file size differs from the manifest",
				ExpectedSize: exp.Size,
				ActualSize:   act.Size,
			})
			report.OK = false
			matched = false
		}
		if checkHash && exp.SHA256 != "" {
			if act.SHA256 == "" || exp.SHA256 != act.SHA256 {
				report.HashMismatches++
				report.Issues = append(report.Issues, VerifyIssue{
					Path:           path,
					Code:           "hash_mismatch",
					Message:        "file hash differs from the manifest",
					ExpectedSHA256: exp.SHA256,
					ActualSHA256:   act.SHA256,
				})
				report.OK = false
				matched = false
			}
		}
		if matched {
			report.MatchedFiles++
		}
	}

	for path := range actualByPath {
		if _, ok := expectedByPath[path]; ok {
			continue
		}
		report.ExtraFiles++
		report.Issues = append(report.Issues, VerifyIssue{
			Path:    path,
			Code:    "extra",
			Message: "file exists on the verification target but not in the manifest",
		})
		report.OK = false
	}

	return report
}
