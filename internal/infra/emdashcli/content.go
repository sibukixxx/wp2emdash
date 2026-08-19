// Package emdashcli thinly wraps the official EmDash CLI for read-only
// destination snapshots.
package emdashcli

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/sibukixxx/wp2emdash/internal/domain/contentverify"
	"github.com/sibukixxx/wp2emdash/internal/shell"
)

type Snapshotter struct {
	URL     string
	Jobs    int
	Version string
	Runner  shell.Runner
}

type listResult struct {
	Items      []struct{ ID, Slug, Locale, Status, Title, UpdatedAt string } `json:"items"`
	NextCursor string                                                        `json:"nextCursor"`
}
type getResult struct {
	ID, Slug, Locale, Status, CreatedAt, UpdatedAt, PublishedAt string
	Data                                                        map[string]any `json:"data"`
}

func (s Snapshotter) Snapshot(ctx context.Context, mapping contentverify.Map) (contentverify.Snapshot, error) {
	jobs := s.Jobs
	if jobs < 1 {
		jobs = 4
	}
	collections := targetCollections(mapping)
	out := contentverify.Snapshot{SchemaVersion: 1, GeneratedAt: time.Now().UTC().Format(time.RFC3339), Tool: "wp2emdash", Version: s.Version, Source: "emdash", SiteURL: s.URL, Complete: true}
	for _, collection := range collections {
		refs, err := s.listAll(ctx, collection)
		if err != nil {
			return contentverify.Snapshot{}, err
		}
		entries := make([]contentverify.Entry, len(refs))
		errs := make(chan error, len(refs))
		sem := make(chan struct{}, jobs)
		var wg sync.WaitGroup
		for i, ref := range refs {
			i, ref := i, ref
			wg.Add(1)
			go func() {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				e, err := s.get(ctx, collection, ref.ID, ref.Status)
				if err != nil {
					errs <- err
					return
				}
				entries[i] = e
			}()
		}
		wg.Wait()
		close(errs)
		if err := <-errs; err != nil {
			return contentverify.Snapshot{}, err
		}
		out.Entries = append(out.Entries, entries...)
	}
	return out, nil
}

func (s Snapshotter) listAll(ctx context.Context, collection string) ([]struct{ ID, Slug, Locale, Status, Title, UpdatedAt string }, error) {
	var all []struct{ ID, Slug, Locale, Status, Title, UpdatedAt string }
	cursor := ""
	for {
		args := []string{"content", "list", collection, "--limit", "100", "--url", s.URL, "--json"}
		if cursor != "" {
			args = append(args, "--cursor", cursor)
		}
		raw, err := s.Runner.Output(ctx, "emdash", args...)
		if err != nil {
			return nil, fmt.Errorf("emdash content list %s: %w", collection, err)
		}
		var page listResult
		if err := json.Unmarshal([]byte(raw), &page); err != nil {
			return nil, fmt.Errorf("decode emdash content list %s: %w", collection, err)
		}
		all = append(all, page.Items...)
		if page.NextCursor == "" {
			break
		}
		cursor = page.NextCursor
	}
	return all, nil
}

func (s Snapshotter) get(ctx context.Context, collection, id, status string) (contentverify.Entry, error) {
	args := []string{"content", "get", collection, id, "--raw"}
	if status == "published" {
		args = append(args, "--published")
	}
	args = append(args, "--url", s.URL, "--json")
	raw, err := s.Runner.Output(ctx, "emdash", args...)
	if err != nil {
		return contentverify.Entry{}, fmt.Errorf("emdash content get %s/%s: %w", collection, id, err)
	}
	var item getResult
	if err := json.Unmarshal([]byte(raw), &item); err != nil {
		return contentverify.Entry{}, fmt.Errorf("decode emdash content get %s/%s: %w", collection, id, err)
	}
	title, _ := item.Data["title"].(string)
	excerpt, _ := item.Data["excerpt"].(string)
	content := item.Data["content"]
	fields := map[string]string{}
	for k, v := range item.Data {
		if k == "content" || k == "title" || k == "excerpt" {
			continue
		}
		b, _ := json.Marshal(v)
		fields[k] = contentverify.HashString(string(b))
	}
	return contentverify.Entry{ID: item.ID, Kind: collection, Slug: item.Slug, Locale: item.Locale, Status: item.Status, Title: title, ExcerptHash: contentverify.HashString(excerpt), CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt, PublishedAt: item.PublishedAt, Body: contentverify.FingerprintPortableText(content), Fields: fields}, nil
}

func targetCollections(m contentverify.Map) []string {
	seen := map[string]bool{"posts": true, "pages": true}
	for _, cm := range m.Collections {
		seen[cm.EmDashCollection] = true
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		if k != "" {
			out = append(out, k)
		}
	}
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j] < out[i] {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}
