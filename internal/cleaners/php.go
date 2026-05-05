package cleaners

import (
	"context"
	"os"
	"path/filepath"
	"runtime"

	"github.com/ariefsn/dust/internal/cleaner"
)

// Composer — `composer clear-cache` if available, else wipe the cache dir.
//
// Composer's cache lives in different places depending on platform and version:
//   - $COMPOSER_CACHE_DIR (override)
//   - $XDG_CACHE_HOME/composer (Linux + newer macOS)
//   - ~/.composer/cache (legacy)
func Composer() cleaner.Cleaner {
	return pathBased{
		id:       "php/composer",
		name:     "Composer — cache",
		category: "PHP",
		resolvePath: func() string {
			if env := os.Getenv("COMPOSER_CACHE_DIR"); env != "" {
				return cleaner.Expand(env)
			}
			// XDG-style first (newer composer), then legacy ~/.composer/cache.
			xdg := filepath.Join(cleaner.XDGCacheHome(), "composer")
			if cleaner.IsDir(xdg) {
				return xdg
			}
			legacy := cleaner.Expand("~/.composer/cache")
			if cleaner.IsDir(legacy) {
				return legacy
			}
			// Mac sometimes uses ~/Library/Caches/composer.
			if runtime.GOOS == "darwin" {
				m := cleaner.Expand("~/Library/Caches/composer")
				if cleaner.IsDir(m) {
					return m
				}
			}
			return xdg // best-effort default
		},
		availableExtra: func(ctx context.Context) bool {
			return cleaner.LookPath("composer")
		},
		tool:     "composer",
		toolArgs: []string{"clear-cache"},
	}
}
