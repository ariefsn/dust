package cleaners

import (
	"runtime"

	"github.com/ariefsn/dust/internal/cleaner"
)

// CocoaPods — wipe ~/Library/Caches/CocoaPods (download cache) and
// ~/.cocoapods/repos (Specs trunk repo). Trunk is re-fetched lazily on next
// `pod install`.
//
// We expose two cleaners so users can keep the trunk repo (it's slow to
// re-download) while clearing the much-larger downloads cache.
func CocoaPodsCache() cleaner.Cleaner {
	return pathBased{
		id:       "ios/cocoapods/cache",
		name:     "CocoaPods — download cache",
		category: "iOS",
		resolvePath: func() string {
			if runtime.GOOS != "darwin" {
				return ""
			}
			return cleaner.Expand("~/Library/Caches/CocoaPods")
		},
	}
}

func CocoaPodsRepos() cleaner.Cleaner {
	return pathBased{
		id:       "ios/cocoapods/repos",
		name:     "CocoaPods — Specs repos (~/.cocoapods/repos)",
		category: "iOS",
		resolvePath: func() string {
			if runtime.GOOS != "darwin" {
				return ""
			}
			return cleaner.Expand("~/.cocoapods/repos")
		},
	}
}

// Carthage — wipe ~/Library/Caches/org.carthage.CarthageKit. Carthage
// re-downloads dependencies on next `carthage update`.
func Carthage() cleaner.Cleaner {
	return pathBased{
		id:       "ios/carthage",
		name:     "Carthage — derived data cache",
		category: "iOS",
		resolvePath: func() string {
			if runtime.GOOS != "darwin" {
				return ""
			}
			return cleaner.Expand("~/Library/Caches/org.carthage.CarthageKit")
		},
	}
}
