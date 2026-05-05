package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/ariefsn/dust/internal/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Inspect or scaffold the dust config file",
		Long: `Inspect or scaffold the dust config file.

The config file lives at:
  - macOS: ~/Library/Application Support/dust/config.yaml
  - Linux: $XDG_CONFIG_HOME/dust/config.yaml (default ~/.config/dust/config.yaml)

Override with --config <path>.`,
	}
	cmd.AddCommand(newConfigInitCmd())
	cmd.AddCommand(newConfigShowCmd())
	cmd.AddCommand(newConfigPathCmd())
	return cmd
}

func newConfigInitCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Write a default config file at the standard path",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, _ := cmd.Flags().GetString("config")
			if path == "" {
				path = config.DefaultPath()
			}
			if path == "" {
				return fmt.Errorf("could not resolve a config path on this OS")
			}
			if err := config.WriteDefault(path, force); err != nil {
				return err
			}
			cmd.Printf("Wrote default config to %s\n", path)
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "overwrite an existing config file")
	return cmd
}

func newConfigShowCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Print the resolved config (file + env + flags merged)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(loaded)
			}
			out, err := yaml.Marshal(loaded)
			if err != nil {
				return err
			}
			_, err = cmd.OutOrStdout().Write(out)
			return err
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print as JSON instead of YAML")
	return cmd
}

func newConfigPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print the path dust will load config from",
		Run: func(cmd *cobra.Command, args []string) {
			path, _ := cmd.Flags().GetString("config")
			if path == "" {
				path = config.DefaultPath()
			}
			fmt.Fprintln(cmd.OutOrStdout(), path)
			if _, err := os.Stat(path); os.IsNotExist(err) {
				fmt.Fprintln(cmd.OutOrStdout(), "(does not exist — run `dust config init` to create it)")
			}
		},
	}
}
