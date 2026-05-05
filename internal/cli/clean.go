package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"text/tabwriter"

	"github.com/ariefsn/dust/internal/cleaner"
	"github.com/ariefsn/dust/internal/cleaners"
	"github.com/ariefsn/dust/internal/cleaners/projects"
	"github.com/spf13/cobra"
)

func newCleanCmd() *cobra.Command {
	var (
		all          bool
		categories   []string
		items        []string
		dryRun       bool
		yes          bool
		runProjects  bool
		projectRoots []string
		staleDays    int
		preferTool   bool
		includeDirty bool
	)
	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Clean selected dev-tool caches",
		Long: `Clean dev-tool, browser, and project caches.

By default, dust clean refuses to do anything without an explicit selection —
either --all, --category, --item, or --projects. Use --dry-run to preview
without deleting.

With --projects, dust walks your project directories and offers to delete
build artifacts (node_modules/, target/, .venv/, etc.) per project. Projects
with uncommitted git changes are skipped unless you pass --include-dirty.`,
		Example: `  dust clean --all --dry-run                 # preview every cleaner
  dust clean --category=js,docker --yes      # clean by category
  dust clean --item=docker,go/modcache       # clean specific cleaners
  dust clean --projects --stale-days=90 --dry-run`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !all && !runProjects && len(categories) == 0 && len(items) == 0 {
				return fmt.Errorf("nothing selected: pass --all, --category, --item, or --projects")
			}
			opts := cleaner.Options{DryRun: dryRun, Yes: yes}

			if all || len(categories) > 0 || len(items) > 0 {
				selected, err := selectCleaners(all, categories, items)
				if err != nil {
					return err
				}
				if len(selected) == 0 {
					cmd.Println("No matching cleaners.")
				} else {
					if err := runClean(cmd.Context(), cmd, selected, opts); err != nil {
						return err
					}
				}
			}

			if runProjects {
				cfg := buildProjectsConfig(projectRoots, staleDays)
				cfg.SkipDirty = !includeDirty
				// preferTool flag wins; otherwise inherit from config file.
				if !cmd.Flags().Changed("prefer-tool") {
					preferTool = loaded.ProjectScanner.PreferTool
				}
				if err := runProjectClean(cmd, cfg, preferTool, opts); err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "select every available cleaner")
	cmd.Flags().StringSliceVar(&categories, "category", nil, "select cleaners by category (comma-separated, repeatable)")
	cmd.Flags().StringSliceVar(&items, "item", nil, "select cleaners by ID (comma-separated, repeatable)")
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "show what would be deleted without touching anything")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip the confirmation prompt")
	cmd.Flags().BoolVar(&runProjects, "projects", false, "also clean stale project build outputs (node_modules, target, .venv, ...)")
	cmd.Flags().StringSliceVar(&projectRoots, "root", nil, "project roots (default: auto-detect ~/Projects, ~/Code, etc.)")
	cmd.Flags().IntVar(&staleDays, "stale-days", 0, "only clean projects untouched for N days (0 = no filter)")
	cmd.Flags().BoolVar(&preferTool, "prefer-tool", true, "prefer per-project clean tools (flutter clean, cargo clean) over rm -r")
	cmd.Flags().BoolVar(&includeDirty, "include-dirty", false, "include projects with uncommitted git changes (default: skip)")
	return cmd
}

