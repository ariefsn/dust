package projects

// Kind describes a project type — its manifest file, the optional `clean`
// tool command that handles cleanup correctly, and the artifact dirs to
// path-delete when the tool isn't available.
type Kind struct {
	Name string // human label, e.g. "Node"

	// Manifests is one or more filenames (relative to project root) that mark
	// a project of this kind. Any match is enough.
	Manifests []string

	// Tool is the binary name to invoke (e.g. "flutter") for `prefer_tool`
	// mode. Empty string means there's no tool — always path-delete.
	Tool string

	// ToolArgs are the args passed to Tool, e.g. ["clean"].
	ToolArgs []string

	// Artifacts are dir names (or globs, relative to project root) to remove
	// when path-delete fallback runs.
	Artifacts []string
}

// AllKinds is the v1 kind list. Order doesn't matter — multiple kinds can
// match a single project (e.g. Flutter + CocoaPods).
var AllKinds = []Kind{
	{
		Name:      "Node",
		Manifests: []string{"package.json"},
		// No universal `clean`; some projects ship one but we can't rely on it.
		Artifacts: []string{
			"node_modules",
			".next", ".nuxt", ".svelte-kit",
			"dist", "build",
			".parcel-cache", ".turbo", ".cache",
		},
	},
	{
		Name:      "Flutter",
		Manifests: []string{"pubspec.yaml"},
		Tool:      "flutter",
		ToolArgs:  []string{"clean"},
		Artifacts: []string{"build", ".dart_tool", "ephemeral", "ios/Pods"},
	},
	{
		Name:      "Rust",
		Manifests: []string{"Cargo.toml"},
		Tool:      "cargo",
		ToolArgs:  []string{"clean"},
		Artifacts: []string{"target"},
	},
	{
		Name:      "Maven",
		Manifests: []string{"pom.xml"},
		Tool:      "mvn",
		ToolArgs:  []string{"clean", "-q"},
		Artifacts: []string{"target"},
	},
	{
		Name:      "Gradle",
		Manifests: []string{"build.gradle", "build.gradle.kts"},
		// Prefer the wrapper if present; resolveGradle() picks at runtime.
		Tool:      "", // resolved via wrapper detection in action.go
		Artifacts: []string{"build", ".gradle"},
	},
	{
		Name:      ".NET",
		Manifests: []string{"*.csproj", "*.sln", "*.fsproj"},
		Tool:      "dotnet",
		ToolArgs:  []string{"clean"},
		Artifacts: []string{"bin", "obj"},
	},
	{
		Name:      "iOS (CocoaPods)",
		Manifests: []string{"Podfile"},
		Artifacts: []string{"Pods"},
	},
	{
		Name:      "Go",
		Manifests: []string{"go.mod"},
		// `go clean` only cleans the current package's build outputs — useful
		// but small. The big wins are vendor/ and any node_modules in tooling.
		Tool:      "go",
		ToolArgs:  []string{"clean"},
		Artifacts: []string{"vendor"},
	},
	{
		Name:      "Python",
		Manifests: []string{"pyproject.toml", "requirements.txt", "setup.py"},
		Artifacts: []string{
			".venv", "venv",
			"__pycache__", ".pytest_cache", ".mypy_cache", ".ruff_cache",
			"build", "dist", "*.egg-info",
		},
	},
	{
		Name:      "PHP",
		Manifests: []string{"composer.json"},
		Artifacts: []string{"vendor"},
	},
	{
		Name:      "Ruby",
		Manifests: []string{"Gemfile"},
		Tool:      "bundle",
		ToolArgs:  []string{"clean", "--force"},
		Artifacts: []string{".bundle", "vendor/bundle"},
	},
}
