package cleaners

import (
	"context"

	"github.com/ariefsn/dust/internal/cleaner"
)

// pathBased is a generic cleaner that scans a directory and deletes its contents.
// If `tool` is non-empty and on $PATH, `toolArgs` runs first; on success, the cleaner
// reports the freed bytes via a before/after diff and skips the path-delete fallback.
type pathBased struct {
	id       string
	name     string
	category string

	// resolvePath returns the cache directory (resolved at scan/clean time so
	// env-var changes are reflected). May return "" to indicate "not present".
	resolvePath func() string

	// availableExtra is an optional check beyond "path exists"; OR'd with path existence.
	availableExtra func(ctx context.Context) bool

	tool     string
	toolArgs []string
}

func (p pathBased) ID() string       { return p.id }
func (p pathBased) Name() string     { return p.name }
func (p pathBased) Category() string { return p.category }

func (p pathBased) Available(ctx context.Context) bool {
	if p.availableExtra != nil && p.availableExtra(ctx) {
		return true
	}
	dir := p.resolvePath()
	return dir != "" && cleaner.IsDir(dir)
}

func (p pathBased) Scan(ctx context.Context) (cleaner.Result, error) {
	dir := p.resolvePath()
	if dir == "" {
		return cleaner.Result{}, nil
	}
	bytes, items, err := cleaner.DirSize(ctx, dir)
	return cleaner.Result{Bytes: bytes, Items: items, Path: dir}, err
}

func (p pathBased) Clean(ctx context.Context, opts cleaner.Options) (cleaner.Result, error) {
	before, _ := p.Scan(ctx)
	if before.Path == "" {
		return cleaner.Result{}, nil
	}
	if opts.DryRun {
		return before, nil
	}
	if p.tool != "" && cleaner.LookPath(p.tool) {
		if _, err := cleaner.RunCmd(ctx, p.tool, p.toolArgs...); err == nil {
			after, _ := p.Scan(ctx)
			freed := before.Bytes - after.Bytes
			if freed < 0 {
				freed = before.Bytes
			}
			return cleaner.Result{Bytes: freed, Path: before.Path}, nil
		}
	}
	if err := cleaner.SafeRemoveAll(before.Path, []string{cleaner.Home()}); err != nil {
		return cleaner.Result{}, err
	}
	return cleaner.Result{Bytes: before.Bytes, Items: before.Items, Path: before.Path}, nil
}
