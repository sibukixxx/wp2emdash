package usecase

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sibukixxx/wp2emdash/internal/domain/seo"
)

// SEOURLMapParams configures `wp2emdash seo url-map`.
type SEOURLMapParams struct {
	OldPath string
	NewPath string
	OutDir  string
	OutPath string // override; defaults to <OutDir>/seo-url-map.json
	Write   bool
	Version string
}

// SEOURLMapResult is the comparison output handed back to the CLI layer.
type SEOURLMapResult struct {
	Diff seo.URLMapDiff
	Path string
}

// RunSEOURLMap reads two URL map files and produces a structured diff.
//
// Each input file may be either a JSON document matching seo.URLMap, or a
// plain text file with one URL per line (lines starting with '#' or empty
// lines are skipped). The format is auto-detected from the file extension:
// .json => JSON, anything else => plain text.
func RunSEOURLMap(params SEOURLMapParams) (SEOURLMapResult, error) {
	if params.OldPath == "" || params.NewPath == "" {
		return SEOURLMapResult{}, errors.New("both --old and --new must be provided")
	}
	oldMap, err := loadURLMap(params.OldPath)
	if err != nil {
		return SEOURLMapResult{}, fmt.Errorf("load old map %s: %w", params.OldPath, err)
	}
	newMap, err := loadURLMap(params.NewPath)
	if err != nil {
		return SEOURLMapResult{}, fmt.Errorf("load new map %s: %w", params.NewPath, err)
	}

	diff := seo.DiffURLMaps(oldMap, newMap)
	diff.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
	diff.Tool = "wp2emdash"
	diff.Version = params.Version

	dest := params.OutPath
	if dest == "" {
		dest = filepath.Join(params.OutDir, "seo-url-map.json")
	}
	res := SEOURLMapResult{Diff: diff, Path: dest}
	if !params.Write {
		return res, nil
	}
	if err := writeJSON(dest, diff); err != nil {
		return SEOURLMapResult{}, fmt.Errorf("write url map diff: %w", err)
	}
	return res, nil
}

// loadURLMap auto-detects JSON vs plain-text based on file extension.
//
// Plain-text mode treats one URL per line as the entry list and uses the file
// stem as the Source label. The format is intentionally simple so users can
// pipe the output of `wp post list --field=url` directly into it.
func loadURLMap(path string) (seo.URLMap, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return seo.URLMap{}, err
	}
	if strings.EqualFold(filepath.Ext(path), ".json") {
		var m seo.URLMap
		if err := json.Unmarshal(body, &m); err != nil {
			return seo.URLMap{}, fmt.Errorf("decode JSON: %w", err)
		}
		return m, nil
	}
	return parseTextURLMap(path, string(body)), nil
}

func parseTextURLMap(path, body string) seo.URLMap {
	source := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	entries := make([]seo.URLEntry, 0)
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		entries = append(entries, seo.URLEntry{URL: line})
	}
	return seo.URLMap{Source: source, Entries: entries}
}
