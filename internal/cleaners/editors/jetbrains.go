package editors

import (
	"context"
	"path/filepath"
	"runtime"

	"github.com/ariefsn/dust/internal/cleaner"
)

// JetBrainsCleaners enumerates ~/Library/Caches/JetBrains/<IDE><version> and
// returns one cleaner per detected install. Settings under
// ~/Library/Application Support/JetBrains/* are never touched.
func JetBrainsCleaners() []cleaner.Cleaner {
	var out []cleaner.Cleaner
	for _, dir := range globDirs(jetbrainsCachesRoot(), "*") {
		base := filepath.Base(dir)
		out = append(out, dirCleaner{
			id:       "editors/jetbrains/" + base,
			name:     "JetBrains — " + base,
			category: "Editors",
			path:     dir,
		})
	}
	if logsRoot := jetbrainsLogsRoot(); logsRoot != "" {
		for _, dir := range globDirs(logsRoot, "*") {
			base := filepath.Base(dir)
			out = append(out, dirCleaner{
				id:       "editors/jetbrains/" + base + "-logs",
				name:     "JetBrains — " + base + " (logs)",
				category: "Editors",
				path:     dir,
			})
		}
	}
	return out
}

func jetbrainsCachesRoot() string {
	switch runtime.GOOS {
	case "darwin":
		return cleaner.Expand("~/Library/Caches/JetBrains")
	default:
		return filepath.Join(cleaner.XDGCacheHome(), "JetBrains")
	}
}

func jetbrainsLogsRoot() string {
	if runtime.GOOS == "darwin" {
		return cleaner.Expand("~/Library/Logs/JetBrains")
	}
	return ""
}

// globDirs returns the directories matching pattern under root.
func globDirs(root, pattern string) []string {
	matches, _ := filepath.Glob(filepath.Join(root, pattern))
	var out []string
	for _, m := range matches {
		if cleaner.IsDir(m) {
			out = append(out, m)
		}
	}
	return out
}

// dirCleaner is a minimal path-based cleaner shared across editor & desktop_app
// packages. (We can't reuse cleaners.pathBased here because that would import
// the parent package.)
type dirCleaner struct {
	id, name, category, path string
}

func (d dirCleaner) ID() string                       { return d.id }
func (d dirCleaner) Name() string                     { return d.name }
func (d dirCleaner) Category() string                 { return d.category }
func (d dirCleaner) Available(ctx context.Context) bool { return cleaner.IsDir(d.path) }

func (d dirCleaner) Scan(ctx context.Context) (cleaner.Result, error) {
	bytes, items, err := cleaner.DirSize(ctx, d.path)
	return cleaner.Result{Bytes: bytes, Items: items, Path: d.path}, err
}

func (d dirCleaner) Clean(ctx context.Context, opts cleaner.Options) (cleaner.Result, error) {
	before, _ := d.Scan(ctx)
	if opts.DryRun {
		return before, nil
	}
	if err := cleaner.SafeRemoveAll(d.path, []string{cleaner.Home()}); err != nil {
		return cleaner.Result{}, err
	}
	return before, nil
}
