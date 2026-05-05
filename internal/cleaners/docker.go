package cleaners

import (
	"context"
	"strconv"
	"strings"

	"github.com/ariefsn/dust/internal/cleaner"
)

type docker struct{}

func Docker() cleaner.Cleaner { return docker{} }

func (docker) ID() string       { return "docker" }
func (docker) Name() string     { return "Docker — system prune" }
func (docker) Category() string { return "Containers" }

func (docker) Available(ctx context.Context) bool {
	if !cleaner.LookPath("docker") {
		return false
	}
	// Daemon must be reachable; `docker info` exits non-zero when it's not.
	_, err := cleaner.RunCmd(ctx, "docker", "info", "--format", "{{.ServerVersion}}")
	return err == nil
}

// Scan reports the reclaimable bytes from `docker system df`.
func (d docker) Scan(ctx context.Context) (cleaner.Result, error) {
	if !d.Available(ctx) {
		return cleaner.Result{}, nil
	}
	out, err := cleaner.RunCmd(ctx, "docker", "system", "df", "--format", "{{.Type}}\t{{.Reclaimable}}")
	if err != nil {
		return cleaner.Result{}, err
	}
	var total int64
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}
		total += parseReclaimable(parts[1])
	}
	return cleaner.Result{Bytes: total, Path: "docker daemon"}, nil
}

func (d docker) Clean(ctx context.Context, opts cleaner.Options) (cleaner.Result, error) {
	if !d.Available(ctx) {
		return cleaner.Result{}, nil
	}
	before, err := d.Scan(ctx)
	if err != nil {
		return cleaner.Result{}, err
	}
	if opts.DryRun {
		return cleaner.Result{Bytes: before.Bytes, Path: "docker daemon"}, nil
	}
	args := []string{"system", "prune", "-f"}
	if _, err := cleaner.RunCmd(ctx, "docker", args...); err != nil {
		return cleaner.Result{}, err
	}
	after, _ := d.Scan(ctx)
	freed := before.Bytes - after.Bytes
	if freed < 0 {
		freed = 0
	}
	return cleaner.Result{Bytes: freed, Path: "docker daemon"}, nil
}

// parseReclaimable extracts the size from strings like "1.234GB (75%)" or "523MB".
func parseReclaimable(s string) int64 {
	s = strings.TrimSpace(s)
	if i := strings.Index(s, "("); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	// Trailing unit. Walk backwards over letters.
	i := len(s)
	for i > 0 && isUnitChar(s[i-1]) {
		i--
	}
	num := strings.TrimSpace(s[:i])
	unit := strings.ToUpper(strings.TrimSpace(s[i:]))
	v, err := strconv.ParseFloat(num, 64)
	if err != nil {
		return 0
	}
	return int64(v * unitBytes(unit))
}

func isUnitChar(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z')
}

func unitBytes(u string) float64 {
	switch u {
	case "B", "":
		return 1
	case "KB":
		return 1_000
	case "MB":
		return 1_000_000
	case "GB":
		return 1_000_000_000
	case "TB":
		return 1_000_000_000_000
	case "KIB":
		return 1024
	case "MIB":
		return 1024 * 1024
	case "GIB":
		return 1024 * 1024 * 1024
	case "TIB":
		return 1024 * 1024 * 1024 * 1024
	}
	return 1
}
