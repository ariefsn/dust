package cleaners

import (
	"context"
	"runtime"

	"github.com/ariefsn/dust/internal/cleaner"
)

// xcodeSubitem builds a single Xcode-related cleaner. All gate on darwin + the
// presence of ~/Library/Developer/Xcode (data dir, not the app).
type xcodeSubitem struct {
	id, name string
	path     string
	confirm  bool // marker — surfaced via Name suffix; enforced in `dust clean`.
}

func (x xcodeSubitem) ID() string       { return x.id }
func (x xcodeSubitem) Name() string     { return x.name }
func (x xcodeSubitem) Category() string { return "Xcode" }

func (x xcodeSubitem) Available(ctx context.Context) bool {
	if runtime.GOOS != "darwin" {
		return false
	}
	return cleaner.IsDir(cleaner.Expand("~/Library/Developer/Xcode")) && cleaner.IsDir(x.path)
}

func (x xcodeSubitem) Scan(ctx context.Context) (cleaner.Result, error) {
	bytes, items, err := cleaner.DirSize(ctx, x.path)
	return cleaner.Result{Bytes: bytes, Items: items, Path: x.path}, err
}

func (x xcodeSubitem) Clean(ctx context.Context, opts cleaner.Options) (cleaner.Result, error) {
	before, _ := x.Scan(ctx)
	if opts.DryRun {
		return before, nil
	}
	if err := cleaner.SafeRemoveAll(before.Path, []string{cleaner.Home()}); err != nil {
		return cleaner.Result{}, err
	}
	return before, nil
}

// XcodeCleaners returns every Xcode sub-item cleaner.
func XcodeCleaners() []cleaner.Cleaner {
	if runtime.GOOS != "darwin" {
		return nil
	}
	expand := cleaner.Expand
	return []cleaner.Cleaner{
		xcodeSubitem{id: "xcode/deriveddata", name: "Xcode — DerivedData", path: expand("~/Library/Developer/Xcode/DerivedData")},
		xcodeSubitem{id: "xcode/archives", name: "Xcode — Archives (confirm-twice)", path: expand("~/Library/Developer/Xcode/Archives"), confirm: true},
		xcodeSubitem{id: "xcode/ios-devicesupport", name: "Xcode — iOS DeviceSupport", path: expand("~/Library/Developer/Xcode/iOS DeviceSupport")},
		xcodeSubitem{id: "xcode/watchos-devicesupport", name: "Xcode — watchOS DeviceSupport", path: expand("~/Library/Developer/Xcode/watchOS DeviceSupport")},
		xcodeSubitem{id: "xcode/tvos-devicesupport", name: "Xcode — tvOS DeviceSupport", path: expand("~/Library/Developer/Xcode/tvOS DeviceSupport")},
		xcodeSubitem{id: "xcode/ios-device-logs", name: "Xcode — iOS Device Logs", path: expand("~/Library/Developer/Xcode/iOS Device Logs")},
		xcodeSubitem{id: "xcode/products", name: "Xcode — Products", path: expand("~/Library/Developer/Xcode/Products")},
		xcodeSubitem{id: "xcode/ib-support", name: "Xcode — IB Support", path: expand("~/Library/Developer/Xcode/UserData/IB Support")},
		xcodeSubitem{id: "xcode/app-cache", name: "Xcode — app cache", path: expand("~/Library/Caches/com.apple.dt.Xcode")},
		xcodeSubitem{id: "xcode/coresimulator-caches", name: "Xcode — CoreSimulator caches", path: expand("~/Library/Developer/CoreSimulator/Caches")},
		simctlDeleteUnavailable{},
	}
}

// simctlDeleteUnavailable shells out to `xcrun simctl delete unavailable`.
type simctlDeleteUnavailable struct{}

func (simctlDeleteUnavailable) ID() string       { return "xcode/simctl-unavailable" }
func (simctlDeleteUnavailable) Name() string     { return "Xcode — delete unavailable simulators" }
func (simctlDeleteUnavailable) Category() string { return "Xcode" }

func (simctlDeleteUnavailable) Available(ctx context.Context) bool {
	return runtime.GOOS == "darwin" && cleaner.LookPath("xcrun")
}

func (simctlDeleteUnavailable) Scan(ctx context.Context) (cleaner.Result, error) {
	// `simctl list` doesn't report bytes; report 0 with a path label so the user
	// sees the row. The shell-out is the action.
	return cleaner.Result{Path: "xcrun simctl delete unavailable"}, nil
}

func (s simctlDeleteUnavailable) Clean(ctx context.Context, opts cleaner.Options) (cleaner.Result, error) {
	if !s.Available(ctx) {
		return cleaner.Result{}, nil
	}
	if opts.DryRun {
		return cleaner.Result{Path: "xcrun simctl delete unavailable"}, nil
	}
	if _, err := cleaner.RunCmd(ctx, "xcrun", "simctl", "delete", "unavailable"); err != nil {
		return cleaner.Result{}, err
	}
	return cleaner.Result{Path: "xcrun simctl delete unavailable"}, nil
}
