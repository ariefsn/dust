package cleaners

import (
	"runtime"

	"github.com/ariefsn/dust/internal/cleaner"
)

// IOSBackups — `~/Library/Application Support/MobileSync/Backup`.
// In v1 this is a single aggregated cleaner. Per-backup selection is a future
// TUI enhancement once we surface the device-name + date metadata from
// each backup's `Info.plist`.
func IOSBackups() cleaner.Cleaner {
	return pathBased{
		id:       "ios/backups",
		name:     "iOS device backups (confirm-twice)",
		category: "iOS",
		resolvePath: func() string {
			if runtime.GOOS != "darwin" {
				return ""
			}
			return cleaner.Expand("~/Library/Application Support/MobileSync/Backup")
		},
	}
}
