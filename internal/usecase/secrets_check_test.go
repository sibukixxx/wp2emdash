package usecase

import "testing"

func TestRunSecretsCheckWithGetenv_SmallProduction(t *testing.T) {
	t.Parallel()

	env := map[string]string{
		"CLOUDFLARE_API_TOKEN":  "token",
		"CLOUDFLARE_ACCOUNT_ID": "acct",
	}
	report := RunSecretsCheckWithGetenv("small-production", func(key string) string {
		return env[key]
	})

	if !report.OK {
		t.Fatalf("report.OK: want true, got false")
	}
	if len(report.Checks) == 0 {
		t.Fatal("checks: want entries, got 0")
	}
}

func TestRunSecretsCheckWithGetenv_MediaHeavyMissingRequired(t *testing.T) {
	t.Parallel()

	report := RunSecretsCheckWithGetenv("media-heavy", func(string) string {
		return ""
	})

	if report.OK {
		t.Fatal("report.OK: want false, got true")
	}
	requiredMissing := 0
	for _, check := range report.Checks {
		if check.Required && !check.Found {
			requiredMissing++
		}
	}
	if requiredMissing < 4 {
		t.Fatalf("required missing: want >= 4, got %d", requiredMissing)
	}
}

func TestRunSecretsCheckWithGetenv_UnknownProfile(t *testing.T) {
	t.Parallel()

	report := RunSecretsCheckWithGetenv("unknown", func(string) string {
		return ""
	})

	if report.OK {
		t.Fatal("report.OK: want false, got true")
	}
	if len(report.Checks) != 1 {
		t.Fatalf("checks: want 1, got %d", len(report.Checks))
	}
}
