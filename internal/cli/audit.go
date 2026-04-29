package cli

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/rokubunnoni-inc/wp2emdash/internal/usecase"
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
	return cmd
}

func runAuditCmd(cmd *cobra.Command, _ []string) error {
	wpRoot := mustString(cmd, "wp-root")
	outDir := mustString(cmd, "out")
	emitJSON := mustBool(cmd, "json")
	write := mustBool(cmd, "write")

	res, err := usecase.RunAudit(cmd.Context(), usecase.AuditParams{
		WPRoot:  wpRoot,
		OutDir:  outDir,
		Write:   write,
		Version: Version,
	})
	if err != nil {
		return err
	}

	if write && !emitJSON {
		abs, _ := filepath.Abs(outDir)
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "wrote %s/summary.json\n", abs); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "wrote %s/risk-report.md\n", abs); err != nil {
			return err
		}
	}

	if emitJSON {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(res.Bundle)
	}

	a := res.Bundle.Audit
	s := res.Bundle.Score
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Risk score: %d (%s) — %s\n", s.Score, s.Level, s.Estimate); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Posts: %d, Pages: %d, Active plugins: %d, Active theme: %s\n",
		a.Content.Posts, a.Content.Pages, a.Plugins.ActiveCount, a.Theme.ActiveTheme); err != nil {
		return err
	}
	return nil
}
