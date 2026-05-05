package cleaners

import (
	"context"

	"github.com/ariefsn/dust/internal/cleaner"
)

// Yarn — `yarn cache clean` if `yarn` is on PATH, otherwise wipe the cache dir.
type yarn struct{}

func Yarn() cleaner.Cleaner { return yarn{} }

func (yarn) ID() string       { return "js/yarn" }
func (yarn) Name() string     { return "Yarn — cache" }
func (yarn) Category() string { return "JS" }

func (yarn) Available(ctx context.Context) bool {
	if cleaner.LookPath("yarn") {
		return true
	}
	return cleaner.IsDir(yarnCacheDir())
}

func (y yarn) Scan(ctx context.Context) (cleaner.Result, error) {
	dir := yarnCacheDir()
	bytes, items, err := cleaner.DirSize(ctx, dir)
	return cleaner.Result{Bytes: bytes, Items: items, Path: dir}, err
}

func (y yarn) Clean(ctx context.Context, opts cleaner.Options) (cleaner.Result, error) {
	before, _ := y.Scan(ctx)
	if opts.DryRun {
		return before, nil
	}
	// Prefer the tool when available — it knows the right paths for all yarn versions.
	if cleaner.LookPath("yarn") {
		if _, err := cleaner.RunCmd(ctx, "yarn", "cache", "clean"); err == nil {
			after, _ := y.Scan(ctx)
			freed := before.Bytes - after.Bytes
			if freed < 0 {
				freed = before.Bytes
			}
			return cleaner.Result{Bytes: freed, Path: before.Path}, nil
		}
		// fall through to path-delete on tool failure
	}
	if err := cleaner.SafeRemoveAll(before.Path, []string{cleaner.Home()}); err != nil {
		return cleaner.Result{}, err
	}
	return cleaner.Result{Bytes: before.Bytes, Items: before.Items, Path: before.Path}, nil
}

func yarnCacheDir() string {
	// Yarn classic + berry both default here on darwin/linux unless overridden.
	// Yarn 3+ uses ~/.yarn/berry/cache; cover both via the Caches/Yarn root.
	if cleaner.IsDir(cleaner.Expand("~/Library/Caches/Yarn")) {
		return cleaner.Expand("~/Library/Caches/Yarn")
	}
	return cleaner.Expand("~/.cache/yarn")
}

// NPM — `npm cache clean --force` if available, else wipe ~/.npm.
func NPM() cleaner.Cleaner {
	return pathBased{
		id:       "js/npm",
		name:     "npm — cache",
		category: "JS",
		resolvePath: func() string {
			return cleaner.Expand("~/.npm")
		},
		availableExtra: func(ctx context.Context) bool {
			return cleaner.LookPath("npm")
		},
		tool:     "npm",
		toolArgs: []string{"cache", "clean", "--force"},
	}
}

// pnpm store prune — orphans only (safe). Wiping the entire store is exposed as
// a separate cleaner so the user has to opt in.
type pnpmPrune struct{}

func PnpmPrune() cleaner.Cleaner { return pnpmPrune{} }

func (pnpmPrune) ID() string       { return "js/pnpm/prune" }
func (pnpmPrune) Name() string     { return "pnpm — store prune (safe)" }
func (pnpmPrune) Category() string { return "JS" }

func (pnpmPrune) Available(ctx context.Context) bool {
	return cleaner.LookPath("pnpm") || cleaner.IsDir(pnpmStoreDir())
}

func (p pnpmPrune) Scan(ctx context.Context) (cleaner.Result, error) {
	dir := pnpmStoreDir()
	bytes, items, err := cleaner.DirSize(ctx, dir)
	return cleaner.Result{Bytes: bytes, Items: items, Path: dir}, err
}

func (p pnpmPrune) Clean(ctx context.Context, opts cleaner.Options) (cleaner.Result, error) {
	before, _ := p.Scan(ctx)
	if opts.DryRun {
		return before, nil
	}
	if !cleaner.LookPath("pnpm") {
		// Without the tool, "safe prune" isn't possible — refuse rather than
		// quietly fall through to a full wipe.
		return cleaner.Result{Path: before.Path}, nil
	}
	if _, err := cleaner.RunCmd(ctx, "pnpm", "store", "prune"); err != nil {
		return cleaner.Result{}, err
	}
	after, _ := p.Scan(ctx)
	freed := before.Bytes - after.Bytes
	if freed < 0 {
		freed = 0
	}
	return cleaner.Result{Bytes: freed, Path: before.Path}, nil
}

// PnpmStoreWipe nukes the entire content-addressable store. Forces every future
// `pnpm install` to re-download — only worth it when the store is truly bloated.
func PnpmStoreWipe() cleaner.Cleaner {
	return pathBased{
		id:       "js/pnpm/wipe",
		name:     "pnpm — wipe entire store",
		category: "JS",
		resolvePath: func() string {
			d := pnpmStoreDir()
			if cleaner.IsDir(d) {
				return d
			}
			return ""
		},
	}
}

func pnpmStoreDir() string {
	// Default locations per pnpm docs:
	//   darwin: ~/Library/pnpm/store
	//   linux:  ~/.local/share/pnpm/store
	if d := cleaner.Expand("~/Library/pnpm/store"); cleaner.IsDir(d) {
		return d
	}
	if d := cleaner.Expand("~/.local/share/pnpm/store"); cleaner.IsDir(d) {
		return d
	}
	if d := cleaner.Expand("~/.pnpm-store"); cleaner.IsDir(d) {
		return d
	}
	return cleaner.Expand("~/Library/pnpm/store")
}

// Bun — `bun pm cache rm` if `bun` is on PATH, else wipe ~/.bun/install/cache.
func Bun() cleaner.Cleaner {
	return pathBased{
		id:       "js/bun",
		name:     "Bun — cache",
		category: "JS",
		resolvePath: func() string {
			return cleaner.Expand("~/.bun/install/cache")
		},
		availableExtra: func(ctx context.Context) bool {
			return cleaner.LookPath("bun")
		},
		tool:     "bun",
		toolArgs: []string{"pm", "cache", "rm"},
	}
}
