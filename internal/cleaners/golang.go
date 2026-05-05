package cleaners

import (
	"context"
	"runtime"

	"github.com/ariefsn/dust/internal/cleaner"
)

// GoModCache — `go clean -modcache` if `go` is on PATH, else wipe ~/go/pkg/mod.
func GoModCache() cleaner.Cleaner {
	return pathBased{
		id:       "go/modcache",
		name:     "Go — module cache",
		category: "Go",
		resolvePath: func() string {
			return cleaner.Expand("~/go/pkg/mod")
		},
		availableExtra: func(ctx context.Context) bool {
			return cleaner.LookPath("go")
		},
		tool:     "go",
		toolArgs: []string{"clean", "-modcache"},
	}
}

// GoBuildCache — `go clean -cache` if available, else wipe the per-OS build cache dir.
func GoBuildCache() cleaner.Cleaner {
	return pathBased{
		id:       "go/buildcache",
		name:     "Go — build cache",
		category: "Go",
		resolvePath: func() string {
			switch runtime.GOOS {
			case "darwin":
				return cleaner.Expand("~/Library/Caches/go-build")
			default:
				return cleaner.Expand("~/.cache/go-build")
			}
		},
		availableExtra: func(ctx context.Context) bool {
			return cleaner.LookPath("go")
		},
		tool:     "go",
		toolArgs: []string{"clean", "-cache"},
	}
}
