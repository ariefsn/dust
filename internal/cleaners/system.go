package cleaners

import (
	"context"
	"path/filepath"
	"runtime"

	"github.com/ariefsn/dust/internal/cleaner"
)

// SystemCleaners returns OS-managed preview cache cleaners (Linux thumbnails,
// Mac Quick Look). Safe — these regenerate on demand.
func SystemCleaners() []cleaner.Cleaner {
	switch runtime.GOOS {
	case "linux":
		root := filepath.Join(cleaner.XDGCacheHome(), "thumbnails")
		out := []cleaner.Cleaner{
			pathBased{
				id:       "system/thumbnails/normal",
				name:     "Thumbnails — normal (128px)",
				category: "System",
				resolvePath: func() string {
					return filepath.Join(root, "normal")
				},
			},
			pathBased{
				id:       "system/thumbnails/large",
				name:     "Thumbnails — large (256px)",
				category: "System",
				resolvePath: func() string {
					return filepath.Join(root, "large")
				},
			},
			pathBased{
				id:       "system/thumbnails/x-large",
				name:     "Thumbnails — x-large (512px)",
				category: "System",
				resolvePath: func() string {
					return filepath.Join(root, "x-large")
				},
			},
			pathBased{
				id:       "system/thumbnails/xx-large",
				name:     "Thumbnails — xx-large (1024px)",
				category: "System",
				resolvePath: func() string {
					return filepath.Join(root, "xx-large")
				},
			},
			pathBased{
				id:       "system/thumbnails/fail",
				name:     "Thumbnails — failed",
				category: "System",
				resolvePath: func() string {
					return filepath.Join(root, "fail")
				},
			},
			pathBased{
				id:       "system/thumbnails-legacy",
				name:     "Thumbnails — legacy ~/.thumbnails",
				category: "System",
				resolvePath: func() string {
					return cleaner.Expand("~/.thumbnails")
				},
			},
			pathBased{
				id:       "system/tumbler",
				name:     "Thumbnails — Thunar tumbler",
				category: "System",
				resolvePath: func() string {
					return filepath.Join(cleaner.XDGCacheHome(), "tumbler")
				},
			},
		}
		return out
	case "darwin":
		return []cleaner.Cleaner{quickLookThumbs{}}
	}
	return nil
}

// quickLookThumbs shells out to `qlmanage -r cache`. macOS-only.
type quickLookThumbs struct{}

func (quickLookThumbs) ID() string       { return "system/quicklook" }
func (quickLookThumbs) Name() string     { return "Quick Look — thumbnail cache" }
func (quickLookThumbs) Category() string { return "System" }

func (quickLookThumbs) Available(ctx context.Context) bool {
	return runtime.GOOS == "darwin" && cleaner.LookPath("qlmanage")
}

func (q quickLookThumbs) Scan(ctx context.Context) (cleaner.Result, error) {
	// Quick Look storage lives under /private/var/folders/.../C/com.apple.QuickLook.thumbnailcache
	// finding it requires shell-globbing through TMPDIR's parent. Cheaper: just
	// report the action via Path; size is unknown until clean runs.
	return cleaner.Result{Path: "qlmanage -r cache"}, nil
}

func (q quickLookThumbs) Clean(ctx context.Context, opts cleaner.Options) (cleaner.Result, error) {
	if !q.Available(ctx) {
		return cleaner.Result{}, nil
	}
	if opts.DryRun {
		return cleaner.Result{Path: "qlmanage -r cache"}, nil
	}
	if _, err := cleaner.RunCmd(ctx, "qlmanage", "-r", "cache"); err != nil {
		return cleaner.Result{}, err
	}
	return cleaner.Result{Path: "qlmanage -r cache"}, nil
}
