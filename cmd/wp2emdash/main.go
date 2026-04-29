// wp2emdash — WordPress → EmDash migration orchestrator.
//
// Built as a Unix-style CLI: each subcommand owns one phase
// (audit, media scan, report, env generate, ...) and outputs
// machine-readable JSON or human-readable Markdown so it can
// be composed in pipelines or wrapped by other tools.
//
// Run `wp2emdash --help` for the command tree.
package main

import (
	"fmt"
	"os"

	"github.com/rokubunnoni-inc/wp2emdash/internal/cli"
)

func main() {
	if err := cli.NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
