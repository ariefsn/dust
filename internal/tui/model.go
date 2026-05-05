package tui

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/ariefsn/dust/internal/cleaner"
	"github.com/ariefsn/dust/internal/cleaners"
	"github.com/ariefsn/dust/internal/cleaners/projects"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
)

// Options controls how the TUI is constructed.
type Options struct {
	Verbose         bool
	IncludeProjects bool
	ProjectsConfig  projects.Config
	PreferTool      bool
}

type focus int

const (
	focusCategories focus = iota
	focusItems
)

type screen int

const (
	screenList screen = iota
	screenConfirm
	screenRunning
	screenDone
)

type item struct {
	c        cleaner.Cleaner
	res      cleaner.Result
	scanned  bool
	scanErr  error
	selected bool
}

type category struct {
	name  string
	items []*item
}

// totalSize sums Bytes across selected items, deduping by Result.Path so the
// pnpm prune/wipe pair (same path, two action variants) isn't double-counted.
func (c category) totalBytes() int64 {
	var n int64
	seen := map[string]bool{}
	for _, it := range c.items {
		if seen[it.res.Path] && it.res.Path != "" {
			continue
		}
		if it.res.Path != "" {
			seen[it.res.Path] = true
		}
		n += it.res.Bytes
	}
	return n
}

type runResult struct {
	itemID string
	freed  int64
	err    error
}

type Model struct {
	keys       keyMap
	categories []*category
	catIdx     int
	itemIdx    int
	focus      focus
	screen     screen

	scanning  bool // true while initial cache scan in progress
	scanStart time.Time

	// Project scanner state — runs in parallel with the cache scan when enabled.
	includeProjects bool
	projectsConfig  projects.Config
	preferTool      bool
	projectsScanning bool
	projectsLoaded   bool

	width     int
	height    int
	dryRun    bool
	verbose   bool
	showEmpty bool // when true, render unavailable items / empty categories
	helpOpen  bool
	statusMsg string

	// Running-screen state.
	runStart    time.Time
	runTotal    int      // total cleaners we'll execute
	runDone     int      // how many have finished
	runCurrent  string   // name of the cleaner currently running
	runLog      []string // per-cleaner log lines, newest at the bottom
	runResults  []runResult
	runProgress progress.Model
}

func New() Model {
	return NewWithOpts(false)
}

func NewWithOpts(verbose bool) Model {
	return NewWithFullOpts(Options{Verbose: verbose})
}

func NewWithFullOpts(o Options) Model {
	bar := progress.New(
		progress.WithDefaultGradient(),
		progress.WithoutPercentage(),
	)
	return Model{
		keys:             defaultKeys(),
		focus:            focusCategories,
		screen:           screenList,
		scanning:         true,
		scanStart:        time.Now(),
		verbose:          o.Verbose,
		includeProjects:  o.IncludeProjects,
		projectsConfig:   o.ProjectsConfig,
		preferTool:       o.PreferTool,
		projectsScanning: o.IncludeProjects,
		runProgress:      bar,
	}
}

// tickMsg drives a 1Hz redraw so elapsed-time displays stay live.
type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// Init kicks off the initial scan in a goroutine via a tea.Cmd. When
// IncludeProjects is set, a parallel project scan runs and merges its results
// into the categories pane when complete.
func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{scanAllCmd(), tea.WindowSize(), tick()}
	if m.includeProjects {
		cmds = append(cmds, scanProjectsCmd(m.projectsConfig, m.preferTool))
	}
	return tea.Batch(cmds...)
}

// projectsLoadedMsg is sent when the project scan completes.
type projectsLoadedMsg struct {
	items     []*item
	totalSeen int      // raw count returned by Scan, before bytes>0 filter
	roots     []string // the roots Scan actually walked
	scanErr   error
}

