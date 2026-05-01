package cli

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/sibukixxx/wp2emdash/internal/cli/output"
	"github.com/sibukixxx/wp2emdash/internal/usecase"
)

func newSecretsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "secrets",
		Short: "Secrets and credential readiness checks",
	}
	cmd.AddCommand(newSecretsCheckCmd())
	return cmd
}

func newSecretsCheckCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Check whether required secret env vars are present",
		Long: `secrets check verifies that the environment already contains the
credentials a migration phase is likely to need.

It does not create, edit, or overwrite .env files. It only reports whether
expected environment variables are present for the selected profile.`,
		RunE: runSecretsCheck,
	}
	cmd.Flags().String("profile", "small-production", "secret profile: small-production, seo-production, media-heavy, custom-rebuild, agent")
	return cmd
}

func runSecretsCheck(cmd *cobra.Command, _ []string) error {
	rep := usecase.RunSecretsCheck(cmd.Context(), mustString(cmd, "profile"))

	if mustBool(cmd, "json") {
		if err := output.JSON(cmd.OutOrStdout(), rep); err != nil {
			return err
		}
	} else {
		w := cmd.OutOrStdout()
		if err := output.Printf(w, "profile: %s\n", rep.Profile); err != nil {
			return err
		}
		for _, c := range rep.Checks {
			tag := "optional"
			if c.Required {
				tag = "required"
			}
			status := "missing"
			if c.Found {
				status = c.Source
			}
			if err := output.Printf(w, "  [%s] %-24s %s\n", tag, c.Name, status); err != nil {
				return err
			}
		}
		if rep.OK {
			if err := output.Println(w, "OK"); err != nil {
				return err
			}
		} else {
			if err := output.Println(w, "FAIL: required secret(s) missing"); err != nil {
				return err
			}
		}
	}

	if !rep.OK {
		return errors.New("required secret missing")
	}
	return nil
}
