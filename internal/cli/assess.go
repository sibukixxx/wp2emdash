package cli

import (
	"path/filepath"
	"time"

	"github.com/sibukixxx/wp2emdash/internal/cli/output"
	"github.com/sibukixxx/wp2emdash/internal/usecase"
	"github.com/spf13/cobra"
)

func newAssessCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "assess",
		Short: "Collect read-only WordPress migration evidence",
		Long:  "Collect a versioned technical inventory, evidence-backed signals, and a deterministic technical score. assess is read-only and never runs migration steps.",
		RunE:  runAssessCmd,
	}
	cmd.Flags().String("wp-root", ".", "WordPress install root (directory containing wp-config.php)")
	cmd.Flags().String("agent-url", "", "HTTP endpoint for a read-only audit agent")
	cmd.Flags().String("agent-token", "", "bearer token for --agent-url (prefer WP2EMDASH_AGENT_TOKEN)")
	cmd.Flags().Duration("agent-timeout", 30*time.Second, "HTTP timeout for --agent-url")
	cmd.Flags().String("ssh", "", "SSH target for remote read-only probes")
	cmd.Flags().Int("ssh-port", 22, "SSH port for --ssh")
	cmd.Flags().String("ssh-key", "", "SSH private key path for --ssh")
	return cmd
}

func runAssessCmd(cmd *cobra.Command, _ []string) error {
	timeout, _ := cmd.Flags().GetDuration("agent-timeout")
	port, _ := cmd.Flags().GetInt("ssh-port")
	res, err := usecase.RunAssessment(cmd.Context(), usecase.AuditParams{
		WPRoot:       mustString(cmd, "wp-root"),
		OutDir:       mustString(cmd, "out"),
		Version:      Version,
		AgentURL:     mustString(cmd, "agent-url"),
		AgentToken:   agentTokenOrEnv(cmd),
		AgentTimeout: timeout,
		SSHTarget:    mustString(cmd, "ssh"),
		SSHPort:      port,
		SSHKey:       mustString(cmd, "ssh-key"),
	})
	if err != nil {
		return err
	}
	if mustBool(cmd, "json") {
		return output.JSON(cmd.OutOrStdout(), res.Summary)
	}
	abs, _ := filepath.Abs(mustString(cmd, "out"))
	return output.Printf(cmd.OutOrStdout(), "wrote read-only assessment artifacts to %s\n", abs)
}
