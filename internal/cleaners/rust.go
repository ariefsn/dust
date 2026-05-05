package cleaners

import "github.com/ariefsn/dust/internal/cleaner"

// CargoRegistry — wipe ~/.cargo/registry (cache + src).
// Cargo redownloads from crates.io on next build.
func CargoRegistry() cleaner.Cleaner {
	return pathBased{
		id:       "rust/registry",
		name:     "Cargo — registry cache",
		category: "Rust",
		resolvePath: func() string {
			return cleaner.Expand("~/.cargo/registry")
		},
	}
}

// CargoGitDB — wipe ~/.cargo/git/db (Git-sourced deps).
func CargoGitDB() cleaner.Cleaner {
	return pathBased{
		id:       "rust/git",
		name:     "Cargo — git db",
		category: "Rust",
		resolvePath: func() string {
			return cleaner.Expand("~/.cargo/git/db")
		},
	}
}
