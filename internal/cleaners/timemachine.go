package cleaners

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"github.com/ariefsn/dust/internal/cleaner"
)

// TimeMachineSnapshots cleans local Time Machine snapshots on the boot disk.
// macOS keeps local snapshots on the system volume — sometimes 50+ GB worth —
// even when an external Time Machine destination is configured.
//
// We can't easily report bytes per snapshot (APFS snapshots are sparse), so
// Scan reports the snapshot *count* via the Path column and 0 B for size.
// Clean shells out to `tmutil deletelocalsnapshots <date>` for each snapshot.
type timeMachineSnapshots struct{}

func TimeMachineSnapshots() cleaner.Cleaner { return timeMachineSnapshots{} }

func (timeMachineSnapshots) ID() string       { return "timemachine/local-snapshots" }
func (timeMachineSnapshots) Name() string     { return "Time Machine — local snapshots (confirm-twice)" }
func (timeMachineSnapshots) Category() string { return "Time Machine" }

func (timeMachineSnapshots) Available(ctx context.Context) bool {
	return runtime.GOOS == "darwin" && cleaner.LookPath("tmutil")
}

// listSnapshotDates returns each snapshot's date string, suitable for use as
// the argument to `tmutil deletelocalsnapshots`. Output looks like:
//
//	Snapshot dates for disk /:
//	2026-05-04-013815
//	2026-05-05-101412
func listSnapshotDates(ctx context.Context) ([]string, error) {
	out, err := cleaner.RunCmd(ctx, "tmutil", "listlocalsnapshotdates", "/")
	if err != nil {
		return nil, err
	}
	var dates []string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		line = strings.TrimSpace(line)
		// Skip the header and any unexpected non-date lines.
		if line == "" || strings.HasPrefix(line, "Snapshot dates") {
			continue
		}
		dates = append(dates, line)
	}
	return dates, nil
}

func (t timeMachineSnapshots) Scan(ctx context.Context) (cleaner.Result, error) {
	if !t.Available(ctx) {
		return cleaner.Result{}, nil
	}
	dates, err := listSnapshotDates(ctx)
	if err != nil {
		return cleaner.Result{}, err
	}
	path := fmt.Sprintf("%d snapshot(s) on /", len(dates))
	if len(dates) == 0 {
		path = "no local snapshots"
	}
	return cleaner.Result{Bytes: 0, Items: len(dates), Path: path}, nil
}

func (t timeMachineSnapshots) Clean(ctx context.Context, opts cleaner.Options) (cleaner.Result, error) {
	if !t.Available(ctx) {
		return cleaner.Result{}, nil
	}
	dates, err := listSnapshotDates(ctx)
	if err != nil {
		return cleaner.Result{}, err
	}
	if len(dates) == 0 {
		return cleaner.Result{Path: "no local snapshots"}, nil
	}
	if opts.DryRun {
		return cleaner.Result{Items: len(dates), Path: fmt.Sprintf("%d snapshot(s) on /", len(dates))}, nil
	}
	for _, d := range dates {
		if _, err := cleaner.RunCmd(ctx, "tmutil", "deletelocalsnapshots", d); err != nil {
			return cleaner.Result{}, fmt.Errorf("deletelocalsnapshots %s: %w", d, err)
		}
	}
	return cleaner.Result{Items: len(dates), Path: fmt.Sprintf("deleted %d snapshot(s)", len(dates))}, nil
}
