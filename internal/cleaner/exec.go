package cleaner

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

// LookPath reports whether `name` is on $PATH.
func LookPath(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// RunCmd runs `name args...` and returns combined stdout/stderr.
// Inherits the caller's environment.
func RunCmd(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		return buf.String(), fmt.Errorf("%s %v: %w (%s)", name, args, err, truncate(buf.String(), 200))
	}
	return buf.String(), nil
}

// RunCmdIn runs the command with cwd set to `dir`.
func RunCmdIn(ctx context.Context, dir, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		return buf.String(), fmt.Errorf("%s %v (cwd=%s): %w (%s)", name, args, dir, err, truncate(buf.String(), 200))
	}
	return buf.String(), nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
