package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/ariefsn/dust/internal/cleaner"
	"github.com/ariefsn/dust/internal/cleaners"
	"github.com/ariefsn/dust/internal/cleaners/projects"
	"github.com/spf13/cobra"
)

func newScanCmd() *cobra.Command {
	var (
		asJSON       bool
		runProjects  bool
		projectRoots []string
		staleDays    int
		showEmpty    bool
	)
	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Scan installed dev-tool caches and report sizes",
		Long: `Scan installed dev-tool caches and report their sizes.

Pass --projects to also walk your project directories and surface stale
node_modules/, target/, .venv/, etc. By default, dust auto-detects common
roots like ~/Projects, ~/Code, ~/Work; override with --root.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			var stop func()
			if !asJSON {
				stop = startSpinner(out, "Scanning caches...")
			}
			results := scanAll(cmd.Context())
			if stop != nil {
				stop()
			}

			var projs []projects.Project
			if runProjects {
				cfg := buildProjectsConfig(projectRoots, staleDays)
				if !asJSON {
					stopP := startSpinner(out, "Scanning projects...")
					var err error
					projs, err = projects.Scan(cmd.Context(), cfg)
					stopP()
					if err != nil {
						return err
					}
				} else {
					var err error
					projs, err = projects.Scan(cmd.Context(), cfg)
					if err != nil {
						return err
					}
				}
			}

			if asJSON {
				return printScanJSON(cmd, results, projs)
			}
			printScanTable(cmd, results, showEmpty)
			if runProjects {
				printProjectsTable(cmd, projs, showEmpty)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable output")
	cmd.Flags().BoolVar(&runProjects, "projects", false, "also scan project directories for stale build outputs")
	cmd.Flags().StringSliceVar(&projectRoots, "root", nil, "project roots to scan (default: auto-detect ~/Projects, ~/Code, etc.)")
	cmd.Flags().IntVar(&staleDays, "stale-days", 0, "only show projects not touched in N days (0 = no filter)")
	cmd.Flags().BoolVar(&showEmpty, "show-empty", false, "include rows with 0 B reclaimable or 'not installed' (default: hide them)")
	return cmd
}

// preferToolWithConfig returns the prefer-tool setting honoring CLI > config.
// The "prefer-tool" flag is only declared on `dust clean`, so for callers
// (root TUI) that don't have it, we fall back to the config value.
func preferToolWithConfig(cmd *cobra.Command) bool {
	if cmd.Flags().Lookup("prefer-tool") != nil && cmd.Flags().Changed("prefer-tool") {
		v, _ := cmd.Flags().GetBool("prefer-tool")
		return v
	}
	return loaded.ProjectScanner.PreferTool
}

// buildProjectsConfig merges CLI flags with config-file values.
// Precedence: CLI flag > config file > default.
//
// Roots resolution rules (per the plan):
//   - --root flag (or config: roots): if set, REPLACES auto-detect
//   - config: extra_roots: ADDED to auto-detect (the common case)
func buildProjectsConfig(cliRoots []string, cliStaleDays int) projects.Config {
	ps := loaded.ProjectScanner

	// Roots: CLI > config.roots > (auto-detect + config.extra_roots).
	var roots []string
	switch {
	case len(cliRoots) > 0:
		roots = expandRoots(cliRoots)
	case len(ps.Roots) > 0:
		roots = expandRoots(ps.Roots)
	default:
		roots = projects.AutoRoots()
		for _, r := range expandRoots(ps.ExtraRoots) {
			roots = append(roots, r)
		}
	}

	stale := cliStaleDays
	if stale == 0 {
		stale = ps.StaleDays
	}
	maxDepth := ps.MaxDepth
	if maxDepth <= 0 {
		maxDepth = 8
	}
	return projects.Config{
		Roots:     roots,
		StaleDays: stale,
		MaxDepth:  maxDepth,
	}
}

func expandRoots(roots []string) []string {
	if len(roots) == 0 {
		return nil
	}
	out := make([]string, 0, len(roots))
	for _, r := range roots {
		out = append(out, cleaner.Expand(strings.TrimSpace(r)))
	}
	return out
}

func printProjectsTable(cmd *cobra.Command, projs []projects.Project, showEmpty bool) {
	if len(projs) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "\nNo projects found.")
		return
	}
	// Compute totals across the FULL set first so the summary stays accurate
	// even when we hide empty rows.
	var total int64
	var hidden int
	for _, p := range projs {
		total += p.Bytes
		if !showEmpty && p.Bytes == 0 {
			hidden++
		}
	}
	fmt.Fprintln(cmd.OutOrStdout(), "\nProjects:")
	tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "  PATH\tKINDS\tSIZE\tLAST TOUCHED\tDIRTY")
	for _, p := range projs {
		if !showEmpty && p.Bytes == 0 {
			continue
		}
		dirty := ""
		if p.Dirty {
			dirty = "yes"
		}
		fmt.Fprintf(tw, "  %s\t%s\t%s\t%s\t%s\n",
			displayProjectPath(p.Path),
			projectKindLabels(p),
			cleaner.HumanBytes(p.Bytes),
			formatLastTouched(p.LastTouched),
			dirty,
		)
	}
	tw.Flush()
	suffix := ""
	if hidden > 0 {
		suffix = fmt.Sprintf(" (%d empty hidden — pass --show-empty to include)", hidden)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "\n%d project(s), total reclaimable: %s%s\n",
		len(projs), cleaner.HumanBytes(total), suffix)
}

func projectKindLabels(p projects.Project) string {
	if len(p.Kinds) == 0 {
		return "?"
	}
	out := make([]string, 0, len(p.Kinds))
	for _, k := range p.Kinds {
		out = append(out, k.Name)
	}
	return strings.Join(out, "+")
}

func displayProjectPath(p string) string {
	home := cleaner.Home()
	if home != "" && strings.HasPrefix(p, home) {
		return "~" + p[len(home):]
	}
	return p
}

func formatLastTouched(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	return t.Format("2006-01-02")
}

type scanRow struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"`
	Bytes    int64  `json:"bytes"`
	Items    int    `json:"items"`
	Path     string `json:"path"`
	Err      string `json:"error,omitempty"`
}

