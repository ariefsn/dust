package cli

import (
	"github.com/ariefsn/dust/internal/config"
	"github.com/ariefsn/dust/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

// Build-time variables. Override with `-ldflags` at build time:
//   go build -ldflags "-X github.com/ariefsn/dust/internal/cli.version=v0.1.0 \
//                       -X github.com/ariefsn/dust/internal/cli.commit=$(git rev-parse HEAD) \
//                       -X github.com/ariefsn/dust/internal/cli.date=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dust",
		Short: "Cross-platform dev-cache cleaner",
		Long: `dust scans dev-tool, browser, and project caches across macOS and Linux,
shows their sizes, and cleans the ones you pick.

Common workflows:
  dust scan                                  # see what's reclaimable
  dust list                                  # list every cleaner by category
  dust clean --all --dry-run                 # preview cleaning everything
  dust clean --category=docker,js --yes      # clean a category, no prompt
  dust clean --item=desktop-apps/discord/http-cache --yes

An interactive TUI launches when dust is run with no arguments. Pass --verbose
(or -v) to stream per-cleaner log lines during clean operations.`,
		Example: `  dust scan
  dust clean --all --dry-run
  dust clean --category=js --yes
  dust clean --item=docker,homebrew --yes`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			cfgPath, _ := cmd.Flags().GetString("config")
			cfg, err := config.Load(cfgPath)
			if err != nil {
				return err
			}
			// CLI flag wins over config file value.
			if v, _ := cmd.Flags().GetBool("verbose"); v {
				cfg.Verbose = true
			}
			loaded = cfg
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			runProjects, _ := cmd.Flags().GetBool("projects")
			projectRoots, _ := cmd.Flags().GetStringSlice("root")
			staleDays, _ := cmd.Flags().GetInt("stale-days")

			// Build projects config eagerly so the in-TUI `p` keypress can
			// kick off a scan with the right roots / preferTool even when
			// the user didn't pass --projects.
			projectsCfg := buildProjectsConfig(projectRoots, staleDays)
			preferTool := preferToolWithConfig(cmd)

			opts := tui.Options{
				Verbose:         loaded.Verbose,
				IncludeProjects: runProjects,
				ProjectsConfig:  projectsCfg,
				PreferTool:      preferTool,
			}

			p := tea.NewProgram(tui.NewWithFullOpts(opts), tea.WithAltScreen())
			_, err := p.Run()
			return err
		},
		SilenceUsage: true,
	}
	cmd.PersistentFlags().BoolP("verbose", "v", false, "verbose mode (show per-cleaner log lines during clean)")
	cmd.PersistentFlags().String("config", "", "path to config file (default: <user config dir>/dust/config.yaml)")
	cmd.Flags().BoolP("projects", "P", false, "also scan project directories (~/Projects, ~/Work, etc.) for stale build outputs")
	cmd.Flags().StringSlice("root", nil, "project roots to scan (default: auto-detect)")
	cmd.Flags().Int("stale-days", 0, "only show projects untouched for N days (0 = no filter)")

	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(newScanCmd())
	cmd.AddCommand(newCleanCmd())
	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newConfigCmd())
	cmd.AddCommand(newUpgradeCmd())
	return cmd
}

func newVersionCmd() *cobra.Command {
	var short bool
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the dust version",
		Run: func(cmd *cobra.Command, args []string) {
			if short {
				cmd.Println(version)
				return
			}
			cmd.Printf("dust %s\n", version)
			cmd.Printf("  commit: %s\n", commit)
			cmd.Printf("  built:  %s\n", date)
		},
	}
	cmd.Flags().BoolVar(&short, "short", false, "print only the version string")
	return cmd
}

func Execute() error {
	return newRootCmd().Execute()
}
