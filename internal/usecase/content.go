package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/sibukixxx/wp2emdash/internal/domain/contentverify"
	"github.com/sibukixxx/wp2emdash/internal/domain/source"
	"github.com/sibukixxx/wp2emdash/internal/infra/emdashcli"
	"github.com/sibukixxx/wp2emdash/internal/infra/wpcli"
	"github.com/sibukixxx/wp2emdash/internal/shell"
)

type ContentSnapshotParams struct {
	Source, WPRoot, EmDashURL, OutDir, OutPath, MapPath, Version, SSHTarget, SSHKey string
	SSHPort, Jobs                                                                   int
	Write                                                                           bool
}
type ContentSnapshotResult struct {
	Snapshot contentverify.Snapshot
	Path     string
}

func RunContentSnapshot(ctx context.Context, p ContentSnapshotParams) (ContentSnapshotResult, error) {
	m, err := LoadContentMap(p.MapPath)
	if err != nil {
		return ContentSnapshotResult{}, err
	}
	var snapper source.ContentSnapshotter
	switch p.Source {
	case "wordpress":
		if p.SSHTarget != "" {
			snapper, err = wpcli.NewRemoteAuditor(wpcli.RemoteConfig{Target: p.SSHTarget, Port: p.SSHPort, Key: p.SSHKey, WPRoot: p.WPRoot})
		} else {
			snapper, err = wpcli.NewAuditor(p.WPRoot)
		}
	case "emdash":
		snapper = emdashcli.Snapshotter{URL: p.EmDashURL, Jobs: p.Jobs, Version: p.Version, Runner: shell.Runner{}}
	default:
		return ContentSnapshotResult{}, fmt.Errorf("unknown content snapshot source %q", p.Source)
	}
	if err != nil {
		return ContentSnapshotResult{}, err
	}
	snap, err := snapper.Snapshot(ctx, m)
	if err != nil {
		return ContentSnapshotResult{}, fmt.Errorf("snapshot %s: %w", p.Source, err)
	}
	snap.Version = p.Version
	dest := p.OutPath
	if dest == "" {
		dest = filepath.Join(p.OutDir, "content-"+p.Source+".json")
	}
	res := ContentSnapshotResult{Snapshot: snap, Path: dest}
	if p.Write {
		if err := writeContentJSON(dest, snap); err != nil {
			return ContentSnapshotResult{}, fmt.Errorf("write content snapshot: %w", err)
		}
	}
	return res, nil
}

type ContentVerifyParams struct {
	ExpectedPath, ActualPath, MapPath, PolicyPath, OutDir, ReportPath, MarkdownPath, ResolvedMapPath, Version string
	Write                                                                                                     bool
}
type ContentVerifyResult struct {
	Report                                  contentverify.Report
	JSONPath, MarkdownPath, ResolvedMapPath string
}

func RunContentVerify(p ContentVerifyParams) (ContentVerifyResult, error) {
	var expected, actual contentverify.Snapshot
	if err := readJSON(p.ExpectedPath, &expected); err != nil {
		return ContentVerifyResult{}, fmt.Errorf("read expected snapshot: %w", err)
	}
	if err := readJSON(p.ActualPath, &actual); err != nil {
		return ContentVerifyResult{}, fmt.Errorf("read actual snapshot: %w", err)
	}
	m, err := LoadContentMap(p.MapPath)
	if err != nil {
		return ContentVerifyResult{}, err
	}
	report := contentverify.Compare(expected, actual, m)
	policy := contentverify.Policy{Version: 1}
	if p.PolicyPath != "" {
		if err := readJSON(p.PolicyPath, &policy); err != nil {
			return ContentVerifyResult{}, fmt.Errorf("read content verify policy: %w", err)
		}
	}
	contentverify.ApplyPolicy(&report, policy)
	report.Tool = "wp2emdash"
	report.Version = p.Version
	report.GeneratedAt = nowUTC()
	jsonPath := p.ReportPath
	if jsonPath == "" {
		jsonPath = filepath.Join(p.OutDir, "content-verify.json")
	}
	mdPath := p.MarkdownPath
	if mdPath == "" {
		mdPath = filepath.Join(p.OutDir, "content-verify.md")
	}
	mapPath := p.ResolvedMapPath
	if mapPath == "" {
		mapPath = filepath.Join(p.OutDir, "content-resolved-map.json")
	}
	res := ContentVerifyResult{Report: report, JSONPath: jsonPath, MarkdownPath: mdPath, ResolvedMapPath: mapPath}
	if p.Write {
		if err := writeContentJSON(jsonPath, report); err != nil {
			return ContentVerifyResult{}, err
		}
		if err := writeContentMarkdown(mdPath, report); err != nil {
			return ContentVerifyResult{}, err
		}
		resolved := resolvedMap(m, report)
		if err := writeContentJSON(mapPath, resolved); err != nil {
			return ContentVerifyResult{}, err
		}
	}
	return res, nil
}

