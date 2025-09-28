package cli

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	versionpkg "github.com/cubetiqlabs/tdb-cli/pkg/tdbcli/version"
)

const (
	upgradeRepoOwner  = "cubetiqlabs"
	upgradeRepoName   = "tdb-cli"
	defaultBinaryPerm = os.FileMode(0o755)
)

type githubRelease struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	Body    string `json:"body"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

var upgradeNoticeOnce sync.Once

func newUpgradeCommand() *cobra.Command {
	var checkOnly bool
	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Check for a newer CLI release and install it",
		Long: "Checks GitHub releases for a newer tdb CLI binary. If a newer release is available, " +
			"downloads the appropriate archive for your platform and replaces the current executable.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			return runUpgrade(ctx, cmd, checkOnly)
		},
	}
	cmd.Flags().BoolVar(&checkOnly, "check", false, "Only check for updates without installing")
	return cmd
}

func runUpgrade(ctx context.Context, cmd *cobra.Command, checkOnly bool) error {
	current := versionpkg.Number()
	if current == "dev" {
		fmt.Fprintln(cmd.ErrOrStderr(), "You are running a development build. Upgrade via source control or a release build.")
	}

	release, err := fetchLatestRelease(ctx)
	if err != nil {
		return fmt.Errorf("fetch latest release: %w", err)
	}
	latest := sanitizeVersion(release.TagName)
	cmp, err := compareVersions(current, latest)
	if err != nil {
		return fmt.Errorf("compare versions: %w", err)
	}

	if cmp >= 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "You are running the latest tdb CLI (%s).\n", versionpkg.Display())
		return nil
	}

	fmt.Fprintf(cmd.OutOrStdout(), "New version available: %s (current %s)\n", latest, current)
	if checkOnly {
		fmt.Fprintln(cmd.OutOrStdout(), "Run without --check to download and install the update.")
		return nil
	}

	asset, err := selectAsset(release)
	if err != nil {
		return err
	}

	tmpDir, err := os.MkdirTemp("", "tdb-upgrade-")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	archivePath, err := downloadAsset(ctx, asset.BrowserDownloadURL, filepath.Join(tmpDir, asset.Name))
	if err != nil {
		return fmt.Errorf("download asset: %w", err)
	}

	newBinary, err := extractBinary(archivePath, asset.Name, tmpDir)
	if err != nil {
		return fmt.Errorf("extract binary: %w", err)
	}

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("determine executable path: %w", err)
	}
	if err := installBinary(newBinary, exePath, cmd); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Successfully updated tdb to %s.\n", latest)
	return nil
}

func scheduleUpgradeNotice(cmd *cobra.Command) {
	if cmd == nil {
		return
	}
	// Avoid redundant notice when the user explicitly runs the upgrade command.
	if strings.EqualFold(cmd.Name(), "upgrade") {
		return
	}
	upgradeNoticeOnce.Do(func() {
		writer := cmd.ErrOrStderr()
		ctx := cmd.Context()
		go func(parent context.Context, out io.Writer) {
			if parent == nil {
				parent = context.Background()
			}
			timedCtx, cancel := context.WithTimeout(parent, 3*time.Second)
			defer cancel()
			latest, current, err := detectAvailableUpdate(timedCtx)
			if err != nil || latest == "" {
				return
			}
			fmt.Fprintf(out, "Update available: tdb CLI %s (current %s). Run \"tdb upgrade\" to install.\n", latest, current)
		}(ctx, writer)
	})
}

func detectAvailableUpdate(ctx context.Context) (string, string, error) {
	current := versionpkg.Number()
	if current == "" || current == "dev" {
		return "", current, nil
	}
	release, err := fetchLatestRelease(ctx)
	if err != nil {
		return "", current, err
	}
	latest := sanitizeVersion(release.TagName)
	if latest == "" {
		return "", current, nil
	}
	cmp, err := compareVersions(current, latest)
	if err != nil {
		return "", current, err
	}
	if cmp >= 0 {
		return "", current, nil
	}
	return latest, current, nil
}

func fetchLatestRelease(ctx context.Context) (*githubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", upgradeRepoOwner, upgradeRepoName)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", versionpkg.UserAgent())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return nil, fmt.Errorf("unexpected status %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}
	if sanitizeVersion(release.TagName) == "" {
		return nil, errors.New("latest release missing tag name")
	}
	return &release, nil
}

func selectAsset(release *githubRelease) (*struct {
	Name               string
	BrowserDownloadURL string
}, error) {
	osID := runtime.GOOS
	archID := runtime.GOARCH

	var exts []string
	switch osID {
	case "windows":
		exts = []string{"zip"}
	case "darwin":
		exts = []string{"zip", "tar.gz"}
	default:
		exts = []string{"tar.gz", "zip"}
	}

	var candidates []string
	cleanTag := sanitizeVersion(release.TagName)
	for _, ext := range exts {
		candidates = append(candidates,
			fmt.Sprintf("tdb_%s_%s.%s", osID, archID, ext),
			fmt.Sprintf("tdb_%s_%s_%s.%s", cleanTag, osID, archID, ext),
			fmt.Sprintf("tdb-%s-%s.%s", osID, archID, ext),
		)
	}
	for _, asset := range release.Assets {
		name := strings.TrimSpace(asset.Name)
		for _, candidate := range candidates {
			if strings.EqualFold(name, candidate) {
				return &struct {
					Name               string
					BrowserDownloadURL string
				}{Name: asset.Name, BrowserDownloadURL: asset.BrowserDownloadURL}, nil
			}
		}
		// Fallback: direct binary without archive
		if strings.EqualFold(name, binaryName()) {
			return &struct {
				Name               string
				BrowserDownloadURL string
			}{Name: asset.Name, BrowserDownloadURL: asset.BrowserDownloadURL}, nil
		}
	}
	return nil, fmt.Errorf("no compatible asset found for %s/%s", runtime.GOOS, runtime.GOARCH)
}

func downloadAsset(ctx context.Context, url, dest string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", versionpkg.UserAgent())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed: %s", resp.Status)
	}
	f, err := os.Create(dest)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := io.Copy(f, resp.Body); err != nil {
		return "", err
	}
	return dest, nil
}

func extractBinary(archivePath, assetName, tmpDir string) (string, error) {
	if strings.HasSuffix(strings.ToLower(assetName), ".tar.gz") || strings.HasSuffix(strings.ToLower(assetName), ".tgz") {
		return extractFromTarGz(archivePath, tmpDir)
	}
	if strings.HasSuffix(strings.ToLower(assetName), ".zip") {
		return extractFromZip(archivePath, tmpDir)
	}
	// assume raw binary
	binaryPath := filepath.Join(tmpDir, binaryName())
	if err := copyFile(archivePath, binaryPath, defaultBinaryPerm); err != nil {
		return "", err
	}
	return binaryPath, os.Chmod(binaryPath, defaultBinaryPerm)
}

func extractFromTarGz(path, tmpDir string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	gzr, err := gzip.NewReader(file)
	if err != nil {
		return "", err
	}
	defer gzr.Close()
	tarReader := tar.NewReader(gzr)
	for {
		hdr, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", err
		}
		if hdr.FileInfo().IsDir() {
			continue
		}
		base := filepath.Base(hdr.Name)
		if base != binaryName() {
			continue
		}
		dest := filepath.Join(tmpDir, base)
		out, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
		if err != nil {
			return "", err
		}
		if _, err := io.Copy(out, tarReader); err != nil {
			out.Close()
			return "", err
		}
		out.Close()
		return dest, nil
	}
	return "", errors.New("binary not found in tar archive")
}

func extractFromZip(path, tmpDir string) (string, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return "", err
	}
	defer r.Close()
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		base := filepath.Base(f.Name)
		if base != binaryName() {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return "", err
		}
		dest := filepath.Join(tmpDir, base)
		out, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
		if err != nil {
			rc.Close()
			return "", err
		}
		if _, err := io.Copy(out, rc); err != nil {
			rc.Close()
			out.Close()
			return "", err
		}
		rc.Close()
		out.Close()
		return dest, nil
	}
	return "", errors.New("binary not found in zip archive")
}

func installBinary(newBinary, exePath string, cmd *cobra.Command) error {
	if runtime.GOOS == "windows" {
		return installOnWindows(newBinary, exePath, cmd)
	}
	return installOnUnix(newBinary, exePath)
}

func installOnUnix(newBinary, exePath string) error {
	targetMode := defaultBinaryPerm
	if info, err := os.Stat(exePath); err == nil {
		perms := info.Mode().Perm()
		perms |= 0o100
		// Drop group/world write to avoid escalating privileges.
		perms &^= 0o022
		targetMode = perms
	} else if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("stat current binary: %w", err)
	}
	if err := os.Chmod(newBinary, targetMode); err != nil {
		return err
	}
	backup := exePath + ".bak"
	_ = os.Remove(backup)
	if err := os.Rename(exePath, backup); err != nil {
		// check if the error is because of permission denied
		if os.IsPermission(err) {
			return fmt.Errorf("permission denied when renaming current binary. Please run the upgrade command with sufficient permissions (e.g. using sudo)")
		}
		return fmt.Errorf("backup current binary: %w", err)
	}
	if err := os.Rename(newBinary, exePath); err != nil {
		if os.IsPermission(err) {
			_ = os.Rename(backup, exePath)
			return fmt.Errorf("permission denied when installing new binary. Please run the upgrade command with sufficient permissions (e.g. using sudo)")
		}
		var linkErr *os.LinkError
		if errors.As(err, &linkErr) && errors.Is(linkErr.Err, syscall.EXDEV) {
			if copyErr := copyFile(newBinary, exePath, targetMode); copyErr != nil {
				_ = os.Rename(backup, exePath)
				return fmt.Errorf("install new binary (copy fallback): %w", copyErr)
			}
			_ = os.Remove(backup)
			return nil
		}
		_ = os.Rename(backup, exePath)
		return fmt.Errorf("install new binary: %w", err)
	}
	// cleanup backup
	_ = os.Remove(backup)
	_ = os.Chmod(exePath, targetMode)
	return nil
}

func installOnWindows(newBinary, exePath string, cmd *cobra.Command) error {
	if err := os.Chmod(newBinary, 0o755); err != nil {
		return err
	}
	pending := exePath + ".new"
	if err := copyFile(newBinary, pending, defaultBinaryPerm); err != nil {
		// check if the error is because of permission denied
		if os.IsPermission(err) {
			return fmt.Errorf("permission denied when preparing new binary. Please run the upgrade command with sufficient permissions (e.g. Run as Administrator)")
		}
		return fmt.Errorf("prepare replacement: %w", err)
	}
	pid := os.Getpid()
	script := fmt.Sprintf(`$ErrorActionPreference = 'Stop'
