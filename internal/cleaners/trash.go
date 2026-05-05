package cleaners

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"

	"github.com/ariefsn/dust/internal/cleaner"
)

// Trash empties the per-user trash. macOS = ~/.Trash, Linux = XDG trash spec.
type trash struct{}

func Trash() cleaner.Cleaner { return trash{} }

func (trash) ID() string       { return "trash" }
func (trash) Name() string     { return "Trash — empty (confirm-twice)" }
func (trash) Category() string { return "System" }

func (trash) Available(ctx context.Context) bool {
	return cleaner.IsDir(trashRoot())
}

func (t trash) Scan(ctx context.Context) (cleaner.Result, error) {
	bytes, items, err := cleaner.DirSize(ctx, trashRoot())
	return cleaner.Result{Bytes: bytes, Items: items, Path: trashRoot()}, err
}

func (t trash) Clean(ctx context.Context, opts cleaner.Options) (cleaner.Result, error) {
	before, _ := t.Scan(ctx)
	if opts.DryRun {
		return before, nil
	}
	root := before.Path
	// Don't delete the trash dir itself — empty its contents.
	entries, err := os.ReadDir(root)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return cleaner.Result{}, nil
		}
		return cleaner.Result{}, err
	}
	for _, e := range entries {
		p := filepath.Join(root, e.Name())
		if err := cleaner.SafeRemoveAll(p, []string{root}); err != nil {
			return cleaner.Result{}, err
		}
	}
	return before, nil
}

func trashRoot() string {
	if runtime.GOOS == "darwin" {
		return cleaner.Expand("~/.Trash")
	}
	return filepath.Join(cleaner.XDGDataHome(), "Trash")
}
