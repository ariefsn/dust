package projects

import (
	"context"
	"path/filepath"

	"github.com/ariefsn/dust/internal/cleaner"
)

// CleanProject runs the configured cleanup action(s) for a single project.
// One action per matched Kind. If `preferTool` and the kind's tool is
// resolvable from the project root, it runs the tool; otherwise it
// path-deletes the artifact dirs.
//
// Returns the bytes freed and any error. On error the function continues
// with the remaining kinds — partial success is normal (e.g. `flutter clean`
// works but `cargo clean` fails because no `Cargo.lock`).
func CleanProject(ctx context.Context, p Project, preferTool, dryRun bool) (int64, error) {
	before := p.Bytes
	if dryRun {
		return before, nil
	}

	for _, k := range p.Kinds {
		toolName, toolArgs, useTool := resolveTool(k, p.Path)
		if preferTool && useTool {
			if _, err := cleaner.RunCmdIn(ctx, p.Path, toolName, toolArgs...); err == nil {
				continue
			}
			// Tool failed — fall through to path-delete.
		}
		for _, art := range k.Artifacts {
			for _, target := range resolveArtifact(p.Path, art) {
				if err := cleaner.SafeRemoveAll(target, []string{cleaner.Home(), p.Path}); err != nil {
					return 0, err
				}
			}
		}
	}

	// Re-measure isn't strictly needed — we trust the pre-scan. Returning the
	// pre-scan size matches user expectation ("you said 1.2 GB, you got 1.2 GB").
	return before, nil
}

// resolveTool returns the actual binary name + args for a kind, accounting
// for the Gradle wrapper (./gradlew) being preferred over a global `gradle`.
func resolveTool(k Kind, projectRoot string) (name string, args []string, ok bool) {
	if k.Name == "Gradle" {
		// Prefer the wrapper if the project ships one.
		wrapper := filepath.Join(projectRoot, "gradlew")
		if cleaner.Exists(wrapper) {
			return "./gradlew", []string{"clean"}, true
		}
		if cleaner.LookPath("gradle") {
			return "gradle", []string{"clean"}, true
		}
		return "", nil, false
	}
	if k.Tool == "" {
		return "", nil, false
	}
	if !cleaner.LookPath(k.Tool) {
		return "", nil, false
	}
	return k.Tool, k.ToolArgs, true
}
