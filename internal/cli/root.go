package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Version is set at build time via -ldflags.
var Version = "0.1.0-dev"

// NewRootCmd builds the root cobra command and wires every subcommand.
// Keep this file thin: each subcommand lives in its own file in this package.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "wp2emdash",
		Short: "WordPress → EmDash migration orchestrator",
		Long: `wp2emdash is a Unix-style CLI that decomposes a WordPress → EmDash migration
into small, composable phases (audit, media scan, db plan, env generate, deploy,
cutover). It wraps wp-cli, wrangler, rclone and friends rather than reimplementing
them, and emits JSON/Markdown so its output can flow into other tools.

Always defaults to --dry-run for destructive operations. Production-affecting
commands require an explicit confirmation flag.`,
		SilenceUsage:  true,
		SilenceErrors: false,
		Version:       Version,
	}

	root.PersistentFlags().BoolP("verbose", "v", false, "verbose log output")
	root.PersistentFlags().Bool("json", false, "emit JSON to stdout instead of human-readable text")
	root.PersistentFlags().String("out", "wp2emdash-output", "directory for generated reports and manifests")

	root.AddCommand(newDoctorCmd())
	root.AddCommand(newAuditCmd())
	root.AddCommand(newMediaCmd())
	root.AddCommand(newReportCmd())
	root.AddCommand(newRunCmd())

	return root
}

func mustString(cmd *cobra.Command, name string) string {
	v, err := cmd.Flags().GetString(name)
	if err != nil {
		panic(fmt.Sprintf("flag %q missing: %v", name, err))
	}
	return v
}

func mustBool(cmd *cobra.Command, name string) bool {
	v, err := cmd.Flags().GetBool(name)
	if err != nil {
		panic(fmt.Sprintf("flag %q missing: %v", name, err))
	}
	return v
}

// agentTokenOrEnv returns --agent-token if set, falling back to WP2EMDASH_AGENT_TOKEN.
// Prefer env over CLI args to keep secrets out of shell history.
func agentTokenOrEnv(cmd *cobra.Command) string {
	if t := mustString(cmd, "agent-token"); t != "" {
		return t
	}
	return os.Getenv("WP2EMDASH_AGENT_TOKEN")
}
