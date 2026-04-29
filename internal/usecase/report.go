package usecase

import (
	"fmt"

	"github.com/rokubunnoni-inc/wp2emdash/internal/usecase/reporting"
)

func LoadReportBundle(path string) (reporting.Bundle, error) {
	bundle, err := reporting.ReadBundle(path)
	if err != nil {
		return reporting.Bundle{}, fmt.Errorf("load bundle: %w", err)
	}
	return bundle, nil
}

func WriteReport(outDir string, bundle reporting.Bundle) error {
	if err := reporting.WriteAll(outDir, bundle); err != nil {
		return fmt.Errorf("write report: %w", err)
	}
	return nil
}
