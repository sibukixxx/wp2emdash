package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/sibukixxx/wp2emdash/internal/cli/output"
	"github.com/sibukixxx/wp2emdash/internal/usecase"
)

func newMediaCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "media",
		Short: "wp-content/uploads inventory and migration helpers",
	}
	cmd.AddCommand(newMediaScanCmd())
	cmd.AddCommand(newMediaVerifyCmd())
	cmd.AddCommand(newMediaSyncCmd())
	return cmd
}

func newMediaScanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Walk wp-content/uploads (or any directory) and emit a manifest",
		Long: `media scan produces a JSON manifest describing every file under --dir:
size, extension, MIME type, and (with --hash) SHA-256.

The manifest is the input for follow-up commands like 'media sync' (planned)
which delegate the actual transfer to rclone / wrangler / aws-cli.`,
		RunE: runMediaScan,
	}
	cmd.Flags().String("dir", "wp-content/uploads", "directory to scan")
	cmd.Flags().Bool("hash", false, "compute SHA-256 for each file (slow on large trees)")
	cmd.Flags().Int("max-files", 0, "stop after this many files (0 = no limit; useful for sample mode)")
	cmd.Flags().Bool("histogram-only", false, "skip the per-file array, only emit totals + extension counts")
	cmd.Flags().String("manifest", "", "write manifest to this file instead of --out/media-manifest.json")
	cmd.Flags().String("agent-url", "", "HTTP endpoint for a read-only media scan agent")
	cmd.Flags().String("agent-token", "", "bearer token for --agent-url")
	cmd.Flags().Duration("agent-timeout", 30*time.Second, "HTTP timeout for --agent-url")
	cmd.Flags().String("ssh", "", "SSH target for remote media scan execution (example: user@example.com)")
	cmd.Flags().Int("ssh-port", 22, "SSH port for --ssh")
	cmd.Flags().String("ssh-key", "", "SSH private key path for --ssh")
	return cmd
}

func runMediaScan(cmd *cobra.Command, _ []string) error {
	dir := mustString(cmd, "dir")
	agentURL := mustString(cmd, "agent-url")
	sshTarget := mustString(cmd, "ssh")
	if agentURL == "" && sshTarget == "" {
		if _, err := os.Stat(dir); err != nil {
			return fmt.Errorf("scan dir %s: %w", dir, err)
		}
	}

	maxFiles, _ := cmd.Flags().GetInt("max-files")
	emitJSON := mustBool(cmd, "json")

	res, err := usecase.RunMediaScan(cmd.Context(), usecase.MediaScanParams{
		Dir:           dir,
		OutDir:        mustString(cmd, "out"),
		ManifestPath:  mustString(cmd, "manifest"),
		Hash:          mustBool(cmd, "hash"),
		MaxFiles:      maxFiles,
		HistogramOnly: mustBool(cmd, "histogram-only"),
		AgentURL:      agentURL,
		AgentToken:    agentTokenOrEnv(cmd),
		AgentTimeout: func() time.Duration {
			v, _ := cmd.Flags().GetDuration("agent-timeout")
			return v
		}(),
		SSHTarget: sshTarget,
		SSHPort: func() int {
			v, _ := cmd.Flags().GetInt("ssh-port")
			return v
		}(),
		SSHKey: mustString(cmd, "ssh-key"),
	})
	if err != nil {
		return err
	}

	w := cmd.OutOrStdout()
	if emitJSON {
		return output.JSON(w, res.Manifest)
	}

	if err := output.Printf(w, "manifest: %s\n", res.Path); err != nil {
		return err
	}
	if err := output.Printf(w, "files:    %d\n", res.Manifest.TotalFiles); err != nil {
		return err
	}
	if err := output.Printf(w, "bytes:    %d\n", res.Manifest.TotalBytes); err != nil {
		return err
	}
	if len(res.Manifest.Extensions) > 0 {
		if err := output.Println(w, "ext:"); err != nil {
			return err
		}
		for ext, n := range res.Manifest.Extensions {
			if err := output.Printf(w, "  %-8s %d\n", ext, n); err != nil {
				return err
			}
		}
	}
	return nil
}

func newMediaVerifyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Compare a saved media manifest with a verification target",
		Long: `media verify loads an expected media-manifest.json and compares it
against either another manifest or a freshly scanned target directory.

It reports missing files, extra files, size mismatches, and, when hashes are
available, SHA-256 mismatches.`,
		RunE: runMediaVerify,
	}
	cmd.Flags().String("from", "wp2emdash-output/media-manifest.json", "path to the expected media manifest")
	cmd.Flags().String("actual-manifest", "", "path to a second manifest to compare against")
	cmd.Flags().String("dir", "", "target directory to scan when --actual-manifest is not used")
	cmd.Flags().Bool("skip-hash", false, "skip SHA-256 comparison even when the expected manifest contains hashes")
	cmd.Flags().String("report", "", "write verify report to this file instead of --out/media-verify.json")
	cmd.Flags().String("agent-url", "", "HTTP endpoint for a read-only media scan agent")
	cmd.Flags().String("agent-token", "", "bearer token for --agent-url")
	cmd.Flags().Duration("agent-timeout", 30*time.Second, "HTTP timeout for --agent-url")
	cmd.Flags().String("ssh", "", "SSH target for remote verification scan (example: user@example.com)")
	cmd.Flags().Int("ssh-port", 22, "SSH port for --ssh")
	cmd.Flags().String("ssh-key", "", "SSH private key path for --ssh")
	return cmd
}

