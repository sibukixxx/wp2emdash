package usecase

import (
	"context"
	"os/exec"
)

type DoctorCheck struct {
	Name      string `json:"name"`
	Required  bool   `json:"required"`
	Found     bool   `json:"found"`
	Path      string `json:"path,omitempty"`
	Hint      string `json:"hint,omitempty"`
	IssueText string `json:"issue,omitempty"`
}

type DoctorReport struct {
	OK     bool          `json:"ok"`
	Checks []DoctorCheck `json:"checks"`
}

func RunDoctor(ctx context.Context) DoctorReport {
	required := []string{"wp", "wrangler", "git"}
	optional := []string{"php", "node", "pnpm", "rclone", "aws", "jq"}

	report := DoctorReport{OK: true}
	for _, tool := range required {
		report.Checks = append(report.Checks, checkTool(ctx, tool, true))
	}
	for _, tool := range optional {
		report.Checks = append(report.Checks, checkTool(ctx, tool, false))
	}
	for _, c := range report.Checks {
		if c.Required && !c.Found {
			report.OK = false
		}
	}
	return report
}

func checkTool(_ context.Context, tool string, required bool) DoctorCheck {
	c := DoctorCheck{Name: tool, Required: required}
	if path, err := exec.LookPath(tool); err == nil {
		c.Found = true
		c.Path = path
		return c
	}

	switch tool {
	case "wp":
		c.Hint = "https://wp-cli.org/#installing"
	case "wrangler":
		c.Hint = "npm install -g wrangler"
	case "rclone":
		c.Hint = "https://rclone.org/install/"
	}
	return c
}
