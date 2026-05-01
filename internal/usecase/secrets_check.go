package usecase

import (
	"context"
	"fmt"
)

type SecretCheck struct {
	Name      string `json:"name"`
	Required  bool   `json:"required"`
	Found     bool   `json:"found"`
	Source    string `json:"source,omitempty"`
	Hint      string `json:"hint,omitempty"`
	IssueText string `json:"issue,omitempty"`
}

type SecretsReport struct {
	Profile string        `json:"profile"`
	OK      bool          `json:"ok"`
	Checks  []SecretCheck `json:"checks"`
}

type getenvFunc func(string) string

func RunSecretsCheck(_ context.Context, profile string) SecretsReport {
	return RunSecretsCheckWithGetenv(profile, getenv)
}

func RunSecretsCheckWithGetenv(profile string, getenv getenvFunc) SecretsReport {
	checks, err := checksForSecretsProfile(profile)
	if err != nil {
		return SecretsReport{
			Profile: profile,
			OK:      false,
			Checks: []SecretCheck{{
				Name:      "profile",
				Required:  true,
				IssueText: err.Error(),
			}},
		}
	}

	report := SecretsReport{
		Profile: profile,
		OK:      true,
		Checks:  make([]SecretCheck, 0, len(checks)),
	}
	for _, spec := range checks {
		found := getenv(spec.Env) != ""
		check := SecretCheck{
			Name:     spec.Env,
			Required: spec.Required,
			Found:    found,
			Hint:     spec.Hint,
		}
		if found {
			check.Source = "env"
		} else if spec.Required {
			check.IssueText = fmt.Sprintf("%s is not set", spec.Env)
			report.OK = false
		}
		report.Checks = append(report.Checks, check)
	}
	return report
}

type secretSpec struct {
	Env      string
	Required bool
	Hint     string
}

func checksForSecretsProfile(profile string) ([]secretSpec, error) {
	switch profile {
	case "small-production", "seo-production", "custom-rebuild":
		return []secretSpec{
			{Env: "CLOUDFLARE_API_TOKEN", Required: true, Hint: "Cloudflare API token for wrangler deploy / DNS / Rules changes"},
			{Env: "CLOUDFLARE_ACCOUNT_ID", Required: true, Hint: "Cloudflare account ID for wrangler / R2 operations"},
			{Env: "WP2EMDASH_AGENT_TOKEN", Hint: "Only needed when you use --agent-url instead of local wp-cli / SSH"},
			{Env: "SSH_AUTH_SOCK", Hint: "Optional, but useful when the migration plan relies on SSH execution"},
		}, nil
	case "media-heavy":
		return []secretSpec{
			{Env: "CLOUDFLARE_API_TOKEN", Required: true, Hint: "Cloudflare API token for wrangler deploy / DNS / Rules changes"},
			{Env: "CLOUDFLARE_ACCOUNT_ID", Required: true, Hint: "Cloudflare account ID for wrangler / R2 operations"},
			{Env: "AWS_ACCESS_KEY_ID", Required: true, Hint: "R2 / S3 compatible access key for media sync / verify"},
			{Env: "AWS_SECRET_ACCESS_KEY", Required: true, Hint: "R2 / S3 compatible secret key for media sync / verify"},
			{Env: "AWS_ENDPOINT_URL_S3", Hint: "Optional endpoint override when the media sync target is S3-compatible"},
			{Env: "RCLONE_CONFIG", Hint: "Optional path to a pre-provisioned rclone config"},
			{Env: "WP2EMDASH_AGENT_TOKEN", Hint: "Only needed when you use --agent-url instead of local wp-cli / SSH"},
			{Env: "SSH_AUTH_SOCK", Hint: "Optional, but useful when the migration plan relies on SSH execution"},
		}, nil
	case "agent":
		return []secretSpec{
			{Env: "WP2EMDASH_AGENT_TOKEN", Required: true, Hint: "Bearer token for the read-only WordPress HTTP agent"},
		}, nil
	default:
		return nil, fmt.Errorf("unknown secrets profile %q", profile)
	}
}

func getenv(key string) string {
	return getenvOS(key)
}
