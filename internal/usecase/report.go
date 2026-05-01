package usecase

import (
	"fmt"
	"path/filepath"

	"github.com/sibukixxx/wp2emdash/internal/usecase/reporting"
)

func LoadReportBundle(path string) (reporting.Bundle, error) {
	bundle, err := reporting.ReadBundle(path)
	if err != nil {
		return reporting.Bundle{}, fmt.Errorf("load bundle: %w", err)
	}
	return bundle, nil
}

func WriteReport(outDir string, bundle reporting.Bundle) error {
	if err := reporting.WriteMarkdown(filepath.Join(outDir, "risk-report.md"), bundle); err != nil {
		return fmt.Errorf("write report: %w", err)
	}
	return nil
}
