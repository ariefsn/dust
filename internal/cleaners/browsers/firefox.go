package browsers

import (
	"context"
	"path/filepath"
	"runtime"

	"github.com/ariefsn/dust/internal/cleaner"
)

// Firefox cache lives under both:
//   - <profile root>/Profiles/<profile>/cache2  (per-profile, persistent)
//   - ~/Library/Caches/Firefox/Profiles/<profile>/cache2 (Mac caches root, smaller)
//
// On Linux it's all under ~/.cache/mozilla/firefox/.

func FirefoxCleaners() []cleaner.Cleaner {
	return []cleaner.Cleaner{
		firefoxCache{name: "Firefox — HTTP cache (cache2)", subPath: "cache2"},
		firefoxCache{name: "Firefox — startup cache", subPath: "startupCache"},
	}
}

type firefoxCache struct {
	name    string
	subPath string
}

func (f firefoxCache) ID() string       { return "browsers/firefox/" + f.subPath }
func (f firefoxCache) Name() string     { return f.name }
func (f firefoxCache) Category() string { return "Browsers" }

func (f firefoxCache) roots() []string {
	switch runtime.GOOS {
	case "darwin":
		return []string{
			cleaner.Expand("~/Library/Caches/Firefox/Profiles"),
			cleaner.Expand("~/Library/Application Support/Firefox/Profiles"),
		}
	default:
		return []string{
			filepath.Join(cleaner.XDGCacheHome(), "mozilla", "firefox"),
			cleaner.Expand("~/.mozilla/firefox"),
		}
	}
}

func (f firefoxCache) targets() []string {
	var out []string
	for _, root := range f.roots() {
		matches, _ := filepath.Glob(filepath.Join(root, "*", f.subPath))
		for _, m := range matches {
			if cleaner.IsDir(m) {
				out = append(out, m)
			}
		}
	}
	return out
}

func (f firefoxCache) Available(ctx context.Context) bool {
	return len(f.targets()) > 0
}

func (f firefoxCache) Scan(ctx context.Context) (cleaner.Result, error) {
	var total int64
	var items int
	targets := f.targets()
	for _, t := range targets {
		b, n, err := cleaner.DirSize(ctx, t)
		if err != nil {
			return cleaner.Result{}, err
		}
		total += b
		items += n
	}
	rootHint := ""
	if rs := f.roots(); len(rs) > 0 {
		rootHint = rs[0]
	}
	return cleaner.Result{Bytes: total, Items: items, Path: rootHint}, nil
}

func (f firefoxCache) Clean(ctx context.Context, opts cleaner.Options) (cleaner.Result, error) {
	before, _ := f.Scan(ctx)
	if opts.DryRun {
		return before, nil
	}
	for _, t := range f.targets() {
		if err := cleaner.SafeRemoveAll(t, []string{cleaner.Home()}); err != nil {
			return cleaner.Result{}, err
		}
	}
	return before, nil
}
