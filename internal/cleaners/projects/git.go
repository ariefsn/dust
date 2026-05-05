package projects

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/ariefsn/dust/internal/cleaner"
)

// hasGit reports whether `dir` (or any ancestor up to one level) contains a .git dir.
func hasGit(dir string) bool {
	if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
		return true
	}
	return false
}

// isDirty runs `git status --porcelain` in dir and reports whether the working
// tree has uncommitted changes. Returns false on any error (including "not a
// git repo") — caller decides whether absent-git counts as dirty.
func isDirty(ctx context.Context, dir string) bool {
	if !cleaner.LookPath("git") {
		return false
	}
	out, err := cleaner.RunCmdIn(ctx, dir, "git", "status", "--porcelain")
	if err != nil {
		return false
	}
	return strings.TrimSpace(out) != ""
}
