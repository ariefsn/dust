package cleaners

import (
	"context"
	"os"

	"github.com/ariefsn/dust/internal/cleaner"
)

// NuGet — `dotnet nuget locals all --clear` if dotnet is available, else wipe
// the package cache directory.
//
// NuGet packages live in:
//   - $NUGET_PACKAGES (override, used by some monorepos)
//   - ~/.nuget/packages (default everywhere)
func NuGet() cleaner.Cleaner {
	return pathBased{
		id:       "dotnet/nuget",
		name:     "NuGet — packages cache",
		category: ".NET",
		resolvePath: func() string {
			if env := os.Getenv("NUGET_PACKAGES"); env != "" {
				return cleaner.Expand(env)
			}
			return cleaner.Expand("~/.nuget/packages")
		},
		availableExtra: func(ctx context.Context) bool {
			return cleaner.LookPath("dotnet")
		},
		tool:     "dotnet",
		toolArgs: []string{"nuget", "locals", "all", "--clear"},
	}
}
