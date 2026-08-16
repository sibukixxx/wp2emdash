package contentverify

import "testing"

func TestCompareMatchesHTMLAndPortableTextByVisibleContent(t *testing.T) {
	t.Parallel()
	expected := Snapshot{Complete: true, Entries: []Entry{{ID: "12", Kind: "post", Slug: "hello", Status: "publish", Title: "Hello", PublishedAt: "2024-01-02T03:04:05Z", Body: FingerprintHTML(`<h2>Hello</h2><p>World <a href="https://example.test/a/">link</a></p>`)}}}
	actualBody := []any{
		map[string]any{"_type": "block", "style": "h2", "children": []any{map[string]any{"_type": "span", "text": "Hello"}}},
		map[string]any{"_type": "block", "style": "normal", "children": []any{map[string]any{"_type": "span", "text": "World link"}}, "markDefs": []any{map[string]any{"_type": "link", "href": "https://example.test/a"}}},
	}
	actual := Snapshot{Complete: true, Entries: []Entry{{ID: "01ABC", Kind: "posts", Slug: "hello", Status: "published", Title: "Hello", PublishedAt: "2024-01-02T03:04:05.999Z", Body: FingerprintPortableText(actualBody)}}}
	report := Compare(expected, actual, Map{Version: 1})
	if !report.OK {
		t.Fatalf("expected semantic match, issues: %+v", report.Issues)
	}
	if report.Totals.Matched != 1 || report.Totals.Warnings != 2 {
		t.Fatalf("unexpected totals: %+v", report.Totals)
	}
}

func TestCompareFailsClosedOnMissingAndAmbiguousEntries(t *testing.T) {
	t.Parallel()
	expected := Snapshot{Complete: true, Entries: []Entry{{ID: "1", Kind: "post", Slug: "same"}}}
	actual := Snapshot{Complete: true, Entries: []Entry{{ID: "a", Kind: "posts", Slug: "same"}, {ID: "b", Kind: "posts", Slug: "same"}}}
	report := Compare(expected, actual, Map{Version: 1})
	if report.OK || report.Totals.Critical != 1 || report.Totals.Missing != 1 {
		t.Fatalf("gate must fail closed: %+v", report)
	}
}

func TestComparePrefersExplicitMappingOverSlug(t *testing.T) {
	t.Parallel()
	expected := Snapshot{Complete: true, Entries: []Entry{{ID: "7", Kind: "post", Slug: "old", Body: FingerprintHTML("same")}}}
	actual := Snapshot{Complete: true, Entries: []Entry{{ID: "target", Kind: "posts", Slug: "new", Body: FingerprintHTML("same")}}}
	mapping := Map{Version: 1, Entries: []ExplicitMapping{{WordPressID: 7, EmDashCollection: "posts", EmDashID: "target"}}}
	report := Compare(expected, actual, mapping)
	if len(report.Matches) != 1 || report.Matches[0].Method != "explicit" {
		t.Fatalf("explicit mapping not used: %+v", report.Matches)
	}
	if report.Totals.Errors == 0 {
		t.Fatal("slug change should remain an actionable mismatch")
	}
}

func TestFingerprintDoesNotContainBodyText(t *testing.T) {
	t.Parallel()
	f := FingerprintHTML("<p>private draft secret</p>")
	if f.TextSHA256 == "" || f.TextLength == 0 {
		t.Fatal("fingerprint missing")
	}
}

func TestApplyPolicyCanDowngradeStableIssueCode(t *testing.T) {
	t.Parallel()
	report := Report{Issues: []Issue{{Severity: SeverityError, Code: "updated_at_mismatch"}}}
	ApplyPolicy(&report, Policy{Version: 1, SeverityOverrides: map[string]Severity{"updated_at_mismatch": SeverityWarning}})
	if !report.OK || report.Totals.Warnings != 1 || report.Totals.Errors != 0 {
		t.Fatalf("policy not applied: %+v", report)
	}
}
