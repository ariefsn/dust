// Package upgrade implements `dust upgrade` — checks GitHub Releases for a
// newer version, downloads the matching platform tarball, verifies its
// SHA256 checksum, and atomically replaces the running binary.
package upgrade

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// DefaultRepo is the GitHub repo dust ships from. Override via DUST_REPO env
// var if you fork it.
const DefaultRepo = "ariefsn/dust"

// Release describes a single GitHub release relevant to self-update.
type Release struct {
	Tag        string
	HTMLURL    string
	ArchiveURL string // tarball for current platform
	ChecksumsURL string
}

// LatestRelease returns metadata for the latest release of the given repo.
// `repo` is the "owner/name" form (e.g. "ariefsn/dust"). `currentOS` /
// `currentArch` should be runtime.GOOS / runtime.GOARCH; passing them in
// keeps the function testable.
func LatestRelease(repo, currentOS, currentArch string) (*Release, error) {
	if repo == "" {
		repo = DefaultRepo
	}
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "dust-self-update")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("no releases found at github.com/%s — has the project published one yet?", repo)
	}
	if resp.StatusCode/100 != 2 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("github API returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var payload struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
		Assets  []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("parse github api response: %w", err)
	}

	// goreleaser's name_template is `dust_<version>_<os>_<arch>.tar.gz`, with
	// `<version>` = tag without the leading `v`.
	plainVersion := strings.TrimPrefix(payload.TagName, "v")
	wantArchive := fmt.Sprintf("dust_%s_%s_%s.tar.gz", plainVersion, currentOS, currentArch)
	wantChecksums := "checksums.txt"

	rel := &Release{Tag: payload.TagName, HTMLURL: payload.HTMLURL}
	for _, a := range payload.Assets {
		switch a.Name {
		case wantArchive:
			rel.ArchiveURL = a.BrowserDownloadURL
		case wantChecksums:
			rel.ChecksumsURL = a.BrowserDownloadURL
		}
	}
	return rel, nil
}

// EnsureAssets validates that the release carries the platform tarball and
// checksums file we need to install it. Caller invokes this only when an
// actual install is about to happen (skipping it for --check + dev-build
// flows that just want the tag string).
func (r *Release) EnsureAssets(currentOS, currentArch string) error {
	plainVersion := strings.TrimPrefix(r.Tag, "v")
	wantArchive := fmt.Sprintf("dust_%s_%s_%s.tar.gz", plainVersion, currentOS, currentArch)
	if r.ArchiveURL == "" {
		return fmt.Errorf("release %s has no archive for %s/%s (looked for %q)", r.Tag, currentOS, currentArch, wantArchive)
	}
	if r.ChecksumsURL == "" {
		return fmt.Errorf("release %s is missing checksums.txt", r.Tag)
	}
	return nil
}

// IsNewer compares two version strings (e.g. "v0.1.0" vs "v0.2.1") and reports
// whether `latest` is strictly greater than `current`. Returns false if either
// is unparseable, so a "dev" build never reports an available upgrade.
func IsNewer(current, latest string) bool {
	c, ok1 := parseSemver(current)
	l, ok2 := parseSemver(latest)
	if !ok1 || !ok2 {
		return false
	}
	for i := 0; i < 3; i++ {
		if l[i] != c[i] {
			return l[i] > c[i]
		}
	}
	return false
}

func parseSemver(s string) ([3]int, bool) {
	var out [3]int
	s = strings.TrimPrefix(s, "v")
	if s == "" || s == "dev" {
		return out, false
	}
	// Strip any pre-release / build metadata after a `-` or `+`.
	if i := strings.IndexAny(s, "-+"); i >= 0 {
		s = s[:i]
	}
	parts := strings.SplitN(s, ".", 3)
	if len(parts) != 3 {
		return out, false
	}
	for i, p := range parts {
		var n int
		_, err := fmt.Sscanf(p, "%d", &n)
		if err != nil {
			return out, false
		}
		out[i] = n
	}
	return out, true
}