// runProjectClean walks the configured roots, prints the discovered projects,
// confirms, then runs the clean action per project.
//
// Projects with 0 B of reclaimable artifacts are filtered out by default —
// they're noise for the cleaning workflow.
func runProjectClean(cmd *cobra.Command, cfg projects.Config, preferTool bool, opts cleaner.Options) error {
	stop := startSpinner(cmd.OutOrStdout(), "Scanning projects...")
	allProjs, err := projects.Scan(cmd.Context(), cfg)
	stop()
	if err != nil {
		return err
	}
	// Drop empties — there's nothing to clean and they pad the confirmation.
	projs := make([]projects.Project, 0, len(allProjs))
	for _, p := range allProjs {
		if p.Bytes > 0 {
			projs = append(projs, p)
		}
	}
	if len(projs) == 0 {
		cmd.Println("No projects with reclaimable artifacts.")
		return nil
	}

	verb := "Clean"
	if opts.DryRun {
		verb = "[dry-run] Would clean"
	}
	var total int64
	for _, p := range projs {
		total += p.Bytes
	}
	cmd.Printf("%s %d project(s), ~%s reclaimable.\n", verb, len(projs), cleaner.HumanBytes(total))

	tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "  PATH\tKINDS\tSIZE\tLAST TOUCHED\tDIRTY")
	for _, p := range projs {
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
	cmd.Println()

	if !opts.Yes && !opts.DryRun {
		if !confirm(cmd, "Proceed? [y/N] ") {
			cmd.Println("Aborted.")
			return nil
		}
	}

	var freed int64
	var failures int

	var setProgress func(int, string)
	var stopProgress func()
	if !opts.DryRun {
		setProgress, stopProgress = startProgressSpinner(cmd.OutOrStdout(), len(projs), "Cleaning projects")
	}

	for i, p := range projs {
		if setProgress != nil {
			setProgress(i, displayProjectPath(p.Path))
		}
		bytes, err := projects.CleanProject(cmd.Context(), p, preferTool, opts.DryRun)
		switch {
		case err != nil:
			failures++
			cmd.Printf("  ✗ %s: %v\n", displayProjectPath(p.Path), err)
		case opts.DryRun:
			cmd.Printf("  ~ %s: would free %s\n", displayProjectPath(p.Path), cleaner.HumanBytes(bytes))
		default:
			freed += bytes
			cmd.Printf("  ✓ %s: freed %s\n", displayProjectPath(p.Path), cleaner.HumanBytes(bytes))
		}
		if setProgress != nil {
			setProgress(i+1, "")
		}
	}
	if stopProgress != nil {
		stopProgress()
	}

	cmd.Println()
	if opts.DryRun {
		cmd.Printf("Project dry run complete. Would have freed ~%s.\n", cleaner.HumanBytes(total))
	} else {
		cmd.Printf("Project clean done. Freed %s across %d project(s).\n", cleaner.HumanBytes(freed), len(projs)-failures)
		if failures > 0 {
			cmd.Printf("%d project(s) failed — see errors above.\n", failures)
		}
	}
	return nil
}

// selectCleaners builds the list of cleaners to run from the user's flags.
// Cleaners that aren't Available() are silently dropped.
func selectCleaners(all bool, categories, items []string) ([]cleaner.Cleaner, error) {
	reg := cleaner.NewRegistry()
	cleaners.RegisterAll(reg)
	universe := reg.All()

	if all {
		return universe, nil
	}

	catSet := lowerSet(categories)
	itemSet := lowerSet(items)
	unmatchedItems := make(map[string]bool, len(itemSet))
	for k := range itemSet {
		unmatchedItems[k] = true
	}

	var out []cleaner.Cleaner
	for _, c := range universe {
		if catSet[strings.ToLower(c.Category())] {
			out = append(out, c)
			continue
		}
		if itemSet[strings.ToLower(c.ID())] {
			out = append(out, c)
			delete(unmatchedItems, strings.ToLower(c.ID()))
		}
	}

	if len(unmatchedItems) > 0 {
		var unknown []string
		for k := range unmatchedItems {
			unknown = append(unknown, k)
		}
		return nil, fmt.Errorf("unknown --item value(s): %s", strings.Join(unknown, ", "))
	}
	return out, nil
}

func lowerSet(in []string) map[string]bool {
	out := make(map[string]bool, len(in))
	for _, s := range in {
		out[strings.ToLower(strings.TrimSpace(s))] = true
	}
	return out
}