func runMediaVerify(cmd *cobra.Command, _ []string) error {
	res, err := usecase.RunMediaVerify(cmd.Context(), usecase.MediaVerifyParams{
		FromManifest:   mustString(cmd, "from"),
		ActualManifest: mustString(cmd, "actual-manifest"),
		Dir:            mustString(cmd, "dir"),
		OutDir:         mustString(cmd, "out"),
		ReportPath:     mustString(cmd, "report"),
		SkipHash:       mustBool(cmd, "skip-hash"),
		AgentURL:       mustString(cmd, "agent-url"),
		AgentToken:     agentTokenOrEnv(cmd),
		AgentTimeout: func() time.Duration {
			v, _ := cmd.Flags().GetDuration("agent-timeout")
			return v
		}(),
		SSHTarget: mustString(cmd, "ssh"),
		SSHPort: func() int {
			v, _ := cmd.Flags().GetInt("ssh-port")
			return v
		}(),
		SSHKey: mustString(cmd, "ssh-key"),
	})
	if err != nil {
		return err
	}

	w := cmd.OutOrStdout()
	if mustBool(cmd, "json") {
		return output.JSON(w, res.Report)
	}
	if err := output.Printf(w, "report:   %s\n", res.Path); err != nil {
		return err
	}
	if err := output.Printf(w, "matched:  %d\n", res.Report.MatchedFiles); err != nil {
		return err
	}
	if err := output.Printf(w, "missing:  %d\n", res.Report.MissingFiles); err != nil {
		return err
	}
	if err := output.Printf(w, "extra:    %d\n", res.Report.ExtraFiles); err != nil {
		return err
	}
	if err := output.Printf(w, "size:     %d\n", res.Report.SizeMismatches); err != nil {
		return err
	}
	if err := output.Printf(w, "hash:     %d\n", res.Report.HashMismatches); err != nil {
		return err
	}
	status := "OK"
	if !res.Report.OK {
		status = "FAIL"
	}
	return output.Printf(w, "status:   %s\n", status)
}

func newMediaSyncCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Thin wrapper around rclone copy/sync for media transfer",
		Long: `media sync delegates the actual transfer to rclone.

By default it performs a dry-run and only records the planned command.
Pass --apply to execute the transfer. Use --delete to switch from
non-destructive copy mode to rclone sync mode.`,
		RunE: runMediaSync,
	}
	cmd.Flags().String("dir", "wp-content/uploads", "source directory to transfer")
	cmd.Flags().String("to", "", "destination rclone remote/path (example: r2:bucket/uploads)")
	cmd.Flags().Bool("dry-run", true, "print the planned rclone command without executing it")
	cmd.Flags().Bool("apply", false, "actually execute the transfer (overrides --dry-run)")
	cmd.Flags().Bool("delete", false, "use rclone sync instead of copy (deletes files missing from the source)")
	cmd.Flags().Bool("checksum", false, "pass --checksum to rclone for content-based sync/compare")
	cmd.Flags().String("report", "", "write sync report to this file instead of --out/media-sync.json")
	return cmd
}

func runMediaSync(cmd *cobra.Command, _ []string) error {
	apply := mustBool(cmd, "apply")
	if mustBool(cmd, "dry-run") && !apply {
		apply = false
	}
	res, err := usecase.RunMediaSync(cmd.Context(), usecase.MediaSyncParams{
		Dir:        mustString(cmd, "dir"),
		Dest:       mustString(cmd, "to"),
		OutDir:     mustString(cmd, "out"),
		ReportPath: mustString(cmd, "report"),
		Apply:      apply,
		Delete:     mustBool(cmd, "delete"),
		Checksum:   mustBool(cmd, "checksum"),
	})
	if err != nil {
		return err
	}

	w := cmd.OutOrStdout()
	if mustBool(cmd, "json") {
		return output.JSON(w, res)
	}
	if err := output.Printf(w, "report:   %s\n", res.Path); err != nil {
		return err
	}
	if err := output.Printf(w, "mode:     %s\n", res.Mode); err != nil {
		return err
	}
	if err := output.Printf(w, "applied:  %t\n", res.Applied); err != nil {
		return err
	}
	return output.Printf(w, "command:  %s\n", res.Command.Command)
}
