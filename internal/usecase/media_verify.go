package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sibukixxx/wp2emdash/internal/domain/media"
)

type MediaVerifyParams struct {
	FromManifest   string
	ActualManifest string
	Dir            string
	OutDir         string
	ReportPath     string
	SkipHash       bool
	SSHTarget      string
	SSHPort        int
	SSHKey         string
	AgentURL       string
	AgentToken     string
	AgentTimeout   time.Duration
}

type MediaVerifyResult struct {
	ExpectedManifestPath string
	ActualManifestPath   string
	Expected             media.Manifest
	Actual               media.Manifest
	Report               media.VerifyReport
	Path                 string
}

func RunMediaVerify(ctx context.Context, params MediaVerifyParams) (MediaVerifyResult, error) {
	expected, err := readMediaManifest(params.FromManifest)
	if err != nil {
		return MediaVerifyResult{}, fmt.Errorf("read expected manifest: %w", err)
	}

	var (
		actual             media.Manifest
		actualManifestPath = params.ActualManifest
	)
	switch {
	case params.ActualManifest != "":
		actual, err = readMediaManifest(params.ActualManifest)
	case params.Dir != "" || params.AgentURL != "" || params.SSHTarget != "":
		hash := expectedManifestHasHashes(expected) && !params.SkipHash
		actual, err = scanMediaSource(ctx, MediaScanParams{
			Dir:          params.Dir,
			Hash:         hash,
			AgentURL:     params.AgentURL,
			AgentToken:   params.AgentToken,
			AgentTimeout: params.AgentTimeout,
			SSHTarget:    params.SSHTarget,
			SSHPort:      params.SSHPort,
			SSHKey:       params.SSHKey,
		})
	default:
		return MediaVerifyResult{}, fmt.Errorf("actual manifest or verification target is required")
	}
	if err != nil {
		return MediaVerifyResult{}, fmt.Errorf("load verification target: %w", err)
	}

	report := media.Compare(expected, actual, expectedManifestHasHashes(expected) && !params.SkipHash)
	dest := params.ReportPath
	if dest == "" {
		dest = filepath.Join(params.OutDir, "media-verify.json")
	}
	if err := writeMediaVerifyReport(dest, report); err != nil {
		return MediaVerifyResult{}, fmt.Errorf("write verify report: %w", err)
	}

	return MediaVerifyResult{
		ExpectedManifestPath: params.FromManifest,
		ActualManifestPath:   actualManifestPath,
		Expected:             expected,
		Actual:               actual,
		Report:               report,
		Path:                 dest,
	}, nil
}

func readMediaManifest(path string) (media.Manifest, error) {
	f, err := os.Open(path)
	if err != nil {
		return media.Manifest{}, err
	}
	defer func() {
		_ = f.Close()
	}()

	var manifest media.Manifest
	if err := json.NewDecoder(f).Decode(&manifest); err != nil {
		return media.Manifest{}, err
	}
	return manifest, nil
}

func expectedManifestHasHashes(manifest media.Manifest) bool {
	for _, f := range manifest.Files {
		if f.SHA256 != "" {
			return true
		}
	}
	return false
}

func writeMediaVerifyReport(path string, report media.VerifyReport) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}
