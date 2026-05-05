package cleaners

import (
	"context"
	"runtime"

	"github.com/ariefsn/dust/internal/cleaner"
)

// Pip — `pip cache purge` if available, else wipe the OS-specific cache dir.
func Pip() cleaner.Cleaner {
	return pathBased{
		id:       "python/pip",
		name:     "pip — cache",
		category: "Python",
		resolvePath: func() string {
			switch runtime.GOOS {
			case "darwin":
				return cleaner.Expand("~/Library/Caches/pip")
			default:
				return cleaner.Expand("~/.cache/pip")
			}
		},
		availableExtra: func(ctx context.Context) bool {
			return cleaner.LookPath("pip") || cleaner.LookPath("pip3")
		},
		tool:     pipTool(),
		toolArgs: []string{"cache", "purge"},
	}
}

func pipTool() string {
	if cleaner.LookPath("pip3") {
		return "pip3"
	}
	return "pip"
}

// Conda — `conda clean -afy` if conda is on PATH, else best-effort path-delete
// of the pkgs/ subdirs in any well-known conda install root. Conda's pkgs
// folder is the bulky bit; envs/ holds user environments and is intentionally
// not touched.
func Conda() cleaner.Cleaner {
	return pathBased{
		id:       "python/conda",
		name:     "Conda — package cache (pkgs)",
		category: "Python",
		resolvePath: func() string {
			candidates := []string{
				"~/miniconda3/pkgs",
				"~/anaconda3/pkgs",
				"~/miniforge3/pkgs",
				"~/mambaforge/pkgs",
			}
			for _, c := range candidates {
				p := cleaner.Expand(c)
				if cleaner.IsDir(p) {
					return p
				}
			}
			return cleaner.Expand("~/miniconda3/pkgs")
		},
		availableExtra: func(ctx context.Context) bool {
			return cleaner.LookPath("conda") || cleaner.LookPath("mamba")
		},
		tool:     "conda",
		toolArgs: []string{"clean", "-afy"},
	}
}
