package browsers

import (
	"runtime"

	"github.com/ariefsn/dust/internal/cleaner"
)

func ChromeCleaners() []cleaner.Cleaner {
	root := func() string {
		switch runtime.GOOS {
		case "darwin":
			return cleaner.Expand("~/Library/Application Support/Google/Chrome")
		default:
			return cleaner.Expand("~/.config/google-chrome")
		}
	}
	return chromiumFamily("browsers/chrome", "Chrome", root)
}

func BraveCleaners() []cleaner.Cleaner {
	root := func() string {
		switch runtime.GOOS {
		case "darwin":
			return cleaner.Expand("~/Library/Application Support/BraveSoftware/Brave-Browser")
		default:
			return cleaner.Expand("~/.config/BraveSoftware/Brave-Browser")
		}
	}
	return chromiumFamily("browsers/brave", "Brave", root)
}

func EdgeCleaners() []cleaner.Cleaner {
	root := func() string {
		switch runtime.GOOS {
		case "darwin":
			return cleaner.Expand("~/Library/Application Support/Microsoft Edge")
		default:
			return cleaner.Expand("~/.config/microsoft-edge")
		}
	}
	return chromiumFamily("browsers/edge", "Edge", root)
}

func ArcCleaners() []cleaner.Cleaner {
	if runtime.GOOS != "darwin" {
		return nil
	}
	root := func() string {
		return cleaner.Expand("~/Library/Application Support/Arc/User Data")
	}
	return chromiumFamily("browsers/arc", "Arc", root)
}

func OperaCleaners() []cleaner.Cleaner {
	root := func() string {
		switch runtime.GOOS {
		case "darwin":
			return cleaner.Expand("~/Library/Application Support/com.operasoftware.Opera")
		default:
			return cleaner.Expand("~/.config/opera")
		}
	}
	return chromiumFamily("browsers/opera", "Opera", root)
}

func ChromiumCleaners() []cleaner.Cleaner {
	root := func() string {
		switch runtime.GOOS {
		case "darwin":
			return cleaner.Expand("~/Library/Application Support/Chromium")
		default:
			return cleaner.Expand("~/.config/chromium")
		}
	}
	return chromiumFamily("browsers/chromium", "Chromium", root)
}
