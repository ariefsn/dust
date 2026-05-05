package browsers

import (
	"context"
	"runtime"

	"github.com/ariefsn/dust/internal/cleaner"
)

// SafariCleaners — darwin only. Only touches caches; never history/bookmarks.
func SafariCleaners() []cleaner.Cleaner {
	if runtime.GOOS != "darwin" {
		return nil
	}
	return []cleaner.Cleaner{
		safariPathCleaner{
			id:   "browsers/safari/cache",
			name: "Safari — cache",
			path: cleaner.Expand("~/Library/Caches/com.apple.Safari"),
		},
		safariPathCleaner{
			id:   "browsers/safari/webkit-cache",
			name: "Safari — WebKit cache",
			path: cleaner.Expand("~/Library/Caches/com.apple.WebKit.PluginProcess"),
		},
	}
}

type safariPathCleaner struct {
	id, name, path string
}

func (s safariPathCleaner) ID() string       { return s.id }
func (s safariPathCleaner) Name() string     { return s.name }
func (s safariPathCleaner) Category() string { return "Browsers" }

func (s safariPathCleaner) Available(ctx context.Context) bool {
	return cleaner.IsDir(s.path)
}

func (s safariPathCleaner) Scan(ctx context.Context) (cleaner.Result, error) {
	bytes, items, err := cleaner.DirSize(ctx, s.path)
	return cleaner.Result{Bytes: bytes, Items: items, Path: s.path}, err
}

func (s safariPathCleaner) Clean(ctx context.Context, opts cleaner.Options) (cleaner.Result, error) {
	before, _ := s.Scan(ctx)
	if opts.DryRun {
		return before, nil
	}
	if err := cleaner.SafeRemoveAll(s.path, []string{cleaner.Home()}); err != nil {
		return cleaner.Result{}, err
	}
	return before, nil
}
