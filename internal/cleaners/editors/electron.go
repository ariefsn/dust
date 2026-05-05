package editors

import (
	"path/filepath"
	"runtime"

	"github.com/ariefsn/dust/internal/cleaner"
)

// Electron editors store caches the same way Chromium browsers do (they're all
// Electron). We expose the same 5 sub-items per detected app.
//
// Detection: scan ~/Library/Application Support/ (darwin) or ~/.config/ (linux)
// for known dirnames. Only emit cleaners for apps that exist.

var electronApps = []electronApp{
	{label: "VS Code", darwinDir: "Code", linuxDir: "Code"},
	{label: "VS Code Insiders", darwinDir: "Code - Insiders", linuxDir: "Code - Insiders"},
	{label: "Cursor", darwinDir: "Cursor", linuxDir: "Cursor"},
	{label: "Windsurf", darwinDir: "Windsurf", linuxDir: "Windsurf"},
	{label: "VSCodium", darwinDir: "VSCodium", linuxDir: "VSCodium"},
}

type electronApp struct {
	label              string
	darwinDir, linuxDir string
}

func (e electronApp) profileRoot() string {
	switch runtime.GOOS {
	case "darwin":
		return cleaner.Expand(filepath.Join("~/Library/Application Support", e.darwinDir))
	default:
		return filepath.Join(cleaner.XDGConfigHome(), e.linuxDir)
	}
}

func (e electronApp) idPrefix() string {
	id := e.label
	for _, c := range []string{" ", "—", "–"} {
		for {
			i := -1
			for j := 0; j+len(c) <= len(id); j++ {
				if id[j:j+len(c)] == c {
					i = j
					break
				}
			}
			if i < 0 {
				break
			}
			id = id[:i] + "-" + id[i+len(c):]
		}
	}
	return "editors/" + lower(id)
}

// lower is an inlined ASCII lowercaser to avoid pulling unicode in here.
func lower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		}
	}
	return string(b)
}

// ElectronCleaners returns one set of cache sub-items per detected Electron app.
func ElectronCleaners() []cleaner.Cleaner {
	var out []cleaner.Cleaner
	for _, app := range electronApps {
		root := app.profileRoot()
		if !cleaner.IsDir(root) {
			continue
		}
		idPrefix := app.idPrefix()
		mk := func(suffix, label, sub string) cleaner.Cleaner {
			return dirCleaner{
				id:       idPrefix + "/" + suffix,
				name:     app.label + " — " + label,
				category: "Editors",
				path:     filepath.Join(root, sub),
			}
		}
		// Electron caches live at the app root (not per-profile).
		out = append(out,
			mk("http-cache", "HTTP cache", filepath.Join("Cache", "Cache_Data")),
			mk("gpu-cache", "GPU cache", "GPUCache"),
			mk("code-cache", "Code cache", "Code Cache"),
			mk("service-workers", "Service workers", filepath.Join("Service Worker", "CacheStorage")),
			mk("cached-extensions", "CachedExtensions", "CachedExtensions"),
			mk("crashpad", "Crash reports", "Crashpad"),
		)
	}
	return out
}
