package cli

import (
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/sibukixxx/wp2emdash/internal/cli/output"
	"github.com/sibukixxx/wp2emdash/internal/usecase"
)

func newAuditCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "audit",
		Short: "Quantify WordPress complexity for an EmDash migration",
		Long: `audit shells out to wp-cli on the target WordPress install and
collects ~14 metrics (content / uploads / theme / plugins / mu-plugins /
postmeta / shortcode / SEO / redirects / external integrations).

It also runs the score rubric and emits both a JSON summary and a
Markdown report under --out (default: wp2emdash-output/).

Run this on the WordPress server (or anywhere with wp-cli access to it).`,
		RunE: runAuditCmd,
	}
	cmd.Flags().String("wp-root", ".", "WordPress install root (directory containing wp-config.php)")
	cmd.Flags().Bool("write", true, "write summary.json + risk-report.md to --out")
	cmd.Flags().String("risk-bands", "", "path to a JSON file that defines public risk bands and estimates")
	cmd.Flags().String("agent-url", "", "HTTP endpoint for a read-only audit agent")
	cmd.Flags().String("agent-token", "", "bearer token for --agent-url")
	cmd.Flags().Duration("agent-timeout", 30*time.Second, "HTTP timeout for --agent-url")
	cmd.Flags().String("ssh", "", "SSH target for remote audit execution (example: user@example.com)")
	cmd.Flags().Int("ssh-port", 22, "SSH port for --ssh")
	cmd.Flags().String("ssh-key", "", "SSH private key path for --ssh")
	return cmd
}

func runAuditCmd(cmd *cobra.Command, _ []string) error {
	wpRoot := mustString(cmd, "wp-root")
	outDir := mustString(cmd, "out")
	emitJSON := mustBool(cmd, "json")
	write := mustBool(cmd, "write")

	res, err := usecase.RunAudit(cmd.Context(), usecase.AuditParams{
		WPRoot:        wpRoot,
		OutDir:        outDir,
		Write:         write,
		Version:       Version,
		RiskBandsPath: mustString(cmd, "risk-bands"),
		AgentURL:      mustString(cmd, "agent-url"),
		AgentToken:    mustString(cmd, "agent-token"),
		AgentTimeout: func() time.Duration {
			v, _ := cmd.Flags().GetDuration("agent-timeout")
			return v
		}(),
		SSHTarget: mustString(cmd, "ssh"),
		SSHPort: func() int {
			port, _ := cmd.Flags().GetInt("ssh-port")
			return port
		}(),
		SSHKey: mustString(cmd, "ssh-key"),
	})
	if err != nil {
		return err
	}

	w := cmd.OutOrStdout()
	if write && !emitJSON {
		abs, _ := filepath.Abs(outDir)
		if err := output.Printf(w, "wrote %s/summary.json\n", abs); err != nil {
			return err
		}
		if err := output.Printf(w, "wrote %s/risk-report.md\n", abs); err != nil {
			return err
		}
	}

	if emitJSON {
		return output.JSON(w, res.Bundle)
	}

	a := res.Bundle.Audit
	s := res.Bundle.Score
	if err := output.Printf(w, "Risk score: %d (%s) — %s\n", s.Score, s.Level, s.Estimate); err != nil {
		return err
	}
	if err := output.Printf(
		w,
		"Posts: %d, Pages: %d, Active plugins: %d, Active theme: %s\n",
		a.Content.Posts, a.Content.Pages, a.Plugins.ActiveCount, a.Theme.ActiveTheme,
	); err != nil {
		return err
	}

	if len(res.Bundle.Warnings) == 0 {
		return nil
	}

	if err := output.Printf(w, "Audit warnings: %d\n", len(res.Bundle.Warnings)); err != nil {
		return err
	}
	for i, warning := range res.Bundle.Warnings {
		if i >= 3 {
			break
		}
		if err := output.Printf(w, "  - %s: %s\n", warning.Code, warning.Message); err != nil {
			return err
		}
	}
	if len(res.Bundle.Warnings) > 3 {
		if err := output.Printf(w, "  - ...and %d more (see summary.json or risk-report.md)\n", len(res.Bundle.Warnings)-3); err != nil {
			return err
		}
	}
	return nil
}
