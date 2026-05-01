package wpcli

import (
	"testing"
)

func TestShellCommandQuotesArguments(t *testing.T) {
	t.Parallel()

	got := shellCommand("wp", "option", "get", "siteurl", "foo'bar")
	want := "'wp' 'option' 'get' 'siteurl' 'foo'\"'\"'bar'"
	if got != want {
		t.Fatalf("shellCommand() = %q, want %q", got, want)
	}
}
