package projects

import (
	"os"
	"path/filepath"
	"strings"
)

// detectKinds returns the Kinds that match the manifests in `dir`.
// A single project may match multiple kinds (e.g. Flutter + CocoaPods).
func detectKinds(dir string) []Kind {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	names := make(map[string]bool, len(entries))
	for _, e := range entries {
		names[e.Name()] = true
	}

	var matched []Kind
	for _, k := range AllKinds {
		if matchesAny(names, k.Manifests) {
			matched = append(matched, k)
		}
	}
	return matched
}

// matchesAny reports whether any name in `names` matches any pattern.
// Patterns containing `*` are matched with filepath.Match; literal patterns
// are checked by direct map lookup.
func matchesAny(names map[string]bool, patterns []string) bool {
	for _, p := range patterns {
		if !strings.ContainsAny(p, "*?[") {
			if names[p] {
				return true
			}
			continue
		}
		for n := range names {
			ok, _ := filepath.Match(p, n)
			if ok {
				return true
			}
		}
	}
	return false
}
