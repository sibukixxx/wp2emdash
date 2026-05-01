package cli

import (
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/sibukixxx/wp2emdash/internal/cli/output"
	"github.com/sibukixxx/wp2emdash/internal/usecase"
)

func newSEOCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "seo",
		Short: "Extract SEO metadata, redirects, and compare URL maps",
	}
	cmd.AddCommand(newSEOExtractMetaCmd())
	cmd.AddCommand(newSEOExtractRedirectsCmd())
	cmd.AddCommand(newSEOURLMapCmd())
	return cmd
}

func newSEOExtractMetaCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "extract-meta",
		Short: "Dump per-post SEO metadata (title / description / canonical / OG)",
		Long: `extract-meta lists every published post / page on the WordPress install and
collects SEO metadata from Yoast / Rank Math / AIOSEO post meta keys.

When several SEO plugins are active the precedence is Yoast > Rank Math >
AIOSEO. Output is a JSON document at <out>/seo-meta.json by default.`,
		RunE: runSEOExtractMeta,
	}
	cmd.Flags().String("wp-root", ".", "WordPress install root (directory containing wp-config.php)")
	cmd.Flags().Bool("write", true, "write seo-meta.json to --out")
	cmd.Flags().String("manifest", "", "write manifest to this file instead of --out/seo-meta.json")
	cmd.Flags().String("ssh", "", "SSH target for remote execution (example: user@example.com)")
	cmd.Flags().Int("ssh-port", 22, "SSH port for --ssh")
	cmd.Flags().String("ssh-key", "", "SSH private key path for --ssh")
	return cmd
}

func runSEOExtractMeta(cmd *cobra.Command, _ []string) error {
	res, err := usecase.RunSEOExtractMeta(cmd.Context(), usecase.SEOMetaParams{
		WPRoot:    mustString(cmd, "wp-root"),
		OutDir:    mustString(cmd, "out"),
		Write:     mustBool(cmd, "write"),
		Version:   Version,
		OutPath:   mustString(cmd, "manifest"),
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
	emitJSON := mustBool(cmd, "json")
	if emitJSON {
		return output.JSON(w, res.Set)
	}
	abs, _ := filepath.Abs(res.Path)
	if mustBool(cmd, "write") {
		if err := output.Printf(w, "wrote %s\n", abs); err != nil {
			return err
		}
	}
	return output.Printf(w, "items: %d\n", len(res.Set.Items))
}

func newSEOExtractRedirectsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "extract-redirects",
		Short: "Dump redirect rules from .htaccess and Redirection / SRM plugins",
		Long: `extract-redirects merges three sources of redirect rules into one JSON file:

  - .htaccess (Redirect, RedirectMatch, RewriteRule with R flag)
  - Redirection plugin (wp_redirection_items)
  - Safe Redirect Manager (post_type=redirect_rule)

Output is written to <out>/seo-redirects.json by default.`,
		RunE: runSEOExtractRedirects,
	}
	cmd.Flags().String("wp-root", ".", "WordPress install root (directory containing wp-config.php)")
	cmd.Flags().Bool("write", true, "write seo-redirects.json to --out")
	cmd.Flags().String("manifest", "", "write manifest to this file instead of --out/seo-redirects.json")
	cmd.Flags().String("ssh", "", "SSH target for remote execution (example: user@example.com)")
	cmd.Flags().Int("ssh-port", 22, "SSH port for --ssh")
	cmd.Flags().String("ssh-key", "", "SSH private key path for --ssh")
	return cmd
}

func runSEOExtractRedirects(cmd *cobra.Command, _ []string) error {
	res, err := usecase.RunSEOExtractRedirects(cmd.Context(), usecase.SEORedirectsParams{
		WPRoot:    mustString(cmd, "wp-root"),
		OutDir:    mustString(cmd, "out"),
		Write:     mustBool(cmd, "write"),
		Version:   Version,
		OutPath:   mustString(cmd, "manifest"),
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
	emitJSON := mustBool(cmd, "json")
	if emitJSON {
		return output.JSON(w, res.Set)
	}
	abs, _ := filepath.Abs(res.Path)
	if mustBool(cmd, "write") {
		if err := output.Printf(w, "wrote %s\n", abs); err != nil {
			return err
		}
	}
	return output.Printf(w, "rules: %d\n", len(res.Set.Rules))
}

func newSEOURLMapCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "url-map",
		Short: "Compare two URL maps and report missing / added URLs",
		Long: `url-map reads two URL maps (--old and --new) and reports which URLs are
matched, missing from the new site, or new on the new site.

Each input may be a JSON file matching the seo.URLMap shape, or a plain text
file with one URL per line (lines starting with '#' are skipped).

URLs are compared after light normalization: scheme (http vs https), trailing
slashes, and fragments are ignored. Path case is preserved.`,
		RunE: runSEOURLMap,
	}
	cmd.Flags().String("old", "", "path to the old URL map (JSON or text)")
	cmd.Flags().String("new", "", "path to the new URL map (JSON or text)")
	cmd.Flags().Bool("write", true, "write seo-url-map.json to --out")
	cmd.Flags().String("manifest", "", "write manifest to this file instead of --out/seo-url-map.json")
	return cmd
}

func runSEOURLMap(cmd *cobra.Command, _ []string) error {
	res, err := usecase.RunSEOURLMap(usecase.SEOURLMapParams{
		OldPath: mustString(cmd, "old"),
		NewPath: mustString(cmd, "new"),
		OutDir:  mustString(cmd, "out"),
		OutPath: mustString(cmd, "manifest"),
		Write:   mustBool(cmd, "write"),
		Version: Version,
	})
	if err != nil {
		return err
	}

	w := cmd.OutOrStdout()
	emitJSON := mustBool(cmd, "json")
	if emitJSON {
		return output.JSON(w, res.Diff)
	}
	abs, _ := filepath.Abs(res.Path)
	if mustBool(cmd, "write") {
		if err := output.Printf(w, "wrote %s\n", abs); err != nil {
			return err
		}
	}
	if err := output.Printf(w, "matched:     %d\n", res.Diff.Total.Matched); err != nil {
		return err
	}
	if err := output.Printf(w, "only in old: %d\n", res.Diff.Total.OnlyInOld); err != nil {
		return err
	}
	return output.Printf(w, "only in new: %d\n", res.Diff.Total.OnlyInNew)
}
