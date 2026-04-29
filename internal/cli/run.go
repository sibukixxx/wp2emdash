package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rokubunnoni-inc/wp2emdash/internal/domain/preset"
	"github.com/rokubunnoni-inc/wp2emdash/internal/usecase"
)

func newRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run a phase preset (minimal / small-production / seo-production / media-heavy / custom-rebuild)",
		Long: `run executes a named preset, which is a curated combination of
phases (doctor → audit → media scan → report). Presets are intentionally
opinionated; pick one that matches the migration scope you're scoping.

Available presets:
  ` + strings.Join(preset.Names(), "\n  ") + `

Default behavior is dry-run: each step prints "would run" without touching
the filesystem (except --out). Pass --apply to actually execute.`,
		RunE: runPreset,
	}
	cmd.Flags().String("preset", string(preset.Minimal), "preset name")
	cmd.Flags().String("wp-root", ".", "WordPress install root")
	cmd.Flags().Bool("dry-run", true, "print steps without executing them")
	cmd.Flags().Bool("apply", false, "actually execute (overrides --dry-run)")
	return cmd
}

func runPreset(cmd *cobra.Command, _ []string) error {
	name := preset.Name(mustString(cmd, "preset"))
	p, err := preset.Lookup(name)
	if err != nil {
		return err
	}

	dryRun := mustBool(cmd, "dry-run")
	if mustBool(cmd, "apply") {
		dryRun = false
	}

	wpRoot := mustString(cmd, "wp-root")
	outDir := mustString(cmd, "out")

	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "preset: %s\n", p.Name); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", p.Description); err != nil {
		return err
	}
	if dryRun {
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), "  mode:   dry-run (pass --apply to execute)"); err != nil {
			return err
		}
	} else {
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), "  mode:   apply"); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(cmd.OutOrStdout(), ""); err != nil {
		return err
	}

	ctx := cmd.Context()
	params := usecase.PresetParams{
		WPRoot:  wpRoot,
		OutDir:  outDir,
		Version: Version,
	}
	for _, ph := range p.Phases {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "phase: %s\n", ph.Name); err != nil {
			return err
		}
		for _, step := range ph.Steps {
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "  - [%-18s] %s\n", step.Kind, step.Summary); err != nil {
				return err
			}
			if dryRun {
				continue
			}
			if err := usecase.RunPresetStep(ctx, step, params); err != nil {
				return fmt.Errorf("%s/%s failed: %w", ph.Name, step.Kind, err)
			}
			if step.Kind == "report" {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "    → %s/risk-report.md\n", outDir); err != nil {
					return err
				}
			}
		}
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), ""); err != nil {
			return err
		}
	}
	if !dryRun {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "output: %s\n", outDir); err != nil {
			return err
		}
	}
	return nil
}
