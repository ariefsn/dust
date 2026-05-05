package projects

import (
	"context"
	"strings"

	"github.com/ariefsn/dust/internal/cleaner"
)

// AsCleaner adapts a Project into the Cleaner interface so it slots into
// the existing TUI/scan/clean pipelines. The adapter snapshots the size
// reported by Scan so subsequent UI reads are cheap; the actual deletion
// runs through CleanProject.
type AsCleaner struct {
	P          Project
	PreferTool bool
}

func (a AsCleaner) ID() string {
	return "projects/" + a.P.Path
}

func (a AsCleaner) Name() string {
	kinds := make([]string, 0, len(a.P.Kinds))
	for _, k := range a.P.Kinds {
		kinds = append(kinds, k.Name)
	}
	label := strings.Join(kinds, "+")
	if label == "" {
		label = "?"
	}
	return shortenHome(a.P.Path) + " (" + label + ")"
}

func (a AsCleaner) Category() string { return "Projects" }

func (a AsCleaner) Available(ctx context.Context) bool {
	return a.P.Bytes > 0
}

func (a AsCleaner) Scan(ctx context.Context) (cleaner.Result, error) {
	return cleaner.Result{
		Bytes: a.P.Bytes,
		Items: a.P.Items,
		Path:  a.P.Path,
	}, nil
}

func (a AsCleaner) Clean(ctx context.Context, opts cleaner.Options) (cleaner.Result, error) {
	bytes, err := CleanProject(ctx, a.P, a.PreferTool, opts.DryRun)
	if err != nil {
		return cleaner.Result{}, err
	}
	return cleaner.Result{Bytes: bytes, Path: a.P.Path}, nil
}

// shortenHome replaces a $HOME prefix with "~" for display.
func shortenHome(p string) string {
	home := cleaner.Home()
	if home == "" {
		return p
	}
	if strings.HasPrefix(p, home) {
		return "~" + p[len(home):]
	}
	return p
}
