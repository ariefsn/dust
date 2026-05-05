# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-05-05

Initial public release.

### Highlights

- **99 cleaners across 19 categories** covering Docker, JS package managers,
  JVM build tools, Python, Go, Rust, Xcode, Android, E2E test runners,
  Homebrew, browsers, JetBrains and Electron editors, desktop apps, system
  caches, Trash, iOS device backups, Time Machine local snapshots, and
  global PHP / .NET / Flutter / Deno / Conda dev caches.
- **Project scanner** — walks configured roots, detects 11 project kinds via
  manifest files, surfaces stale build artifacts, prefers per-project clean
  tools (`flutter clean`, `cargo clean`, `mvn clean`, `dotnet clean`,
  `bundle clean`) over blunt `rm -r`, skips dirty git working trees by
  default.
- **Two-pane bubbletea TUI** with live elapsed-time spinner, gradient
  progress bar during clean, optional verbose log, and a hard `y/N`
  confirmation gate.
- **Single static binary** for darwin and linux on amd64 and arm64. CGO
  disabled. Drops anywhere on `$PATH`.
- **Self-update** (`dust upgrade`) that checks GitHub Releases, downloads the
  matching tarball, verifies its SHA256, and atomically swaps the binary.
- **Shell completion** for zsh (incl. Oh My Zsh), bash, fish, PowerShell.

### Added

- `dust scan` — concurrent cache scan with table or `--json` output. Hides
  zero-byte rows by default; `--show-empty` brings them back. Optional
  `--projects` walks dev project roots in parallel.
- `dust clean` — selects via `--all` / `--category=...` / `--item=...` /
  `--projects`. `--dry-run` previews without deleting. `--yes` skips the
  confirmation prompt. `--include-dirty` opts into projects with
  uncommitted git changes. `--prefer-tool` (default true) runs the
  canonical clean tool when available; falls back to `rm -r`.
- `dust list` — enumerates every registered cleaner ID grouped by category;
  `--categories` prints just category names.
- `dust config init|show|path` — viper-backed YAML config at
  `$XDG_CONFIG_HOME/dust/config.yaml` (defaults to `~/.config/dust/config.yaml`
  on both macOS and Linux). Env-var overrides via `DUST_*`.
- `dust upgrade` (and `dust upgrade --check`) — self-update from GitHub
  Releases with SHA256 verification.
- `dust completion <shell>` — Cobra-generated completion scripts.
- TUI keys: `↑/↓`, `j/k`, `←/→`, `h/l`, `tab`, `space`, `a`, `n`, `enter`,
  `d` (dry-run), `v` (verbose log), `r` (rescan), `?`, `q`.
- `-P` / `--projects` flag on the root command launches the TUI with the
  Projects category populated.

### Safety

- `SafeRemoveAll` refuses to delete `/`, `$HOME`, or any path outside the
  registry's allowed roots.
- Cleaners self-report availability — uninstalled tools and missing cache
  dirs are silently skipped during clean (and shown as `not installed` in
  scan when `--show-empty` is passed).
- The pre-scan summary dedupes by path to avoid double-counting the
  pnpm prune/wipe pair.
- `dust clean --projects` skips dirty git working trees by default.
- `dust upgrade` refuses to run on `dev` builds and on binaries managed by
  `go install`.

### Build / release

- `Makefile` with `build`, `run`, `test`, `vet`, `tidy`, `fmt`, `clean`,
  `snapshot`, `release-check`, `release-tag` targets.
- `.goreleaser.yaml` for cross-compiled tarballs + checksums on push of a
  `v*.*.*` tag.
- `install.sh` one-line installer (`curl ... | bash`) that detects OS+arch,
  fetches the latest release, verifies SHA256, installs to `/usr/local/bin`
  (or `~/.local/bin`).
- GitHub Actions workflow (`.github/workflows/release.yml`) runs goreleaser
  on tag push.

### Known limitations

- Windows support is stubbed out — `internal/platform/windows.go` returns
  `ErrUnsupported`. Planned for v2.
- Project scanner sizes are computed by walking artifact dirs; very deep
  monorepos may take a few seconds.
- Time Machine snapshot scan reports count, not bytes — APFS snapshots are
  sparse and don't have a single "size" we can cheaply compute.

[Unreleased]: https://github.com/ariefsn/dust/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/ariefsn/dust/releases/tag/v0.1.0
