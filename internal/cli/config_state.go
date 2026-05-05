package cli

import "github.com/ariefsn/dust/internal/config"

// loaded holds the merged config. Populated in PersistentPreRunE on the root.
// Subcommands read this directly to honor settings the user put in their
// config file (project roots, stale_days, prefer_tool, etc.).
var loaded config.Config
