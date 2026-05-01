package usecase

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/sibukixxx/wp2emdash/internal/domain/seo"
	"github.com/sibukixxx/wp2emdash/internal/domain/source"
	"github.com/sibukixxx/wp2emdash/internal/infra/wpcli"
)

// SEORedirectsParams configures `wp2emdash seo extract-redirects`.
type SEORedirectsParams struct {
	WPRoot    string
	OutDir    string
	Write     bool
	Version   string
	OutPath   string // override; defaults to <OutDir>/seo-redirects.json
	SSHTarget string
	SSHPort   int
	SSHKey    string
}

// SEORedirectsResult is the output handed back to the CLI layer.
type SEORedirectsResult struct {
	Set  seo.RedirectSet
	Path string
}

// RunSEOExtractRedirects builds the source adapter and delegates to
// RunSEOExtractRedirectsFromSource.
func RunSEOExtractRedirects(ctx context.Context, params SEORedirectsParams) (SEORedirectsResult, error) {
	src, err := buildRedirectExtractor(params)
	if err != nil {
		return SEORedirectsResult{}, err
	}
	return RunSEOExtractRedirectsFromSource(ctx, src, params)
}

// RunSEOExtractRedirectsFromSource accepts a pre-built RedirectExtractor.
func RunSEOExtractRedirectsFromSource(ctx context.Context, src source.RedirectExtractor, params SEORedirectsParams) (SEORedirectsResult, error) {
	rules, err := src.ExtractRedirects(ctx)
	if err != nil {
		return SEORedirectsResult{}, fmt.Errorf("extract redirects: %w", err)
	}
	if rules == nil {
		rules = []seo.RedirectRule{}
	}
	set := seo.RedirectSet{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Tool:        "wp2emdash",
		Version:     params.Version,
		Rules:       rules,
	}
	dest := params.OutPath
	if dest == "" {
		dest = filepath.Join(params.OutDir, "seo-redirects.json")
	}
	res := SEORedirectsResult{Set: set, Path: dest}
	if !params.Write {
		return res, nil
	}
	if err := writeJSON(dest, set); err != nil {
		return SEORedirectsResult{}, fmt.Errorf("write seo redirects: %w", err)
	}
	return res, nil
}

func buildRedirectExtractor(params SEORedirectsParams) (source.RedirectExtractor, error) {
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
