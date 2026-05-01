package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/sibukixxx/wp2emdash/internal/cli/output"
	"github.com/sibukixxx/wp2emdash/internal/domain/preset"
	"github.com/sibukixxx/wp2emdash/internal/usecase"
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
	cmd.Flags().String("risk-bands", "", "path to a JSON file that defines public risk bands and estimates")
	cmd.Flags().String("agent-url", "", "HTTP endpoint for a read-only preset agent")
	cmd.Flags().String("agent-audit-url", "", "HTTP endpoint for the preset audit step")
	cmd.Flags().String("agent-media-url", "", "HTTP endpoint for the preset media scan steps")
	cmd.Flags().String("agent-token", "", "bearer token for --agent-url")
	cmd.Flags().Duration("agent-timeout", 30*time.Second, "HTTP timeout for --agent-url")
	cmd.Flags().String("ssh", "", "SSH target for remote preset execution (example: user@example.com)")
	cmd.Flags().Int("ssh-port", 22, "SSH port for --ssh")
	cmd.Flags().String("ssh-key", "", "SSH private key path for --ssh")
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

	w := cmd.OutOrStdout()
	if err := output.Printf(w, "preset: %s\n", p.Name); err != nil {
		return err
	}
	if err := output.Printf(w, "  %s\n", p.Description); err != nil {
		return err
	}
	modeMsg := "  mode:   apply"
	if dryRun {
		modeMsg = "  mode:   dry-run (pass --apply to execute)"
	}
	if err := output.Println(w, modeMsg); err != nil {
		return err
	}
	if err := output.Println(w, ""); err != nil {
		return err
	}

	ctx := cmd.Context()
	agentURL := mustString(cmd, "agent-url")
	params := usecase.PresetParams{
		PresetName:    string(p.Name),
		WPRoot:        wpRoot,
		OutDir:        outDir,
		Version:       Version,
		RiskBandsPath: mustString(cmd, "risk-bands"),
		AgentAuditURL: fallbackString(mustString(cmd, "agent-audit-url"), agentURL),
		AgentMediaURL: fallbackString(mustString(cmd, "agent-media-url"), agentURL),
		AgentToken:    agentTokenOrEnv(cmd),
		AgentTimeout: func() time.Duration {
			v, _ := cmd.Flags().GetDuration("agent-timeout")
			return v
		}(),
		SSHTarget: mustString(cmd, "ssh"),
		SSHPort: func() int {
			port, _ := cmd.Flags().GetInt("ssh-port")
			return port
		}(),
		SSHKey: mustString(cmd, "ssh-key"),
	}
	for _, ph := range p.Phases {
		phaseWarnings := 0
		warningCodes := make([]string, 0, 2)
		if err := output.Printf(w, "phase: %s\n", ph.Name); err != nil {
			return err
		}
		for _, step := range ph.Steps {
			if err := output.Printf(w, "  - [%-18s] %s\n", step.Kind, step.Summary); err != nil {
				return err
			}
			if dryRun {
				continue
			}
			res, err := usecase.RunPresetStep(ctx, step, params)
			if err != nil {
				return fmt.Errorf("%s/%s failed: %w", ph.Name, step.Kind, err)
			}
			phaseWarnings += len(res.Warnings)
			for _, warning := range res.Warnings {
				if len(warningCodes) >= 2 {
					break
				}
				warningCodes = append(warningCodes, warning.Code)
			}
			if step.Kind == "report" {
				if err := output.Printf(w, "    → %s/risk-report.md\n", outDir); err != nil {
					return err
				}
			}
		}
		if !dryRun {
			if err := output.Printf(w, "  warnings: %d\n", phaseWarnings); err != nil {
				return err
			}
			if phaseWarnings > 0 && len(warningCodes) > 0 {
				if err := output.Printf(w, "  warning codes: %s\n", strings.Join(warningCodes, ", ")); err != nil {
					return err
				}
			}
		}
		if err := output.Println(w, ""); err != nil {
			return err
		}
	}
	if !dryRun {
		if err := output.Printf(w, "output: %s\n", outDir); err != nil {
			return err
		}
	}
	return nil
}

func fallbackString(primary, fallback string) string {
	if primary != "" {
		return primary
	}
	return fallback
}
