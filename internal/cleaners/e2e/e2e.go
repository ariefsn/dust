package e2e

import (
	"context"
	"os"
	"path/filepath"
	"runtime"

	"github.com/ariefsn/dust/internal/cleaner"
)

// runnerCacheDir resolves a per-tool cache root, honoring its env-var override.
// `envVar` may be empty if no override is supported.
type runnerCacheDir struct {
	envVar string
	darwin string
	linux  string
}

func (r runnerCacheDir) resolve() string {
	if r.envVar != "" {
		if v := os.Getenv(r.envVar); v != "" {
			return cleaner.Expand(v)
		}
	}
	if runtime.GOOS == "darwin" && r.darwin != "" {
		return cleaner.Expand(r.darwin)
	}
	return cleaner.Expand(r.linux)
}

// runnerCleaner is the shared shape for Cypress / Playwright / Puppeteer.
// It exposes two cleaners per tool: a smart-prune (uses the tool's own
// command, requires the tool be reachable) and a full wipe (path-delete).
type runnerCleaner struct {
	id       string
	name     string
	cache    runnerCacheDir
	tool     string   // npx-callable name (e.g. "cypress"), or "" for path-only
	toolArgs []string // smart-prune args (e.g. ["cache", "prune"])
}

func (rc runnerCleaner) ID() string       { return rc.id }
func (rc runnerCleaner) Name() string     { return rc.name }
func (rc runnerCleaner) Category() string { return "E2E" }

func (rc runnerCleaner) Available(ctx context.Context) bool {
	if cleaner.IsDir(rc.cache.resolve()) {
		return true
	}
	if rc.tool != "" && cleaner.LookPath(rc.tool) {
		return true
	}
	return false
}

func (rc runnerCleaner) Scan(ctx context.Context) (cleaner.Result, error) {
	dir := rc.cache.resolve()
	bytes, items, err := cleaner.DirSize(ctx, dir)
	return cleaner.Result{Bytes: bytes, Items: items, Path: dir}, err
}

// smart-prune action.
func (rc runnerCleaner) prune(ctx context.Context, opts cleaner.Options) (cleaner.Result, error) {
	before, _ := rc.Scan(ctx)
	if opts.DryRun {
		return before, nil
	}
	if rc.tool == "" || !cleaner.LookPath(rc.tool) {
		return cleaner.Result{Path: before.Path}, nil
	}
	if _, err := cleaner.RunCmd(ctx, rc.tool, rc.toolArgs...); err != nil {
		return cleaner.Result{}, err
	}
	after, _ := rc.Scan(ctx)
	freed := before.Bytes - after.Bytes
	if freed < 0 {
		freed = 0
	}
	return cleaner.Result{Bytes: freed, Path: before.Path}, nil
}

// full-wipe action.
func (rc runnerCleaner) wipe(ctx context.Context, opts cleaner.Options) (cleaner.Result, error) {
	before, _ := rc.Scan(ctx)
	if opts.DryRun {
		return before, nil
	}
	if err := cleaner.SafeRemoveAll(before.Path, []string{cleaner.Home(), filepath.Dir(before.Path)}); err != nil {
		return cleaner.Result{}, err
	}
	return before, nil
}

// pruneAction wraps a runnerCleaner so its Clean() runs the smart prune.
type pruneAction struct{ runnerCleaner }

func (p pruneAction) Clean(ctx context.Context, opts cleaner.Options) (cleaner.Result, error) {
	return p.prune(ctx, opts)
}

// wipeAction wraps a runnerCleaner so its Clean() runs a full wipe.
type wipeAction struct{ runnerCleaner }

func (w wipeAction) Clean(ctx context.Context, opts cleaner.Options) (cleaner.Result, error) {
	return w.wipe(ctx, opts)
}

func newRunner(idPrefix, label string, cache runnerCacheDir, tool string, pruneArgs []string) []cleaner.Cleaner {
	out := []cleaner.Cleaner{}
	if tool != "" {
		out = append(out, pruneAction{runnerCleaner{
			id:       idPrefix + "/prune",
			name:     label + " — smart prune",
			cache:    cache,
			tool:     tool,
			toolArgs: pruneArgs,
		}})
	}
	out = append(out, wipeAction{runnerCleaner{
		id:    idPrefix + "/wipe",
		name:  label + " — wipe entire cache",
		cache: cache,
	}})
	return out
}

func Cypress() []cleaner.Cleaner {
	return newRunner(
		"e2e/cypress",
		"Cypress",
		runnerCacheDir{
			envVar: "CYPRESS_CACHE_FOLDER",
			darwin: "~/Library/Caches/Cypress",
			linux:  "~/.cache/Cypress",
		},
		"cypress",
		[]string{"cache", "prune"},
	)
}

func Playwright() []cleaner.Cleaner {
	cache := runnerCacheDir{
		envVar: "PLAYWRIGHT_BROWSERS_PATH",
		darwin: "~/Library/Caches/ms-playwright",
		linux:  "~/.cache/ms-playwright",
	}
	out := []cleaner.Cleaner{}
	// Playwright's smart-prune path is `npx playwright uninstall`. We can't easily
	// detect "playwright reachable via npx" without invoking it, so we expose the
	// prune action only when `npx` and `playwright` are both findable. Fallback
	// to the wipe is always available.
	if cleaner.LookPath("npx") && cleaner.LookPath("playwright") {
		out = append(out, pruneAction{runnerCleaner{
			id:       "e2e/playwright/prune",
			name:     "Playwright — smart prune (uninstall unused)",
			cache:    cache,
			tool:     "playwright",
			toolArgs: []string{"uninstall"},
		}})
	}
	out = append(out, wipeAction{runnerCleaner{
		id:    "e2e/playwright/wipe",
		name:  "Playwright — wipe entire cache",
		cache: cache,
	}})
	return out
}

func Puppeteer() []cleaner.Cleaner {
	cache := runnerCacheDir{
		envVar: "PUPPETEER_CACHE_DIR",
		darwin: "~/.cache/puppeteer",
		linux:  "~/.cache/puppeteer",
	}
	out := []cleaner.Cleaner{}
	if cleaner.LookPath("puppeteer") {
		out = append(out, pruneAction{runnerCleaner{
			id:       "e2e/puppeteer/prune",
			name:     "Puppeteer — smart prune",
			cache:    cache,
			tool:     "puppeteer",
			toolArgs: []string{"browsers", "prune"},
		}})
	}
	out = append(out, wipeAction{runnerCleaner{
		id:    "e2e/puppeteer/wipe",
		name:  "Puppeteer — wipe entire cache",
		cache: cache,
	}})
	return out
}