func LoadContentMap(path string) (contentverify.Map, error) {
	m := contentverify.Map{Version: 1}
	if path == "" {
		return m, nil
	}
	if err := readJSON(path, &m); err != nil {
		return m, fmt.Errorf("read content map: %w", err)
	}
	if err := contentverify.ValidateMap(m); err != nil {
		return m, err
	}
	return m, nil
}
func resolvedMap(base contentverify.Map, r contentverify.Report) contentverify.Map {
	out := base
	seen := map[string]bool{}
	for _, e := range out.Entries {
		seen[fmt.Sprintf("%d\x00%s", e.WordPressID, e.EmDashCollection)] = true
	}
	for _, match := range r.Matches {
		id, err := strconv.Atoi(match.SourceID)
		if err != nil {
			continue
		}
		target := targetKind(base, match.Kind)
		k := fmt.Sprintf("%d\x00%s", id, target)
		if seen[k] {
			continue
		}
		out.Entries = append(out.Entries, contentverify.ExplicitMapping{WordPressID: id, EmDashCollection: target, EmDashID: match.TargetID, Locale: match.Locale})
		seen[k] = true
	}
	sort.Slice(out.Entries, func(i, j int) bool { return out.Entries[i].WordPressID < out.Entries[j].WordPressID })
	return out
}
func targetKind(m contentverify.Map, kind string) string {
	for _, cm := range m.Collections {
		if cm.WordPressPostType == kind {
			return cm.EmDashCollection
		}
	}
	if kind == "post" {
		return "posts"
	}
	if kind == "page" {
		return "pages"
	}
	return kind
}
func readJSON(path string, v any) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	return json.NewDecoder(f).Decode(v)
}
func writeContentJSON(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
func writeContentMarkdown(path string, r contentverify.Report) error {
	var b strings.Builder
	status := "PASS"
	if !r.OK {
		status = "FAIL"
	}
	fmt.Fprintf(&b, "# Content migration verification\n\n**Gate: %s**\n\n", status)
	fmt.Fprintf(&b, "- Expected: %d\n- Actual: %d\n- Matched: %d\n- Missing: %d\n- Extra: %d\n- Critical: %d\n- Errors: %d\n- Warnings: %d\n\n", r.Totals.Expected, r.Totals.Actual, r.Totals.Matched, r.Totals.Missing, r.Totals.Extra, r.Totals.Critical, r.Totals.Errors, r.Totals.Warnings)
	if len(r.Issues) > 0 {
		b.WriteString("## Issues\n\n| Severity | Code | Source | Target | Slug |\n| --- | --- | --- | --- | --- |\n")
		for _, i := range r.Issues {
			fmt.Fprintf(&b, "| %s | `%s` | %s | %s | %s |\n", i.Severity, i.Code, i.SourceID, i.TargetID, i.Slug)
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}
func nowUTC() string { return time.Now().UTC().Format(time.RFC3339) }
