package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Config is the root config struct unmarshaled from YAML / env / flags.
type Config struct {
	Verbose        bool                 `mapstructure:"verbose" yaml:"verbose" json:"verbose"`
	ProjectScanner ProjectScannerConfig `mapstructure:"project_scanner" yaml:"project_scanner" json:"project_scanner"`
}

// ProjectScannerConfig tunes `dust scan --projects` and `dust clean --projects`.
type ProjectScannerConfig struct {
	Enabled    bool     `mapstructure:"enabled" yaml:"enabled" json:"enabled"`
	Roots      []string `mapstructure:"roots" yaml:"roots,omitempty" json:"roots,omitempty"`
	ExtraRoots []string `mapstructure:"extra_roots" yaml:"extra_roots,omitempty" json:"extra_roots,omitempty"`
	StaleDays  int      `mapstructure:"stale_days" yaml:"stale_days" json:"stale_days"`
	MaxDepth   int      `mapstructure:"max_depth" yaml:"max_depth" json:"max_depth"`
	PreferTool bool     `mapstructure:"prefer_tool" yaml:"prefer_tool" json:"prefer_tool"`
}

// DefaultPath returns the config file path. We deliberately use the XDG-style
// `~/.config/dust/config.yaml` on every Unix-like OS (macOS + Linux) instead of
// macOS's `~/Library/Application Support`, so dotfiles repos can share a single
// path. Honors $XDG_CONFIG_HOME if set.
//
// Returns "" if the home dir can't be resolved (rare).
func DefaultPath() string {
	if v := os.Getenv("XDG_CONFIG_HOME"); v != "" {
		return filepath.Join(v, "dust", "config.yaml")
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".config", "dust", "config.yaml")
}

// Load reads the config file (if `explicitPath` is non-empty, that path
// exclusively; otherwise the default). Env vars prefixed `DUST_` override
// values from the file.
//
// Returns ((zero Config), nil) when no config file exists — that's not an
// error, callers should fall back to defaults.
func Load(explicitPath string) (Config, error) {
	v := viper.New()
	v.SetEnvPrefix("DUST")
	v.AutomaticEnv()
	// Map dotted config keys (project_scanner.stale_days) to env-var form
	// (DUST_PROJECT_SCANNER_STALE_DAYS).
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	setDefaults(v)

	if explicitPath != "" {
		v.SetConfigFile(explicitPath)
	} else {
		path := DefaultPath()
		if path == "" {
			// Fall through to defaults.
			var cfg Config
			if err := v.Unmarshal(&cfg); err != nil {
				return Config{}, err
			}
			return cfg, nil
		}
		v.SetConfigFile(path)
	}

	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) && !os.IsNotExist(unwrap(err)) {
			return Config{}, fmt.Errorf("read config: %w", err)
		}
		// File missing: defaults + env still apply.
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	return cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("verbose", false)
	v.SetDefault("project_scanner.enabled", true)
	v.SetDefault("project_scanner.stale_days", 0)
	v.SetDefault("project_scanner.max_depth", 8)
	v.SetDefault("project_scanner.prefer_tool", true)
}

// unwrap returns the wrapped error or the original.
func unwrap(err error) error {
	type unwrapper interface{ Unwrap() error }
	if u, ok := err.(unwrapper); ok && u.Unwrap() != nil {
		return u.Unwrap()
	}
	return err
}
