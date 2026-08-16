package cli

import (
	"errors"
	"path/filepath"

	"github.com/sibukixxx/wp2emdash/internal/cli/output"
	"github.com/sibukixxx/wp2emdash/internal/usecase"
	"github.com/spf13/cobra"
)

func newContentCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "content", Short: "Snapshot and independently verify migrated content"}
	snap := &cobra.Command{Use: "snapshot", Short: "Create a privacy-preserving content snapshot"}
	snap.AddCommand(newContentSnapshotCmd("wordpress"), newContentSnapshotCmd("emdash"))
	cmd.AddCommand(snap, newContentVerifyCmd())
	return cmd
}

func newContentSnapshotCmd(source string) *cobra.Command {
	cmd := &cobra.Command{Use: source, Short: "Snapshot " + source + " content without storing body text", RunE: runContentSnapshot}
	cmd.Flags().String("wp-root", ".", "WordPress install root")
	cmd.Flags().String("url", "http://localhost:4321", "EmDash instance URL")
	cmd.Flags().String("map", "", "content mapping JSON")
	cmd.Flags().String("snapshot", "", "output path instead of --out/content-<source>.json")
	cmd.Flags().Bool("write", true, "write the snapshot to disk")
	cmd.Flags().Int("jobs", 4, "maximum concurrent EmDash reads")
	cmd.Flags().String("ssh", "", "SSH target for WordPress snapshot")
	cmd.Flags().Int("ssh-port", 22, "SSH port")
	cmd.Flags().String("ssh-key", "", "SSH private key path")
	return cmd
}

func runContentSnapshot(cmd *cobra.Command, _ []string) error {
	res, err := usecase.RunContentSnapshot(cmd.Context(), usecase.ContentSnapshotParams{Source: cmd.Name(), WPRoot: mustString(cmd, "wp-root"), EmDashURL: mustString(cmd, "url"), OutDir: mustString(cmd, "out"), OutPath: mustString(cmd, "snapshot"), MapPath: mustString(cmd, "map"), Version: Version, Write: mustBool(cmd, "write"), SSHTarget: mustString(cmd, "ssh"), SSHPort: mustInt(cmd, "ssh-port"), SSHKey: mustString(cmd, "ssh-key"), Jobs: mustInt(cmd, "jobs")})
	if err != nil {
		return err
	}
	if mustBool(cmd, "json") {
		return output.JSON(cmd.OutOrStdout(), res.Snapshot)
	}
	abs, _ := filepath.Abs(res.Path)
	if mustBool(cmd, "write") {
		if err := output.Printf(cmd.OutOrStdout(), "wrote %s\n", abs); err != nil {
			return err
		}
	}
	return output.Printf(cmd.OutOrStdout(), "entries: %d\n", len(res.Snapshot.Entries))
}

func newContentVerifyCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "verify", Short: "Prove WordPress content survived an EmDash import", RunE: runContentVerify}
	cmd.Flags().String("expected", "wp2emdash-output/content-wordpress.json", "WordPress snapshot")
	cmd.Flags().String("actual", "wp2emdash-output/content-emdash.json", "EmDash snapshot")
	cmd.Flags().String("map", "", "content mapping JSON")
	cmd.Flags().String("policy", "", "verification gate policy JSON")
	cmd.Flags().String("report", "", "JSON report path")
	cmd.Flags().String("markdown", "", "Markdown report path")
	cmd.Flags().String("resolved-map", "", "resolved identity ledger path")
	cmd.Flags().Bool("write", true, "write reports and resolved identity ledger")
	return cmd
}

func runContentVerify(cmd *cobra.Command, _ []string) error {
	res, err := usecase.RunContentVerify(usecase.ContentVerifyParams{ExpectedPath: mustString(cmd, "expected"), ActualPath: mustString(cmd, "actual"), MapPath: mustString(cmd, "map"), PolicyPath: mustString(cmd, "policy"), OutDir: mustString(cmd, "out"), ReportPath: mustString(cmd, "report"), MarkdownPath: mustString(cmd, "markdown"), ResolvedMapPath: mustString(cmd, "resolved-map"), Version: Version, Write: mustBool(cmd, "write")})
	if err != nil {
		return err
	}
	if mustBool(cmd, "json") {
		if err := output.JSON(cmd.OutOrStdout(), res.Report); err != nil {
			return err
		}
	} else {
		status := "PASS"
		if !res.Report.OK {
			status = "FAIL"
		}
		if err := output.Printf(cmd.OutOrStdout(), "gate:     %s\nmatched:  %d/%d\ncritical: %d\nerrors:   %d\nwarnings: %d\n", status, res.Report.Totals.Matched, res.Report.Totals.Expected, res.Report.Totals.Critical, res.Report.Totals.Errors, res.Report.Totals.Warnings); err != nil {
			return err
		}
	}
	if !res.Report.OK {
		return errors.New("content verification gate failed")
	}
	return nil
}