func scanProjectsCmd(cfg projects.Config, preferTool bool) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		// projects.Scan auto-detects roots when cfg.Roots is empty; mirror
		// that here so we can report what was actually walked.
		walked := cfg.Roots
		if len(walked) == 0 {
			walked = projects.AutoRoots()
		}
		ps, err := projects.Scan(ctx, cfg)
		if err != nil {
			return projectsLoadedMsg{scanErr: err, roots: walked}
		}
		out := make([]*item, 0, len(ps))
		for _, p := range ps {
			if p.Bytes == 0 {
				continue
			}
			c := projects.AsCleaner{P: p, PreferTool: preferTool}
			out = append(out, &item{
				c:       c,
				res:     cleaner.Result{Bytes: p.Bytes, Items: p.Items, Path: p.Path},
				scanned: true,
			})
		}
		return projectsLoadedMsg{items: out, totalSeen: len(ps), roots: walked}
	}
}

// scanAllMsg is sent when the initial scan completes.
type scanAllMsg struct {
	categories []*category
}

func scanAllCmd() tea.Cmd {
	return func() tea.Msg {
		reg := cleaner.NewRegistry()
		cleaners.RegisterAll(reg)
		all := reg.All()

		// Pre-build category structure.
		byCat := map[string][]*item{}
		for _, c := range all {
			byCat[c.Category()] = append(byCat[c.Category()], &item{c: c})
		}

		// Run all scans in parallel.
		ctx := context.Background()
		var wg sync.WaitGroup
		for _, items := range byCat {
			for _, it := range items {
				it := it
				wg.Add(1)
				go func() {
					defer wg.Done()
					if !it.c.Available(ctx) {
						return
					}
					res, err := it.c.Scan(ctx)
					it.res = res
					it.scanErr = err
					it.scanned = true
				}()
			}
		}
		wg.Wait()

		catNames := make([]string, 0, len(byCat))
		for n := range byCat {
			catNames = append(catNames, n)
		}
		sort.Strings(catNames)

		cats := make([]*category, 0, len(catNames))
		for _, n := range catNames {
			its := byCat[n]
			sort.Slice(its, func(i, j int) bool {
				if its[i].res.Bytes != its[j].res.Bytes {
					return its[i].res.Bytes > its[j].res.Bytes
				}
				return its[i].c.ID() < its[j].c.ID()
			})
			cats = append(cats, &category{name: n, items: its})
		}
		return scanAllMsg{categories: cats}
	}
}

// runStartMsg announces the cleaner about to run (so the UI can show its name).
type runStartMsg struct {
	idx   int // 1-based
	total int
	name  string
}

// runOneDoneMsg reports a single cleaner's outcome.
type runOneDoneMsg struct {
	idx     int
	total   int
	name    string
	freed   int64
	err     error
	itemID  string
}

// runAllDoneMsg fires after every cleaner finishes.
type runAllDoneMsg struct{}

// runSelectedCmds builds a chain of commands: for each item, a "start" message,
// then the actual Clean (which produces a "done" message). After the last item,
// runAllDoneMsg flips the screen.
func runSelectedCmds(items []*item, dryRun bool) tea.Cmd {
	cmds := make([]tea.Cmd, 0, 2*len(items)+1)
	total := len(items)
	for i, it := range items {
		i, it := i, it
		idx := i + 1
		cmds = append(cmds, func() tea.Msg {
			return runStartMsg{idx: idx, total: total, name: it.c.Name()}
		})
		cmds = append(cmds, func() tea.Msg {
			ctx := context.Background()
			res, err := it.c.Clean(ctx, cleaner.Options{DryRun: dryRun, Yes: true})
			return runOneDoneMsg{
				idx:    idx,
				total:  total,
				name:   it.c.Name(),
				freed:  res.Bytes,
				err:    err,
				itemID: it.c.ID(),
			}
		})
	}
	cmds = append(cmds, func() tea.Msg { return runAllDoneMsg{} })
	return tea.Sequence(cmds...)
}
