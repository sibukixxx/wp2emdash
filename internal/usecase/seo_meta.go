package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sibukixxx/wp2emdash/internal/domain/seo"
	"github.com/sibukixxx/wp2emdash/internal/domain/source"
	"github.com/sibukixxx/wp2emdash/internal/infra/wpcli"
)

// SEOMetaParams configures `wp2emdash seo extract-meta`.
type SEOMetaParams struct {
	WPRoot    string
	OutDir    string
	Write     bool
	Version   string
	OutPath   string // override; defaults to <OutDir>/seo-meta.json
	SSHTarget string
	SSHPort   int
	SSHKey    string
}

// SEOMetaResult is what the CLI prints / further pipes consume.
type SEOMetaResult struct {
	Set  seo.MetaSet
	Path string
}

// RunSEOExtractMeta is the standard entry point. It picks the appropriate
// source adapter based on params (local wp-cli or SSH) and delegates to
// RunSEOExtractMetaFromSource.
func RunSEOExtractMeta(ctx context.Context, params SEOMetaParams) (SEOMetaResult, error) {
	src, err := buildMetaExtractor(params)
	if err != nil {
		return SEOMetaResult{}, err
	}
	return RunSEOExtractMetaFromSource(ctx, src, params)
}

// RunSEOExtractMetaFromSource accepts a pre-built MetaExtractor — useful for
// tests and for non-WordPress sources implementing the same interface.
func RunSEOExtractMetaFromSource(ctx context.Context, src source.MetaExtractor, params SEOMetaParams) (SEOMetaResult, error) {
	items, err := src.ExtractMeta(ctx)
	if err != nil {
		return SEOMetaResult{}, fmt.Errorf("extract seo meta: %w", err)
	}
	set := seo.MetaSet{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Tool:        "wp2emdash",
		Version:     params.Version,
		Items:       items,
	}
	dest := params.OutPath
	if dest == "" {
		dest = filepath.Join(params.OutDir, "seo-meta.json")
	}
	res := SEOMetaResult{Set: set, Path: dest}
	if !params.Write {
		return res, nil
	}
	if err := writeJSON(dest, set); err != nil {
		return SEOMetaResult{}, fmt.Errorf("write seo meta: %w", err)
	}
	return res, nil
}

func buildMetaExtractor(params SEOMetaParams) (source.MetaExtractor, error) {
	if params.SSHTarget != "" {
		return wpcli.NewRemoteAuditor(wpcli.RemoteConfig{
			Target: params.SSHTarget,
			Port:   params.SSHPort,
			Key:    params.SSHKey,
			WPRoot: params.WPRoot,
		})
	}
	return wpcli.NewAuditor(params.WPRoot)
}

func writeJSON(path string, v any) error {
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
	return enc.Encode(v)
}
