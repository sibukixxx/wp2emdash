package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rokubunnoni-inc/wp2emdash/internal/usecase"
)

func newDoctorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check that the host has the external tools wp2emdash needs",
		Long: `doctor inspects the local environment for the external tools wp2emdash
delegates to (wp-cli, wrangler, rclone, awscli, git, jq, php, node).

Each tool is reported as required or optional. Missing required tools cause
a non-zero exit code so this command is safe to use as a CI gate.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runDoctor(cmd)
		},
	}
	return cmd
}

func runDoctor(cmd *cobra.Command) error {
	rep := usecase.RunDoctor(cmd.Context())

	if mustBool(cmd, "json") {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		if err := enc.Encode(rep); err != nil {
			return err
		}
	} else {
		fmt.Fprintln(cmd.OutOrStdout(), "wp2emdash doctor")
		for _, c := range rep.Checks {
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
		if rep.OK {
			fmt.Fprintln(cmd.OutOrStdout(), "OK")
		} else {
			fmt.Fprintln(cmd.OutOrStdout(), "FAIL: required tool(s) missing")
		}
	}

	if !rep.OK {
		return fmt.Errorf("required tool missing")
	}
	return nil
}
