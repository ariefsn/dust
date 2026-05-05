package cleaners

import (
	"context"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ariefsn/dust/internal/cleaner"
)

// AndroidStaticCleaners returns the cleaners that don't depend on enumerating
// the filesystem at registration time (the AVD/SDK enumerators run lazily).
func AndroidStaticCleaners() []cleaner.Cleaner {
	expand := cleaner.Expand

	// Android Studio cache/log roots. Android Studio is JetBrains-based but uses
	// Google-branded paths. Both old (Caches/Google/AndroidStudio*) and newer
	// (Caches/JetBrains/AndroidStudio*) layouts exist depending on version.
	studioCachePaths := func() []string {
		var globs []string
		switch runtime.GOOS {
		case "darwin":
			globs = []string{
				expand("~/Library/Caches/Google/AndroidStudio*"),
				expand("~/Library/Caches/JetBrains/AndroidStudio*"),
				expand("~/Library/Logs/Google/AndroidStudio*"),
				expand("~/Library/Logs/JetBrains/AndroidStudio*"),
			}
		default:
			globs = []string{
				expand("~/.cache/Google/AndroidStudio*"),
				expand("~/.cache/JetBrains/AndroidStudio*"),
			}
		}
		var out []string
		for _, g := range globs {
			matches, _ := filepath.Glob(g)
			out = append(out, matches...)
		}
		return out
	}

	// Studio install marker — proves the app exists even when cache/log dirs
	// don't (fresh install, never launched, or already cleaned).
	studioInstalled := func() bool {
		if runtime.GOOS != "darwin" {
			matches, _ := filepath.Glob(expand("~/.config/Google/AndroidStudio*"))
			if len(matches) > 0 {
				return true
			}
			return false
		}
		matches, _ := filepath.Glob(expand("~/Library/Application Support/Google/AndroidStudio*"))
		if len(matches) > 0 {
			return true
		}
		matches, _ = filepath.Glob(expand("~/Library/Application Support/JetBrains/AndroidStudio*"))
		return len(matches) > 0
	}

	return []cleaner.Cleaner{
		multiPath{
			id:       "android/studio-caches",
			name:     "Android Studio — caches & logs",
			category: "Android",
			paths:    studioCachePaths,
			availableExtra: func(ctx context.Context) bool {
				return studioInstalled()
			},
		},
		pathBased{
			id:       "android/tracecaches",
			name:     "Android — ~/.android/cache",
			category: "Android",
			resolvePath: func() string {
				return filepath.Join(cleaner.AndroidUserHome(), "cache")
			},
		},
		avdSnapshots{},
	}
}

// multiPath wipes (or sums) the contents of a list of paths produced by a glob.
type multiPath struct {
	id, name, category string
	paths              func() []string
	// availableExtra reports availability when the cache paths don't exist
	// (e.g., the app is installed but its caches were already cleaned).
	availableExtra func(ctx context.Context) bool
}

func (m multiPath) ID() string       { return m.id }
func (m multiPath) Name() string     { return m.name }
func (m multiPath) Category() string { return m.category }

func (m multiPath) Available(ctx context.Context) bool {
	if len(m.paths()) > 0 {
		return true
	}
	if m.availableExtra != nil {
		return m.availableExtra(ctx)
	}
	return false
}

func (m multiPath) Scan(ctx context.Context) (cleaner.Result, error) {
	var total int64
	var items int
	paths := m.paths()
	for _, p := range paths {
		b, n, err := cleaner.DirSize(ctx, p)
		if err != nil {
			return cleaner.Result{}, err
		}
		total += b
		items += n
	}
	return cleaner.Result{Bytes: total, Items: items, Path: strings.Join(paths, ", ")}, nil
}

func (m multiPath) Clean(ctx context.Context, opts cleaner.Options) (cleaner.Result, error) {
	before, _ := m.Scan(ctx)
	if opts.DryRun {
		return before, nil
	}
	for _, p := range m.paths() {
		if err := cleaner.SafeRemoveAll(p, []string{cleaner.Home()}); err != nil {
			return cleaner.Result{}, err
		}
	}
	return before, nil
}

// avdSnapshots reclaims the snapshots subdir of every AVD. Skips userdata to
// avoid resetting emulator state.
type avdSnapshots struct{}

func (avdSnapshots) ID() string       { return "android/avd-snapshots" }
func (avdSnapshots) Name() string     { return "Android — AVD snapshots (all AVDs)" }
func (avdSnapshots) Category() string { return "Android" }

func (avdSnapshots) Available(ctx context.Context) bool {
	matches, _ := filepath.Glob(filepath.Join(cleaner.AndroidAVDHome(), "*.avd"))
	for _, avd := range matches {
		if cleaner.IsDir(filepath.Join(avd, "snapshots")) {
			return true
		}
	}
	return false
}

func (a avdSnapshots) snapshotDirs() []string {
	matches, _ := filepath.Glob(filepath.Join(cleaner.AndroidAVDHome(), "*.avd"))
	var out []string
	for _, avd := range matches {
		snap := filepath.Join(avd, "snapshots")
		if cleaner.IsDir(snap) {
			out = append(out, snap)
		}
	}
	return out
}

func (a avdSnapshots) Scan(ctx context.Context) (cleaner.Result, error) {
	var total int64
	dirs := a.snapshotDirs()
	for _, d := range dirs {
		b, _, err := cleaner.DirSize(ctx, d)
		if err != nil {
			return cleaner.Result{}, err
		}
		total += b
	}
	return cleaner.Result{Bytes: total, Path: strings.Join(dirs, ", ")}, nil
}

func (a avdSnapshots) Clean(ctx context.Context, opts cleaner.Options) (cleaner.Result, error) {
	before, _ := a.Scan(ctx)
	if opts.DryRun {
		return before, nil
	}
	for _, d := range a.snapshotDirs() {
		if err := cleaner.SafeRemoveAll(d, []string{cleaner.Home()}); err != nil {
			return cleaner.Result{}, err
		}
	}
	return before, nil
}
