package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"
)

type doctorCheck struct {
	Name      string `json:"name"`
	Required  bool   `json:"required"`
	Found     bool   `json:"found"`
	Path      string `json:"path,omitempty"`
	Hint      string `json:"hint,omitempty"`
	IssueText string `json:"issue,omitempty"`
}

type doctorReport struct {
	OK     bool          `json:"ok"`
	Checks []doctorCheck `json:"checks"`
}

func newDoctorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check that the host has the external tools wp2emdash needs",
		Long: `doctor inspects the local environment for the external tools wp2emdash
delegates to (wp-cli, wrangler, rclone, awscli, git, jq, php, node).

Each tool is reported as required or optional. Missing required tools cause
a non-zero exit code so this command is safe to use as a CI gate.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDoctor(cmd.Context(), cmd)
		},
	}
	return cmd
}

func runDoctor(ctx context.Context, cmd *cobra.Command) error {
	required := []string{"wp", "wrangler", "git"}
	optional := []string{"php", "node", "pnpm", "rclone", "aws", "jq"}

	report := doctorReport{OK: true}

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

	if mustBool(cmd, "json") {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			return err
		}
	} else {
		fmt.Fprintln(cmd.OutOrStdout(), "wp2emdash doctor")
		for _, c := range report.Checks {
			tag := "optional"
			if c.Required {
				tag = "required"
			}
			status := "missing"
			if c.Found {
				status = c.Path
			}
			fmt.Fprintf(cmd.OutOrStdout(), "  [%s] %-10s %s\n", tag, c.Name, status)
		}
		if report.OK {
			fmt.Fprintln(cmd.OutOrStdout(), "OK")
		} else {
			fmt.Fprintln(cmd.OutOrStdout(), "FAIL: required tool(s) missing")
		}
	}

	if !report.OK {
		return fmt.Errorf("required tool missing")
	}
	return nil
}

func checkTool(_ context.Context, tool string, required bool) doctorCheck {
	c := doctorCheck{Name: tool, Required: required}
	if path, err := exec.LookPath(tool); err == nil {
		c.Found = true
		c.Path = path
	} else {
		switch tool {
		case "wp":
			c.Hint = "https://wp-cli.org/#installing"
		case "wrangler":
			c.Hint = "npm install -g wrangler"
		case "rclone":
			c.Hint = "https://rclone.org/install/"
		}
	}
	return c
}
