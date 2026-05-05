package desktop_apps

import (
	"context"
	"path/filepath"
	"runtime"

	"github.com/ariefsn/dust/internal/cleaner"
)

// All returns desktop-app cache cleaners (Slack, Discord, Spotify).
// Login state, message history, downloads — never touched.
func All() []cleaner.Cleaner {
	out := []cleaner.Cleaner{}
	out = append(out, slackCleaners()...)
	out = append(out, discordCleaners()...)
	out = append(out, spotifyCleaners()...)
	return out
}

// dirCleaner: minimal path-delete cleaner duplicated here to avoid a circular
// import on cleaners.pathBased. Kept identical in shape to editors.dirCleaner.
type dirCleaner struct {
	id, name, category, path string
}

func (d dirCleaner) ID() string                         { return d.id }
func (d dirCleaner) Name() string                       { return d.name }
func (d dirCleaner) Category() string                   { return d.category }
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

func slackRoot() string {
	switch runtime.GOOS {
	case "darwin":
		return cleaner.Expand("~/Library/Application Support/Slack")
	default:
		return filepath.Join(cleaner.XDGConfigHome(), "Slack")
	}
}

func slackCleaners() []cleaner.Cleaner {
	root := slackRoot()
	if !cleaner.IsDir(root) {
		return nil
	}
	mk := func(suffix, label, sub string) cleaner.Cleaner {
		return dirCleaner{
			id:       "desktop-apps/slack/" + suffix,
			name:     "Slack — " + label,
			category: "Desktop apps",
			path:     filepath.Join(root, sub),
		}
	}
	return []cleaner.Cleaner{
		mk("http-cache", "HTTP cache", filepath.Join("Cache", "Cache_Data")),
		mk("gpu-cache", "GPU cache", "GPUCache"),
		mk("code-cache", "Code cache", "Code Cache"),
		mk("service-workers", "Service workers", filepath.Join("Service Worker", "CacheStorage")),
	}
}

func discordRoot() string {
	switch runtime.GOOS {
	case "darwin":
		return cleaner.Expand("~/Library/Application Support/discord")
	default:
		return filepath.Join(cleaner.XDGConfigHome(), "discord")
	}
}

func discordCleaners() []cleaner.Cleaner {
	root := discordRoot()
	if !cleaner.IsDir(root) {
		return nil
	}
	mk := func(suffix, label, sub string) cleaner.Cleaner {
		return dirCleaner{
			id:       "desktop-apps/discord/" + suffix,
			name:     "Discord — " + label,
			category: "Desktop apps",
			path:     filepath.Join(root, sub),
		}
	}
	return []cleaner.Cleaner{
		mk("http-cache", "HTTP cache", filepath.Join("Cache", "Cache_Data")),
		mk("gpu-cache", "GPU cache", "GPUCache"),
		mk("code-cache", "Code cache", "Code Cache"),
	}
}

func spotifyCleaners() []cleaner.Cleaner {
	if runtime.GOOS != "darwin" {
		// Linux Spotify is usually flatpak/snap-managed; skip in v1.
		return nil
	}
	path := cleaner.Expand("~/Library/Caches/com.spotify.client")
	if !cleaner.IsDir(path) {
		return nil
	}
	return []cleaner.Cleaner{
		dirCleaner{
			id:       "desktop-apps/spotify",
			name:     "Spotify — cache",
			category: "Desktop apps",
			path:     path,
		},
	}
}
