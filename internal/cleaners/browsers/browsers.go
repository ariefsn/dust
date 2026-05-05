package browsers

import (
	"context"
	"path/filepath"

	"github.com/ariefsn/dust/internal/cleaner"
)

// All returns every browser cleaner available on the current platform.
func All() []cleaner.Cleaner {
	out := []cleaner.Cleaner{}
	out = append(out, ChromeCleaners()...)
	out = append(out, BraveCleaners()...)
	out = append(out, EdgeCleaners()...)
	out = append(out, ArcCleaners()...)
	out = append(out, OperaCleaners()...)
	out = append(out, ChromiumCleaners()...)
	out = append(out, FirefoxCleaners()...)
	out = append(out, SafariCleaners()...)
	return out
}

// chromiumSubitem cleans a named cache subdir across every profile in a
// Chromium-family profile root. E.g. wipe Cache/Cache_Data under Default,
// Profile 1, Profile 2, ... in one go.
type chromiumSubitem struct {
	id, name, browser string
	profileRoot       func() string // resolves at scan/clean time
	subPath           string        // e.g. "Cache/Cache_Data"
}

func (c chromiumSubitem) ID() string       { return c.id }
func (c chromiumSubitem) Name() string     { return c.name }
func (c chromiumSubitem) Category() string { return "Browsers" }

func (c chromiumSubitem) Available(ctx context.Context) bool {
	return cleaner.IsDir(c.profileRoot())
}

func (c chromiumSubitem) targets() []string {
	root := c.profileRoot()
	if !cleaner.IsDir(root) {
		return nil
	}
	// Chromium profiles: Default, Profile 1, Profile 2, Guest Profile, ...
	patterns := []string{
		filepath.Join(root, "Default", c.subPath),
		filepath.Join(root, "Profile *", c.subPath),
		filepath.Join(root, "Guest Profile", c.subPath),
		filepath.Join(root, "System Profile", c.subPath),
	}
	var out []string
	for _, p := range patterns {
		matches, _ := filepath.Glob(p)
		for _, m := range matches {
			if cleaner.IsDir(m) {
				out = append(out, m)
			}
		}
	}
	return out
}

func (c chromiumSubitem) Scan(ctx context.Context) (cleaner.Result, error) {
	var total int64
	var items int
	targets := c.targets()
	for _, t := range targets {
		b, n, err := cleaner.DirSize(ctx, t)
		if err != nil {
			return cleaner.Result{}, err
		}
		total += b
		items += n
	}
	path := c.profileRoot()
	return cleaner.Result{Bytes: total, Items: items, Path: path}, nil
}

func (c chromiumSubitem) Clean(ctx context.Context, opts cleaner.Options) (cleaner.Result, error) {
	before, _ := c.Scan(ctx)
	if opts.DryRun {
		return before, nil
	}
	for _, t := range c.targets() {
		if err := cleaner.SafeRemoveAll(t, []string{cleaner.Home(), c.profileRoot()}); err != nil {
			return cleaner.Result{}, err
		}
	}
	return before, nil
}

// chromiumFamily builds the standard 5 cache cleaners for a Chromium-based
// browser given its profile-root resolver and a short ID prefix.
func chromiumFamily(idPrefix, browserLabel string, profileRoot func() string) []cleaner.Cleaner {
	mk := func(suffix, label, sub string) cleaner.Cleaner {
		return chromiumSubitem{
			id:          idPrefix + "/" + suffix,
			name:        browserLabel + " — " + label,
			browser:     browserLabel,
			profileRoot: profileRoot,
			subPath:     sub,
		}
	}
	return []cleaner.Cleaner{
		mk("http-cache", "HTTP cache", filepath.Join("Cache", "Cache_Data")),
		mk("gpu-cache", "GPU cache", "GPUCache"),
		mk("code-cache", "Code cache", "Code Cache"),
		mk("service-workers", "Service workers", filepath.Join("Service Worker", "CacheStorage")),
		mk("indexeddb", "IndexedDB (may sign you out of PWAs)", "IndexedDB"),
	}
}
