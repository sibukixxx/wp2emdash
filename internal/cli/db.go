package cli

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/sibukixxx/wp2emdash/internal/cli/output"
	"github.com/sibukixxx/wp2emdash/internal/usecase"
)

func newDBCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "db",
		Short: "Database migration planning helpers",
	}
	cmd.AddCommand(newDBPlanCmd())
	return cmd
}

func newDBPlanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Generate a database migration plan from summary.json",
		Long: `db plan reads an existing summary.json and turns the audit result into
an export/review/transform plan for core WordPress tables and metadata.

It does not dump or modify a database. The output is a JSON/Markdown plan
that helps scope the migration work before any SQL leaves the source site.`,
		RunE: runDBPlan,
	}
	cmd.Flags().String("from", "wp2emdash-output/summary.json", "path to summary.json")
	cmd.Flags().String("preset", "small-production", "target preset name used to contextualize the plan")
	cmd.Flags().Bool("write", true, "write db-plan.json + db-plan.md to --out")
	return cmd
}

func runDBPlan(cmd *cobra.Command, _ []string) error {
	res, err := usecase.RunDBPlan(usecase.DBPlanParams{
		From:   mustString(cmd, "from"),
		OutDir: mustString(cmd, "out"),
		Preset: mustString(cmd, "preset"),
		Write:  mustBool(cmd, "write"),
	})
	if err != nil {
		return err
	}

	w := cmd.OutOrStdout()
	if mustBool(cmd, "json") {
		return output.JSON(w, res.Plan)
	}

	if mustBool(cmd, "write") {
		if err := output.Printf(w, "wrote %s\n", res.JSONPath); err != nil {
			return err
		}
		if err := output.Printf(w, "wrote %s\n", res.MarkdownPath); err != nil {
			return err
		}
	}
	if err := output.Printf(w, "strategy: %s\n", res.Plan.Strategy); err != nil {
		return err
	}
	if err := output.Printf(w, "tables: %d\n", len(res.Plan.Tables)); err != nil {
		return err
	}
	if len(res.Plan.Risks) > 0 {
		if err := output.Printf(w, "risks: %d\n", len(res.Plan.Risks)); err != nil {
			return err
		}
	}
	if !mustBool(cmd, "write") && !mustBool(cmd, "json") {
		return errors.New("db plan generated in memory only; pass --write to persist artifacts or --json to inspect the plan")
	}
	return nil
}
