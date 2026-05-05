package cleaner

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func Home() string {
	if h, err := os.UserHomeDir(); err == nil {
		return h
	}
	return os.Getenv("HOME")
}

func Expand(p string) string {
	if p == "" {
		return p
	}
	if p == "~" {
		return Home()
	}
	if strings.HasPrefix(p, "~/") {
		return filepath.Join(Home(), p[2:])
	}
	return os.ExpandEnv(p)
}

func XDGCacheHome() string {
	if v := os.Getenv("XDG_CACHE_HOME"); v != "" {
		return v
	}
	return filepath.Join(Home(), ".cache")
}

func XDGConfigHome() string {
	if v := os.Getenv("XDG_CONFIG_HOME"); v != "" {
		return v
	}
	return filepath.Join(Home(), ".config")
}

func XDGDataHome() string {
	if v := os.Getenv("XDG_DATA_HOME"); v != "" {
		return v
	}
	return filepath.Join(Home(), ".local", "share")
}

// Exists reports whether the path exists (file or dir).
func Exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// IsDir reports whether the path exists and is a directory.
func IsDir(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}

// AndroidSDKRoot returns ($ANDROID_HOME → $ANDROID_SDK_ROOT → default), and whether the path exists.
func AndroidSDKRoot() (string, bool) {
	if p := os.Getenv("ANDROID_HOME"); p != "" {
		return p, IsDir(p)
	}
	if p := os.Getenv("ANDROID_SDK_ROOT"); p != "" {
		return p, IsDir(p)
	}
	var def string
	switch runtime.GOOS {
	case "darwin":
		def = filepath.Join(Home(), "Library", "Android", "sdk")
	case "linux":
		def = filepath.Join(Home(), "Android", "Sdk")
	default:
		return "", false
	}
	return def, IsDir(def)
}

// AndroidUserHome returns $ANDROID_USER_HOME or ~/.android.
func AndroidUserHome() string {
	if p := os.Getenv("ANDROID_USER_HOME"); p != "" {
		return p
	}
	return filepath.Join(Home(), ".android")
}

// AndroidAVDHome returns $ANDROID_AVD_HOME or <user_home>/avd.
func AndroidAVDHome() string {
	if p := os.Getenv("ANDROID_AVD_HOME"); p != "" {
		return p
	}
	return filepath.Join(AndroidUserHome(), "avd")
}