$target = '%s'
$source = '%s'
$pid = %d
while (Get-Process -Id $pid -ErrorAction SilentlyContinue) { Start-Sleep -Milliseconds 200 }
Move-Item -Force -Path $source -Destination $target
`, powershellEscape(exePath), powershellEscape(pending), pid)
	cmdPS := exec.Command("powershell", "-NoProfile", "-WindowStyle", "Hidden", "-Command", script)
	// No SysProcAttr for cross-platform compatibility
	if err := cmdPS.Start(); err != nil {
		return fmt.Errorf("launch replacement helper: %w", err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Update scheduled. The CLI will close to complete the upgrade.\n")
	return nil
}

func copyFile(src, dst string, perm os.FileMode) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, perm)
	if err != nil {
		return err
	}
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		dstFile.Close()
		return err
	}
	if err := dstFile.Chmod(perm); err != nil {
		dstFile.Close()
		return err
	}
	return dstFile.Close()
}

func binaryName() string {
	if runtime.GOOS == "windows" {
		return "tdb.exe"
	}
	return "tdb"
}

func sanitizeVersion(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	v = strings.TrimPrefix(v, "V")
	return v
}

func compareVersions(current, latest string) (int, error) {
	curParts, err := parseVersionParts(current)
	if err != nil {
		return 0, err
	}
	latestParts, err := parseVersionParts(latest)
	if err != nil {
		return 0, err
	}
	for i := 0; i < len(curParts); i++ {
		if curParts[i] < latestParts[i] {
			return -1, nil
		}
		if curParts[i] > latestParts[i] {
			return 1, nil
		}
	}
	return 0, nil
}

func parseVersionParts(v string) ([3]int, error) {
	var parts [3]int
	v = sanitizeVersion(v)
	if v == "" {
		return parts, nil
	}
	main := v
	if idx := strings.IndexAny(main, "-+"); idx != -1 {
		main = main[:idx]
	}
	segments := strings.Split(main, ".")
	for i := 0; i < len(segments) && i < 3; i++ {
		if segments[i] == "" {
			continue
		}
		var value int
		fmt.Sscanf(segments[i], "%d", &value)
		parts[i] = value
	}
	return parts, nil
}

func powershellEscape(path string) string {
	return strings.ReplaceAll(path, "'", "''")
}
