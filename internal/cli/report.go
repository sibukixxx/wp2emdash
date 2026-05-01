package cli

import (
	"github.com/spf13/cobra"

	"github.com/sibukixxx/wp2emdash/internal/usecase"
	"github.com/sibukixxx/wp2emdash/internal/usecase/reporting"
)

func newReportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "report",
		Short: "Render risk-report.md from an existing summary.json",
		Long: `report regenerates the human-facing risk-report.md from a previously
written summary.json. Use it after editing the JSON manually, or to preview
the report on stdout without re-running the audit.`,
		RunE: runReport,
	}
	cmd.Flags().String("from", "wp2emdash-output/summary.json", "path to summary.json")
	cmd.Flags().Bool("stdout", false, "print to stdout instead of writing risk-report.md")
	return cmd
}

func runReport(cmd *cobra.Command, _ []string) error {
	from := mustString(cmd, "from")
	toStdout := mustBool(cmd, "stdout")
	outDir := mustString(cmd, "out")

	bundle, err := usecase.LoadReportBundle(from)
	if err != nil {
		return err
	}

	if toStdout {
		return reporting.RenderMarkdown(cmd.OutOrStdout(), bundle)
	}
	return usecase.WriteReport(outDir, bundle)
}
