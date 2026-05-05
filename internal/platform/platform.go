package platform

import "runtime"

type OS int

const (
	Unknown OS = iota
	Darwin
	Linux
	Windows
)

func (o OS) String() string {
	switch o {
	case Darwin:
		return "darwin"
	case Linux:
		return "linux"
	case Windows:
		return "windows"
	default:
		return "unknown"
	}
}

func Current() OS {
	switch runtime.GOOS {
	case "darwin":
		return Darwin
	case "linux":
		return Linux
	case "windows":
		return Windows
	default:
		return Unknown
	}
}

func IsDarwin() bool { return Current() == Darwin }
func IsLinux() bool  { return Current() == Linux }

// Supported reports whether dust runs on this OS in v1.
func Supported() bool {
	o := Current()
	return o == Darwin || o == Linux
}
