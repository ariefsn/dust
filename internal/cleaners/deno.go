package cleaners

import (
	"os"
	"runtime"

	"github.com/ariefsn/dust/internal/cleaner"
)

// Deno — wipe the deno dir (deps, npm cache, registries.json, etc.).
// Deno respects $DENO_DIR as an override. Default locations:
//   - macOS: ~/Library/Caches/deno
//   - Linux: ~/.cache/deno
func Deno() cleaner.Cleaner {
	return pathBased{
		id:       "deno",
		name:     "Deno — deno dir",
		category: "JS",
		resolvePath: func() string {
			if env := os.Getenv("DENO_DIR"); env != "" {
				return cleaner.Expand(env)
			}
			switch runtime.GOOS {
			case "darwin":
				return cleaner.Expand("~/Library/Caches/deno")
			default:
				return cleaner.Expand("~/.cache/deno")
			}
		},
	}
}