// DownloadAndVerify fetches the release archive + checksums, verifies the
// SHA256, and returns the path to the verified tarball in a tempdir.
//
// Caller must remove the returned tempdir when done.
func DownloadAndVerify(rel *Release) (tarballPath, tempDir string, err error) {
	tempDir, err = os.MkdirTemp("", "dust-upgrade-*")
	if err != nil {
		return "", "", err
	}
	cleanup := func() { os.RemoveAll(tempDir) }

	archiveName := filepath.Base(strings.SplitN(rel.ArchiveURL, "?", 2)[0])
	tarballPath = filepath.Join(tempDir, archiveName)
	if err := download(rel.ArchiveURL, tarballPath); err != nil {
		cleanup()
		return "", "", fmt.Errorf("download archive: %w", err)
	}

	sumsPath := filepath.Join(tempDir, "checksums.txt")
	if err := download(rel.ChecksumsURL, sumsPath); err != nil {
		cleanup()
		return "", "", fmt.Errorf("download checksums: %w", err)
	}

	expected, err := readChecksum(sumsPath, archiveName)
	if err != nil {
		cleanup()
		return "", "", err
	}
	actual, err := sha256File(tarballPath)
	if err != nil {
		cleanup()
		return "", "", err
	}
	if !strings.EqualFold(expected, actual) {
		cleanup()
		return "", "", fmt.Errorf("checksum mismatch: expected %s, got %s", expected, actual)
	}
	return tarballPath, tempDir, nil
}

// ExtractBinary extracts the `dust` binary from `tarballPath` into `tempDir`
// and returns its path.
func ExtractBinary(tarballPath, tempDir string) (string, error) {
	f, err := os.Open(tarballPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return "", err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", fmt.Errorf("tar read: %w", err)
		}
		// Match the binary by basename — guards against a future archive that
		// nests it in a subdir.
		if filepath.Base(hdr.Name) != "dust" || hdr.Typeflag != tar.TypeReg {
			continue
		}
		out := filepath.Join(tempDir, "dust")
		dst, err := os.OpenFile(out, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
		if err != nil {
			return "", err
		}
		if _, err := io.Copy(dst, tr); err != nil {
			dst.Close()
			return "", err
		}
		if err := dst.Close(); err != nil {
			return "", err
		}
		return out, nil
	}
	return "", errors.New("dust binary not found in archive")
}

// Replace atomically swaps the running binary with the new one. On macOS and
// Linux, os.Rename across the same filesystem is atomic. We rename the
// existing binary aside first so the caller can rollback on failure.
func Replace(currentPath, newPath string) error {
	dir := filepath.Dir(currentPath)
	backup := filepath.Join(dir, ".dust.old")

	// Move the existing binary aside (best effort — Linux allows replacing a
	// running binary; macOS does too).
	if err := os.Rename(currentPath, backup); err != nil {
		// On some shared filesystems Rename returns "cross-device link" — fall
		// back to a manual swap via temp file in the same dir.
		if !os.IsNotExist(err) && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("move running binary aside: %w", err)
		}
	}

	if err := moveAcross(newPath, currentPath); err != nil {
		// Try to roll back.
		_ = os.Rename(backup, currentPath)
		return fmt.Errorf("install new binary: %w", err)
	}

	// Best-effort cleanup of the old binary.
	_ = os.Remove(backup)
	return nil
}

// CurrentBinaryPath returns the absolute path of the currently running
// executable, resolving symlinks. Used to know what to overwrite.
func CurrentBinaryPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return exe, nil
	}
	return resolved, nil
}

// IsManagedByGoInstall reports whether the running binary appears to live
// under $GOPATH/bin or $GOBIN — the canonical `go install` locations. We
// refuse to self-update those because Go's tool chain expects to manage them.
func IsManagedByGoInstall(binPath string) bool {
	if binPath == "" {
		return false
	}
	gobin := os.Getenv("GOBIN")
	if gobin != "" && strings.HasPrefix(binPath, gobin+string(filepath.Separator)) {
		return true
	}
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		if home, err := os.UserHomeDir(); err == nil {
			gopath = filepath.Join(home, "go")
		}
	}
	if gopath != "" {
		gopathBin := filepath.Join(gopath, "bin") + string(filepath.Separator)
		if strings.HasPrefix(binPath, gopathBin) {
			return true
		}
	}
	return false
}

// CurrentPlatform returns the GOOS/GOARCH for the running binary.
func CurrentPlatform() (string, string) {
	return runtime.GOOS, runtime.GOARCH
}

// --- internal helpers --------------------------------------------------------

func download(url, dst string) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "dust-self-update")

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	return err
}

func readChecksum(sumsPath, archiveName string) (string, error) {
	data, err := os.ReadFile(sumsPath)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		// goreleaser's checksums.txt format: "<sha256>  <filename>"
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) >= 2 && fields[len(fields)-1] == archiveName {
			return fields[0], nil
		}
	}
	return "", fmt.Errorf("no checksum entry for %q in checksums.txt", archiveName)
}

func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// moveAcross renames if possible; falls back to copy+remove for cross-fs moves.
func moveAcross(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Remove(src)
}
