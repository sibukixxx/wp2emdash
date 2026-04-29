// Package shell wraps os/exec with a small Runner type that captures
// stdout/stderr/exit-code and supports "dry-run" so commands can be logged
// without actually executing.
package shell

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
)

// Result describes a single command invocation.
type Result struct {
	Command  string `json:"command"`
	Stdout   string `json:"stdout,omitempty"`
	Stderr   string `json:"stderr,omitempty"`
	ExitCode int    `json:"exit_code"`
	DryRun   bool   `json:"dry_run,omitempty"`
}

// Runner executes commands. Reuse a single Runner per logical task so its
// configuration (working dir, dry-run, env) applies consistently.
type Runner struct {
	Dir    string
	Env    []string
	DryRun bool
}

// Run executes name with args (or, if DryRun is set, returns a synthesized
// "would have run" Result without calling exec).
func (r Runner) Run(ctx context.Context, name string, args ...string) (Result, error) {
	res := Result{Command: formatCommand(name, args), DryRun: r.DryRun}
	if r.DryRun {
		return res, nil
	}

	cmd := exec.CommandContext(ctx, name, args...)
	if r.Dir != "" {
		cmd.Dir = r.Dir
	}
	if len(r.Env) > 0 {
		cmd.Env = append(cmd.Env, r.Env...)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	res.Stdout = stdout.String()
	res.Stderr = stderr.String()

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			res.ExitCode = exitErr.ExitCode()
		} else {
			res.ExitCode = -1
		}
		return res, err
	}
	return res, nil
}

// Output is a thin wrapper that returns trimmed stdout for the common case.
func (r Runner) Output(ctx context.Context, name string, args ...string) (string, error) {
	res, err := r.Run(ctx, name, args...)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(res.Stdout, "\n\r \t"), nil
}

func formatCommand(name string, args []string) string {
	parts := append([]string{name}, args...)
	return strings.Join(parts, " ")
}