// runClean previews the planned actions, asks for confirmation (unless --yes),
// then runs each cleaner sequentially and prints a per-cleaner result.
func runClean(ctx context.Context, cmd *cobra.Command, all []cleaner.Cleaner, opts cleaner.Options) error {
	// Filter out unavailable cleaners (silent skip — `dust scan` shows them).
	var avail []cleaner.Cleaner
	for _, c := range all {
		if c.Available(ctx) {
			avail = append(avail, c)
		}
	}
	if len(avail) == 0 {
		cmd.Println("No available cleaners matched the selection.")
		return nil
	}

	// Pre-scan in parallel to estimate total. Dedupe by Result.Path so the
	// pnpm-prune / pnpm-wipe pair isn't counted twice.
	type preScanRow struct {
		c   cleaner.Cleaner
		res cleaner.Result
	}
	rows := make([]preScanRow, len(avail))
	stopScan := startSpinner(cmd.OutOrStdout(), "Scanning selected cleaners...")
	var wg sync.WaitGroup
	for i, c := range avail {
		wg.Add(1)
		go func(i int, c cleaner.Cleaner) {
			defer wg.Done()
			res, _ := c.Scan(ctx)
			rows[i] = preScanRow{c: c, res: res}
		}(i, c)
	}
	wg.Wait()
	stopScan()

	seenPaths := make(map[string]bool)
	var planned int64
	for _, r := range rows {
		if r.res.Path != "" && seenPaths[r.res.Path] {
			continue
		}
		if r.res.Path != "" {
			seenPaths[r.res.Path] = true
		}
		planned += r.res.Bytes
	}

	verb := "Clean"
	if opts.DryRun {
		verb = "[dry-run] Would clean"
	}
	cmd.Printf("%s %d cleaner(s), ~%s reclaimable.\n", verb, len(avail), cleaner.HumanBytes(planned))

	tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "  CLEANER\tSIZE\tPATH")
	for _, r := range rows {
		fmt.Fprintf(tw, "  %s\t%s\t%s\n", r.c.Name(), cleaner.HumanBytes(r.res.Bytes), r.res.Path)
	}
	tw.Flush()
	cmd.Println()

	if !opts.Yes && !opts.DryRun {
		if !confirm(cmd, fmt.Sprintf("Proceed? [y/N] ")) {
			cmd.Println("Aborted.")
			return nil
		}
	}

	// Execute. Sequential — many cleaners shell out and we don't want to
	// flood the daemon with parallel `docker` / `brew` calls.
	var totalFreed int64
	var failures int

	var setProgress func(int, string)
	var stopProgress func()
	if !opts.DryRun {
		setProgress, stopProgress = startProgressSpinner(cmd.OutOrStdout(), len(avail), "Cleaning")
	}

	for i, c := range avail {
		if setProgress != nil {
			setProgress(i, c.Name())
		}
		res, err := c.Clean(ctx, opts)
		switch {
		case err != nil:
			failures++
			cmd.Printf("  ✗ %s: %v\n", c.Name(), err)
		case opts.DryRun:
			cmd.Printf("  ~ %s: would free %s\n", c.Name(), cleaner.HumanBytes(res.Bytes))
		default:
			totalFreed += res.Bytes
			cmd.Printf("  ✓ %s: freed %s\n", c.Name(), cleaner.HumanBytes(res.Bytes))
		}
		if setProgress != nil {
			setProgress(i+1, "")
		}
	}
	if stopProgress != nil {
		stopProgress()
	}

	cmd.Println()
	if opts.DryRun {
		cmd.Printf("Dry run complete. Would have freed ~%s.\n", cleaner.HumanBytes(planned))
	} else {
		cmd.Printf("Done. Freed %s across %d cleaner(s).\n", cleaner.HumanBytes(totalFreed), len(avail)-failures)
		if failures > 0 {
			cmd.Printf("%d cleaner(s) failed — see errors above.\n", failures)
		}
	}
	return nil
}

func confirm(cmd *cobra.Command, prompt string) bool {
	fmt.Fprint(cmd.OutOrStdout(), prompt)
	r := bufio.NewReader(os.Stdin)
	line, err := r.ReadString('\n')
	if err != nil {
		return false
	}
	line = strings.TrimSpace(strings.ToLower(line))
	return line == "y" || line == "yes"
}
