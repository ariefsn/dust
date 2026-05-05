package cli

import (
	"fmt"
	"os"

	"github.com/ariefsn/dust/internal/upgrade"
	"github.com/spf13/cobra"
)

func newUpgradeCmd() *cobra.Command {
	var (
		check bool
		repo  string
	)
	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade dust in place from the latest GitHub release",
		Long: `Upgrade dust in place by downloading the latest GitHub release, verifying
its SHA256 checksum, and atomically replacing the running binary.

By default, dust upgrade refuses to run when the binary appears to be installed
via 'go install' — use 'go install github.com/ariefsn/dust/cmd/dust@latest'
in that case. It also refuses if the current build is 'dev' (built from
source without a tagged version), since there's no reliable comparison.

Pass --check to see whether a newer release is available without modifying
anything.`,
		Example: `  dust upgrade --check
  dust upgrade
  DUST_REPO=myfork/dust dust upgrade`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if envRepo := os.Getenv("DUST_REPO"); envRepo != "" && repo == "" {
				repo = envRepo
			}
			if repo == "" {
				repo = upgrade.DefaultRepo
			}

			binPath, err := upgrade.CurrentBinaryPath()
			if err != nil {
				return fmt.Errorf("locate running binary: %w", err)
			}

			goOS, goArch := upgrade.CurrentPlatform()
			cmd.Printf("current: dust %s (%s/%s) at %s\n", version, goOS, goArch, binPath)

			rel, err := upgrade.LatestRelease(repo, goOS, goArch)
			if err != nil {
				return err
			}
			cmd.Printf("latest:  %s\n", rel.Tag)

			if !upgrade.IsNewer(version, rel.Tag) {
				if version == "dev" {
					cmd.Println("dust is a dev build — upgrade skipped. Install a release first:")
					cmd.Println("  curl -fsSL https://raw.githubusercontent.com/" + repo + "/main/install.sh | bash")
					return nil
				}
				cmd.Println("Already up to date.")
				return nil
			}

			if check {
				cmd.Printf("\nA newer version is available: %s -> %s\n", version, rel.Tag)
				cmd.Println("Run 'dust upgrade' to install it.")
				cmd.Println("Release notes:", rel.HTMLURL)
				return nil
			}

			if upgrade.IsManagedByGoInstall(binPath) {
				return fmt.Errorf("dust appears to be managed by `go install` (binary in $GOPATH/bin or $GOBIN).\n"+
					"Run this instead:\n  go install github.com/%s/cmd/dust@latest", repo)
			}

			if err := rel.EnsureAssets(goOS, goArch); err != nil {
				return err
			}

			cmd.Printf("\nDownloading %s for %s/%s...\n", rel.Tag, goOS, goArch)
			tarballPath, tempDir, err := upgrade.DownloadAndVerify(rel)
			if err != nil {
				return err
			}
			defer os.RemoveAll(tempDir)

			cmd.Println("Checksum verified.")

			newBin, err := upgrade.ExtractBinary(tarballPath, tempDir)
			if err != nil {
				return err
			}

			if err := upgrade.Replace(binPath, newBin); err != nil {
				return err
			}

			cmd.Printf("Upgraded dust to %s.\n", rel.Tag)
			cmd.Println("Re-run any in-flight `dust` commands to pick up the new binary.")
			return nil
		},
	}
	cmd.Flags().BoolVar(&check, "check", false, "only check for a newer release; don't modify anything")
	cmd.Flags().StringVar(&repo, "repo", "", "override repo (default: env DUST_REPO or "+upgrade.DefaultRepo+")")
	return cmd
}
