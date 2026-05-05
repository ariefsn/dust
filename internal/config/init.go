package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// DefaultYAML is the template written by `dust config init`. Reflects the
// defaults in setDefaults() but with explanatory comments.
const DefaultYAML = `# dust config — see ` + "`dust --help`" + ` for the full feature surface.

# Global flag — equivalent to ` + "`dust --verbose`" + ` on every invocation.
verbose: false

project_scanner:
  enabled: true

  # Project roots. If 'roots' is set it REPLACES auto-detection. The usual
  # case is to add a root or two via 'extra_roots' — those are merged with
  # the auto-detected list (~/Projects, ~/Work, ~/Code, ~/dev, ~/src,
  # ~/repos, ~/Documents/Projects).
  # roots:
  #   - ~/MyStuff
  extra_roots: []

  # Only show projects untouched for N days. 0 = no filter.
  stale_days: 0

  # Maximum directory depth to descend during the walk.
  max_depth: 8

  # Prefer per-project tools (` + "`flutter clean`" + `, ` + "`cargo clean`" + `, etc.) over
  # plain ` + "`rm -r`" + ` of artifact dirs. Falls back to rm -r if the tool isn't
  # on $PATH or fails.
  prefer_tool: true
`

// WriteDefault writes the default YAML to `path`, creating parent dirs as
// needed. Refuses to overwrite an existing file unless `force` is true.
func WriteDefault(path string, force bool) error {
	if path == "" {
		return fmt.Errorf("no config path resolvable on this OS")
	}
	if !force {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("%s already exists; pass --force to overwrite", path)
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(DefaultYAML), 0o644)
}
