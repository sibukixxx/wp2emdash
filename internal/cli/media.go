package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/rokubunnoni-inc/wp2emdash/internal/cli/output"
	"github.com/rokubunnoni-inc/wp2emdash/internal/usecase"
)

func newMediaCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "media",
		Short: "wp-content/uploads inventory and migration helpers",
	}
	cmd.AddCommand(newMediaScanCmd())
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
	return cmd
}

func runMediaScan(cmd *cobra.Command, _ []string) error {
	dir := mustString(cmd, "dir")
	if _, err := os.Stat(dir); err != nil {
		return fmt.Errorf("scan dir %s: %w", dir, err)
	}

	maxFiles, _ := cmd.Flags().GetInt("max-files")
	emitJSON := mustBool(cmd, "json")

	res, err := usecase.RunMediaScan(usecase.MediaScanParams{
		Dir:           dir,
		OutDir:        mustString(cmd, "out"),
		ManifestPath:  mustString(cmd, "manifest"),
		Hash:          mustBool(cmd, "hash"),
		MaxFiles:      maxFiles,
		HistogramOnly: mustBool(cmd, "histogram-only"),
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
