package cleaners

import (
	"github.com/ariefsn/dust/internal/cleaner"
	"github.com/ariefsn/dust/internal/cleaners/browsers"
	desktop_apps "github.com/ariefsn/dust/internal/cleaners/desktop_apps"
	"github.com/ariefsn/dust/internal/cleaners/e2e"
	"github.com/ariefsn/dust/internal/cleaners/editors"
)

// RegisterAll adds every available cleaner to the registry.
// Cleaners self-filter via Available(), so registration is unconditional.
func RegisterAll(r *cleaner.Registry) {
	r.Register(Docker())

	// JS
	r.Register(Yarn())
	r.Register(NPM())
	r.Register(PnpmPrune())
	r.Register(PnpmStoreWipe())
	r.Register(Bun())
	r.Register(Deno())

	// JVM
	r.Register(Gradle())
	r.Register(Maven())

	// Python
	r.Register(Pip())
	r.Register(Conda())

	// Go
	r.Register(GoModCache())
	r.Register(GoBuildCache())

	// Rust
	r.Register(CargoRegistry())
	r.Register(CargoGitDB())

	// PHP
	r.Register(Composer())

	// .NET
	r.Register(NuGet())

	// Flutter / Dart
	r.Register(FlutterPubCache())

	// Xcode (darwin)
	for _, c := range XcodeCleaners() {
		r.Register(c)
	}

	// Android
	for _, c := range AndroidStaticCleaners() {
		r.Register(c)
	}

	// E2E test runners
	for _, c := range e2e.Cypress() {
		r.Register(c)
	}
	for _, c := range e2e.Playwright() {
		r.Register(c)
	}
	for _, c := range e2e.Puppeteer() {
		r.Register(c)
	}

	// Homebrew, System, Trash, iOS backups
	r.Register(Homebrew())
	for _, c := range SystemCleaners() {
		r.Register(c)
	}
	r.Register(Trash())
	r.Register(TimeMachineSnapshots())

	// iOS dev caches + backups
	r.Register(CocoaPodsCache())
	r.Register(CocoaPodsRepos())
	r.Register(Carthage())
	r.Register(IOSBackups())

	// Browsers
	for _, c := range browsers.All() {
		r.Register(c)
	}

	// Editors
	for _, c := range editors.JetBrainsCleaners() {
		r.Register(c)
	}
	for _, c := range editors.ElectronCleaners() {
		r.Register(c)
	}

	// Desktop apps
	for _, c := range desktop_apps.All() {
		r.Register(c)
	}
}
