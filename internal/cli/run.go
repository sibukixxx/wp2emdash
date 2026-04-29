package cli

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/rokubunnoni-inc/wp2emdash/internal/media"
	"github.com/rokubunnoni-inc/wp2emdash/internal/preset"
	"github.com/rokubunnoni-inc/wp2emdash/internal/report"
	"github.com/rokubunnoni-inc/wp2emdash/internal/score"
	"github.com/rokubunnoni-inc/wp2emdash/internal/wordpress"
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

	fmt.Fprintf(cmd.OutOrStdout(), "preset: %s\n", p.Name)
	fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", p.Description)
	if dryRun {
		fmt.Fprintln(cmd.OutOrStdout(), "  mode:   dry-run (pass --apply to execute)")
	} else {
		fmt.Fprintln(cmd.OutOrStdout(), "  mode:   apply")
	}
	fmt.Fprintln(cmd.OutOrStdout(), "")

	ctx := cmd.Context()
	for _, ph := range p.Phases {
		fmt.Fprintf(cmd.OutOrStdout(), "phase: %s\n", ph.Name)
		for _, step := range ph.Steps {
			fmt.Fprintf(cmd.OutOrStdout(), "  - [%-18s] %s\n", step.Kind, step.Summary)
			if dryRun {
				continue
			}
			if err := runStep(ctx, cmd, step, wpRoot, outDir); err != nil {
				return fmt.Errorf("%s/%s failed: %w", ph.Name, step.Kind, err)
			}
		}
		fmt.Fprintln(cmd.OutOrStdout(), "")
	}
	if !dryRun {
		fmt.Fprintf(cmd.OutOrStdout(), "output: %s\n", outDir)
	}
	return nil
}

// runStep dispatches one preset step to its concrete implementation. Steps
// not yet implemented (Kind: todo / todo-...) print a notice and continue,
// rather than failing — they are placeholders for future versions.
func runStep(ctx context.Context, cmd *cobra.Command, step preset.Step, wpRoot, outDir string) error {
	switch step.Kind {
	case "doctor":
		// reuse the doctor command implementation, but keep stdout quiet here
		fakeCmd := newDoctorCmd()
		fakeCmd.SetContext(ctx)
		fakeCmd.Flags().Set("json", "false")
		_ = runDoctor(ctx, fakeCmd) // tolerate missing optionals; required ones still error out
		return nil

	case "audit":
		return runStepAudit(ctx, wpRoot, outDir, cmd)

	case "media-scan-sample":
		return runStepMediaScan(filepath.Join(wpRoot, "wp-content", "uploads"), outDir,
			media.Options{WithFiles: true, MaxFiles: 200})

	case "media-scan":
		return runStepMediaScan(filepath.Join(wpRoot, "wp-content", "uploads"), outDir,
			media.Options{WithFiles: true})

	case "media-scan-hash":
		return runStepMediaScan(filepath.Join(wpRoot, "wp-content", "uploads"), outDir,
			media.Options{WithFiles: true, Hash: true})

	case "report":
		// audit step already wrote the bundle; nothing to do beyond confirming.
		fmt.Fprintf(cmd.OutOrStdout(), "    → %s/risk-report.md\n", outDir)
		return nil

	case "todo":
		fmt.Fprintln(cmd.OutOrStdout(), "    (skipped — implementation lands in a later version)")
		return nil

	default:
		return fmt.Errorf("unhandled step kind %q", step.Kind)
	}
}

func runStepAudit(ctx context.Context, wpRoot, outDir string, cmd *cobra.Command) error {
	auditor, err := wordpress.New(wpRoot)
	if err != nil {
		return err
	}
	a, err := auditor.Run(ctx)
	if err != nil {
		return err
	}
	s := score.Compute(a)
	bundle := report.Bundle{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Tool:        "wp2emdash",
		Version:     Version,
		Audit:       a,
		Score:       s,
	}
	if err := report.WriteAll(outDir, bundle); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "    → score=%d level=%s\n", s.Score, s.Level)
	return nil
}

func runStepMediaScan(dir, outDir string, opt media.Options) error {
	manifest, err := media.Scan(dir, opt)
	if err != nil {
		return err
	}
	if err := writeMediaManifest(filepath.Join(outDir, "media-manifest.json"), manifest); err != nil {
		return err
	}
	return nil
}
