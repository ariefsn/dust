package cleaners

import (
	"os"

	"github.com/ariefsn/dust/internal/cleaner"
)

// FlutterPubCache — ~/.pub-cache. Holds every Dart package version any
// project on this machine has ever depended on; can grow to multiple GB
// after a year of Flutter work.
//
// Pub respects $PUB_CACHE as an override. We don't ship a tool action because
// `dart pub cache repair` only repairs entries — there's no global "clear"
// command, so a path-delete is the equivalent. Pub re-downloads on next
// `flutter pub get`.
func FlutterPubCache() cleaner.Cleaner {
	return pathBased{
		id:       "flutter/pub-cache",
		name:     "Flutter / Dart — pub cache",
		category: "Flutter",
		resolvePath: func() string {
			if env := os.Getenv("PUB_CACHE"); env != "" {
				return cleaner.Expand(env)
			}
			return cleaner.Expand("~/.pub-cache")
		},
	}
}
