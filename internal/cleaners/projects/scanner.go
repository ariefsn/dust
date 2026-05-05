package projects

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ariefsn/dust/internal/cleaner"
)

// Project is a single detected project with one or more matched kinds.
type Project struct {
	Path        string    // absolute project root
	Kinds       []Kind    // every matched kind (e.g. Flutter + CocoaPods)
	Bytes       int64     // total bytes of all artifact dirs
	Items       int       // file count across artifact dirs
	LastTouched time.Time // most recent mtime among non-skip files
	Dirty       bool      // git working tree has uncommitted changes
	HasGit      bool
}

// Config tunes the walk.
type Config struct {
	Roots     []string
	MaxDepth  int  // default 8
	StaleDays int  // 0 = no staleness filter
	SkipDirty bool // skip projects with uncommitted git changes
}

// AutoRoots returns the set of common project roots that exist on the user's
// home dir. Used when the user hasn't configured `roots` in config.yaml.
func AutoRoots() []string {
	candidates := []string{
		"~/Projects", "~/Work", "~/Code", "~/code",
		"~/dev", "~/src", "~/repos", "~/Documents/Projects",
	}
	var out []string
	for _, c := range candidates {
		p := cleaner.Expand(c)
		if cleaner.IsDir(p) {
			out = append(out, p)
		}
	}
	return out
}

// hardSkipDirs are directory names we never descend into during the walk.
// They're build outputs or VCS dirs — finding `package.json` inside
// `node_modules/some-pkg` is never useful.
var hardSkipDirs = map[string]bool{
	".git":           true,
	"node_modules":   true,
	"vendor":         true,
	"target":         true,
	".venv":          true,
	"venv":           true,
	"__pycache__":    true,
	".dart_tool":     true,
	"Pods":           true,
	"DerivedData":    true,
	"build":          true,
	"dist":           true,
	".next":          true,
	".turbo":         true,
	".nuxt":          true,
	".svelte-kit":    true,
	".parcel-cache":  true,
	".gradle":        true,
	".cargo":         true,
	"bin":            true,
	"obj":            true,
}

// Scan walks cfg.Roots and returns every detected project, with per-project
// size + last-touched metadata computed concurrently.
func Scan(ctx context.Context, cfg Config) ([]Project, error) {
	if cfg.MaxDepth <= 0 {
		cfg.MaxDepth = 8
	}
	if len(cfg.Roots) == 0 {
		cfg.Roots = AutoRoots()
	}

	var projectPaths []string
	var pathsMu sync.Mutex
	var walkWG sync.WaitGroup
	for _, root := range cfg.Roots {
		root := root
		walkWG.Add(1)
		go func() {
			defer walkWG.Done()
			ps := walkRoot(ctx, root, cfg.MaxDepth)
			pathsMu.Lock()
			projectPaths = append(projectPaths, ps...)
			pathsMu.Unlock()
		}()
	}
	walkWG.Wait()

	// Build Project structs, computing size + mtime concurrently.
	results := make([]Project, len(projectPaths))
	sem := make(chan struct{}, runtime.NumCPU())
	var sizeWG sync.WaitGroup
	for i, p := range projectPaths {
		i, p := i, p
		sizeWG.Add(1)
		sem <- struct{}{}
		go func() {
			defer sizeWG.Done()
			defer func() { <-sem }()
			results[i] = enrichProject(ctx, p)
		}()
	}
	sizeWG.Wait()

	// Apply filters.
	cutoff := time.Time{}
	if cfg.StaleDays > 0 {
		cutoff = time.Now().AddDate(0, 0, -cfg.StaleDays)
	}
	out := make([]Project, 0, len(results))
	for _, p := range results {
		if cfg.SkipDirty && p.Dirty {
			continue
		}
		if !cutoff.IsZero() && p.LastTouched.After(cutoff) {
			continue
		}
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Bytes != out[j].Bytes {
			return out[i].Bytes > out[j].Bytes
		}
		return out[i].Path < out[j].Path
	})
	return out, nil
}

// walkRoot descends from `root` up to `maxDepth` levels, returning every dir
// that contains a recognized manifest. Stops descending at any matched dir.
func walkRoot(ctx context.Context, root string, maxDepth int) []string {
	var out []string
	rootDepth := strings.Count(filepath.Clean(root), string(filepath.Separator))

	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil {
			return nil // tolerate permission errors
		}
		if !d.IsDir() {
			return nil
		}
		// Depth guard.
		curDepth := strings.Count(filepath.Clean(path), string(filepath.Separator)) - rootDepth
		if curDepth > maxDepth {
			return filepath.SkipDir
		}
		// Hard skip set.
		base := d.Name()
		if hardSkipDirs[base] {
			return filepath.SkipDir
		}
		// Hidden dirs except an allowlist.
		if curDepth > 0 && len(base) > 1 && base[0] == '.' {
			if !hiddenAllowed[base] {
				return filepath.SkipDir
			}
		}
		// Manifest detection.
		kinds := detectKinds(path)
		if len(kinds) > 0 {
			out = append(out, path)
			return filepath.SkipDir // stop descending into a matched project
		}
		return nil
	})
	return out
}

// hiddenAllowed are hidden directories worth descending into.
var hiddenAllowed = map[string]bool{
	".github": true,
	".vscode": true,
	".idea":   true,
}

// enrichProject computes size, last-touched, and git-dirty metadata.
func enrichProject(ctx context.Context, root string) Project {
	p := Project{
		Path:   root,
		Kinds:  detectKinds(root),
		HasGit: hasGit(root),
	}
	if p.HasGit {
		p.Dirty = isDirty(ctx, root)
	}

	// Aggregate size across every artifact dir referenced by the matched kinds.
	seen := make(map[string]bool)
	for _, k := range p.Kinds {
		for _, art := range k.Artifacts {
			matches := resolveArtifact(root, art)
			for _, m := range matches {
				if seen[m] {
					continue
				}
				seen[m] = true
				b, n, err := cleaner.DirSize(ctx, m)
				if err == nil {
					p.Bytes += b
					p.Items += n
				}
			}
		}
	}

	// Last-touched: most recent mtime among files in the project root,
	// excluding hard-skip dirs (so build outputs don't poison the signal).
	p.LastTouched = newestSourceMtime(ctx, root)

	return p
}

// resolveArtifact handles the `*.egg-info` glob style and slash-bearing names
// like "ios/Pods", returning every concrete path that exists.
func resolveArtifact(root, pattern string) []string {
	if !strings.ContainsAny(pattern, "*?[") {
		full := filepath.Join(root, pattern)
		if _, err := os.Stat(full); err == nil {
			return []string{full}
		}
		return nil
	}
	matches, _ := filepath.Glob(filepath.Join(root, pattern))
	return matches
}

// newestSourceMtime returns the most recent mtime among files under `root`,
// skipping hardSkipDirs and hidden dirs. Used as the staleness signal so
// build artifacts don't make a project look "active".
func newestSourceMtime(ctx context.Context, root string) time.Time {
	var newest time.Time
	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil {
			return nil
		}
		base := d.Name()
		if d.IsDir() {
			if path == root {
				return nil
			}
			if hardSkipDirs[base] || (len(base) > 1 && base[0] == '.' && !hiddenAllowed[base]) {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		if info.ModTime().After(newest) {
			newest = info.ModTime()
		}
		return nil
	})
	return newest
}
