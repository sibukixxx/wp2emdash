package seo_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/sibukixxx/wp2emdash/internal/domain/seo"
)

func TestParseHtaccessRedirectsExtractsRedirectDirective(t *testing.T) {
	tests := []struct {
		name string
		body string
		want []seo.RedirectRule
	}{
		{
			name: "Redirect with default 302",
			body: `Redirect /old /new`,
			want: []seo.RedirectRule{
				{From: "/old", To: "/new", Code: 302, Match: "exact", Source: "htaccess"},
			},
		},
		{
			name: "Redirect with explicit 301",
			body: `Redirect 301 /old /new`,
			want: []seo.RedirectRule{
				{From: "/old", To: "/new", Code: 301, Match: "exact", Source: "htaccess"},
			},
		},
		{
			name: "Redirect permanent maps to 301",
			body: `Redirect permanent /old /new`,
			want: []seo.RedirectRule{
				{From: "/old", To: "/new", Code: 301, Match: "exact", Source: "htaccess"},
			},
		},
		{
			name: "Redirect temp maps to 302",
			body: `Redirect temp /old /new`,
			want: []seo.RedirectRule{
				{From: "/old", To: "/new", Code: 302, Match: "exact", Source: "htaccess"},
			},
		},
		{
			name: "RedirectMatch regex with explicit 301",
			body: `RedirectMatch 301 ^/old/(.*)$ /new/$1`,
			want: []seo.RedirectRule{
				{From: "^/old/(.*)$", To: "/new/$1", Code: 301, Match: "regex", Source: "htaccess"},
			},
		},
		{
			name: "RewriteRule with R=301",
			body: `RewriteRule ^old$ /new [R=301,L]`,
			want: []seo.RedirectRule{
				{From: "^old$", To: "/new", Code: 301, Match: "regex", Source: "htaccess"},
			},
		},
		{
			name: "RewriteRule with R only defaults to 302",
			body: `RewriteRule ^old$ /new [R,L]`,
			want: []seo.RedirectRule{
				{From: "^old$", To: "/new", Code: 302, Match: "regex", Source: "htaccess"},
			},
		},
		{
			name: "RewriteRule without R flag is ignored",
			body: `RewriteRule ^old$ /new [L]`,
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := seo.ParseHtaccessRedirects(strings.NewReader(tt.body))
			if !equalRules(got, tt.want) {
				t.Errorf("got %+v want %+v", got, tt.want)
			}
		})
	}
}

func TestParseHtaccessRedirectsSkipsCommentsAndBlankLines(t *testing.T) {
	body := `
# This is a comment
   # indented comment

Redirect 301 /a /b

# trailing comment
RewriteRule ^x$ /y [R=301,L]
`
	got := seo.ParseHtaccessRedirects(strings.NewReader(body))
	want := []seo.RedirectRule{
		{From: "/a", To: "/b", Code: 301, Match: "exact", Source: "htaccess"},
		{From: "^x$", To: "/y", Code: 301, Match: "regex", Source: "htaccess"},
	}
	if !equalRules(got, want) {
		t.Errorf("got %+v want %+v", got, want)
	}
}

func TestParseHtaccessRedirectsRecordsLineNumbersInNote(t *testing.T) {
	body := "# header\n\nRedirect 301 /a /b\n"
	got := seo.ParseHtaccessRedirects(strings.NewReader(body))
	if len(got) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(got))
	}
	if got[0].Note != "line 3" {
		t.Errorf("expected note to record line 3, got %q", got[0].Note)
	}
}

func TestParseHtaccessRedirectsIgnoresMalformedDirectives(t *testing.T) {
	body := `
Redirect 301
Redirect /onlyone
RewriteRule
RedirectMatch 301
`
	got := seo.ParseHtaccessRedirects(strings.NewReader(body))
	if len(got) != 0 {
		t.Errorf("malformed lines should yield no rules, got %+v", got)
	}
}

func equalRules(a, b []seo.RedirectRule) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	stripped := func(rules []seo.RedirectRule) []seo.RedirectRule {
		out := make([]seo.RedirectRule, len(rules))
		for i, r := range rules {
			r.Note = ""
			out[i] = r
		}
		return out
	}
	return reflect.DeepEqual(stripped(a), stripped(b))
}
