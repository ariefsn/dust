package cleaners

import (
	"context"
	"runtime"
	"strconv"
	"strings"

	"github.com/ariefsn/dust/internal/cleaner"
)

// Homebrew — `brew cleanup -s` removes old versions and clears the download cache.
type homebrew struct{}

func Homebrew() cleaner.Cleaner { return homebrew{} }

func (homebrew) ID() string       { return "homebrew" }
func (homebrew) Name() string     { return "Homebrew — cleanup" }
func (homebrew) Category() string { return "Homebrew" }

func (homebrew) Available(ctx context.Context) bool {
	return cleaner.LookPath("brew")
}

func (h homebrew) Scan(ctx context.Context) (cleaner.Result, error) {
	if !h.Available(ctx) {
		return cleaner.Result{}, nil
	}
	// `brew cleanup -n` reports "This operation would free approximately X" in
	// its last line. Parse that as the reclaimable size estimate.
	out, err := cleaner.RunCmd(ctx, "brew", "cleanup", "-n")
	if err != nil {
		// Even on failure, try to keep going with whatever output we got.
		return cleaner.Result{Path: brewCacheDir()}, nil
	}
	return cleaner.Result{Bytes: parseBrewFreed(out), Path: brewCacheDir()}, nil
}

func (h homebrew) Clean(ctx context.Context, opts cleaner.Options) (cleaner.Result, error) {
	if !h.Available(ctx) {
		return cleaner.Result{}, nil
	}
	before, _ := h.Scan(ctx)
	if opts.DryRun {
		return before, nil
	}
	if _, err := cleaner.RunCmd(ctx, "brew", "cleanup", "-s"); err != nil {
		return cleaner.Result{}, err
	}
	return before, nil
}

func brewCacheDir() string {
	if runtime.GOOS == "darwin" {
		return cleaner.Expand("~/Library/Caches/Homebrew")
	}
	return cleaner.Expand("~/.cache/Homebrew")
}

// parseBrewFreed extracts the byte size from a line like:
//
//	"This operation would free approximately 1.2GB of disk space."
func parseBrewFreed(out string) int64 {
	for _, line := range strings.Split(out, "\n") {
		l := strings.ToLower(line)
		if !strings.Contains(l, "free approximately") {
			continue
		}
		// Pull tokens after "approximately".
		idx := strings.Index(l, "approximately")
		rest := strings.TrimSpace(line[idx+len("approximately"):])
		// rest looks like "1.2GB of disk space." — strip trailing punctuation.
		rest = strings.TrimRight(rest, ".")
		fields := strings.Fields(rest)
		if len(fields) == 0 {
			continue
		}
		token := fields[0]
		// Split numeric prefix from unit suffix.
		i := 0
		for i < len(token) && (token[i] == '.' || (token[i] >= '0' && token[i] <= '9')) {
			i++
		}
		num, err := strconv.ParseFloat(token[:i], 64)
		if err != nil {
			continue
		}
		unit := strings.ToUpper(token[i:])
		return int64(num * unitBytes(unit))
	}
	return 0
}