func scanAll(ctx context.Context) []scanRow {
	reg := cleaner.NewRegistry()
	cleaners.RegisterAll(reg)
	all := reg.All()

	rows := make([]scanRow, len(all))
	var wg sync.WaitGroup
	for i, c := range all {
		wg.Add(1)
		go func(i int, c cleaner.Cleaner) {
			defer wg.Done()
			row := scanRow{ID: c.ID(), Name: c.Name(), Category: c.Category()}
			if !c.Available(ctx) {
				row.Err = "not installed"
				rows[i] = row
				return
			}
			res, err := c.Scan(ctx)
			if err != nil {
				row.Err = err.Error()
			}
			row.Bytes = res.Bytes
			row.Items = res.Items
			row.Path = res.Path
			rows[i] = row
		}(i, c)
	}
	wg.Wait()
	return rows
}

func printScanTable(cmd *cobra.Command, rows []scanRow, showEmpty bool) {
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Category != rows[j].Category {
			return rows[i].Category < rows[j].Category
		}
		return rows[i].Bytes > rows[j].Bytes
	})

	// Compute total + hidden count across the FULL set so the summary stays
	// honest when --show-empty is off.
	var total int64
	var hidden int
	for _, r := range rows {
		total += r.Bytes
		if !showEmpty && r.Bytes == 0 {
			hidden++
		}
	}

	tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "CATEGORY\tCLEANER\tSIZE\tPATH")
	prevCat := ""
	for _, r := range rows {
		// Hide both true 0 B rows AND "not installed" rows (Err set, Bytes 0).
		if !showEmpty && r.Bytes == 0 {
			continue
		}
		cat := r.Category
		if cat == prevCat {
			cat = ""
		} else {
			prevCat = r.Category
		}
		size := cleaner.HumanBytes(r.Bytes)
		if r.Err != "" {
			size = "—"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", cat, r.Name, size, displayPath(r))
	}
	tw.Flush()
	fmt.Fprintln(cmd.OutOrStdout())

	suffix := ""
	if hidden > 0 {
		suffix = fmt.Sprintf(" (%d empty hidden — pass --show-empty to include)", hidden)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Total reclaimable: %s%s\n", cleaner.HumanBytes(total), suffix)
}

func displayPath(r scanRow) string {
	if r.Err != "" {
		return r.Err
	}
	return r.Path
}

type scanJSON struct {
	Cleaners []scanRow         `json:"cleaners"`
	Projects []projectJSONRow  `json:"projects,omitempty"`
}

type projectJSONRow struct {
	Path        string    `json:"path"`
	Kinds       []string  `json:"kinds"`
	Bytes       int64     `json:"bytes"`
	Items       int       `json:"items"`
	LastTouched time.Time `json:"last_touched"`
	Dirty       bool      `json:"dirty"`
	HasGit      bool      `json:"has_git"`
}

func printScanJSON(cmd *cobra.Command, rows []scanRow, projs []projects.Project) error {
	pjs := make([]projectJSONRow, 0, len(projs))
	for _, p := range projs {
		kinds := make([]string, len(p.Kinds))
		for i, k := range p.Kinds {
			kinds[i] = k.Name
		}
		pjs = append(pjs, projectJSONRow{
			Path:        p.Path,
			Kinds:       kinds,
			Bytes:       p.Bytes,
			Items:       p.Items,
			LastTouched: p.LastTouched,
			Dirty:       p.Dirty,
			HasGit:      p.HasGit,
		})
	}
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(scanJSON{Cleaners: rows, Projects: pjs})
}
