package tests

import (
	"strings"
	"testing"

	"github.com/sibukixxx/wp2emdash/test/e2e"
)

func TestSecretsCheckCommand_SucceedsWhenRequiredSecretsExist(t *testing.T) {
	t.Parallel()

	cli := e2e.NewCLI(t)
	res := cli.RunWithEnv(t, []string{
		"CLOUDFLARE_API_TOKEN=token",
		"CLOUDFLARE_ACCOUNT_ID=acct",
	}, "secrets", "check", "--profile", "small-production")

	if res.Err != nil {
		t.Fatalf("command failed: %v\nstdout:\n%s\nstderr:\n%s", res.Err, res.Stdout, res.Stderr)
	}
	if !strings.Contains(res.Stdout, "OK") {
		t.Fatalf("stdout missing OK:\n%s", res.Stdout)
	}
}

func TestSecretsCheckCommand_FailsWhenRequiredSecretsMissing(t *testing.T) {
	t.Parallel()

	cli := e2e.NewCLI(t)
	res := cli.RunWithEnv(t, nil, "secrets", "check", "--profile", "media-heavy")

	if res.Err == nil {
		t.Fatalf("expected error, got success\nstdout:\n%s", res.Stdout)
	}
	if !strings.Contains(res.Stdout, "FAIL: required secret(s) missing") {
		t.Fatalf("stdout missing failure summary:\n%s", res.Stdout)
	}
}
