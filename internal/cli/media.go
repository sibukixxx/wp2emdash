package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/rokubunnoni-inc/wp2emdash/internal/media"
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

	hash := mustBool(cmd, "hash")
	maxFiles, _ := cmd.Flags().GetInt("max-files")
	histOnly := mustBool(cmd, "histogram-only")
	emitJSON := mustBool(cmd, "json")
	manifestPath := mustString(cmd, "manifest")
	outDir := mustString(cmd, "out")

	manifest, err := media.Scan(dir, media.Options{
		Hash:      hash,
		MaxFiles:  maxFiles,
		WithFiles: !histOnly,
	})
	if err != nil {
		return err
	}

	dest := manifestPath
	if dest == "" {
		dest = filepath.Join(outDir, "media-manifest.json")
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(manifest); err != nil {
		return err
	}

	if emitJSON {
		// Re-encode to stdout for piping.
		stdout := json.NewEncoder(cmd.OutOrStdout())
		stdout.SetIndent("", "  ")
		return stdout.Encode(manifest)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "manifest: %s\n", dest)
	fmt.Fprintf(cmd.OutOrStdout(), "files:    %d\n", manifest.TotalFiles)
	fmt.Fprintf(cmd.OutOrStdout(), "bytes:    %d\n", manifest.TotalBytes)
	if len(manifest.Extensions) > 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "ext:")
		for ext, n := range manifest.Extensions {
			fmt.Fprintf(cmd.OutOrStdout(), "  %-8s %d\n", ext, n)
		}
	}
	return nil
}
