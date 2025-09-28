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

var enableColor = shouldUseColor()

var (
	colorInfo    = ansiWrap("36") // cyan
	colorSuccess = ansiWrap("32") // green
	colorWarn    = ansiWrap("33") // yellow
	colorError   = ansiWrap("31") // red
	colorBold    = ansiWrap("1")  // bold
)

var upgradeNoticeOnce sync.Once

func ansiWrap(code string) func(string) string {
	return func(text string) string {
		if text == "" || !enableColor {
			return text
		}
		return fmt.Sprintf("\x1b[%sm%s\x1b[0m", code, text)
	}
}

func shouldUseColor() bool {
	if _, disabled := os.LookupEnv("NO_COLOR"); disabled {
		return false
	}
	return true
}

func style(text string, wrappers ...func(string) string) string {
	for _, wrap := range wrappers {
		if wrap != nil {
			text = wrap(text)
		}
	}
	return text
}

func logWith(out io.Writer, icon string, iconWrap func(string) string, message string, msgWraps ...func(string) string) {
	if out == nil {
		return
	}
	if iconWrap != nil {
		icon = iconWrap(icon)
	}
	fmt.Fprintf(out, "%s %s\n", icon, style(message, msgWraps...))
}

func logStep(out io.Writer, message string) {
	logWith(out, "➜", colorInfo, message, colorBold)
}

func logInfo(out io.Writer, message string) {
	logWith(out, "ℹ", colorInfo, message)
}

func logSuccess(out io.Writer, message string) {
	logWith(out, "✔", colorSuccess, message, colorSuccess)
}

func logWarn(out io.Writer, message string) {
	logWith(out, "!", colorWarn, message, colorWarn)
}

func logError(out io.Writer, message string) {
	logWith(out, "✖", colorError, message, colorError)
}

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
		logWarn(cmd.ErrOrStderr(), "You are running a development build. Upgrade via source control or a release build.")
	}
	statusOut := cmd.ErrOrStderr()
	stdout := cmd.OutOrStdout()

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
		logSuccess(stdout, fmt.Sprintf("You are running the latest tdb CLI (%s).", versionpkg.Display()))
		return nil
	}

	logStep(stdout, fmt.Sprintf("New version available: %s (current %s)", latest, current))
	if checkOnly {
		logInfo(stdout, "Run without --check to download and install the update.")
		return nil
	}

	asset, err := selectAsset(release)
	if err != nil {
		return err
	}
	logStep(statusOut, fmt.Sprintf("Downloading %s", asset.Name))

	tmpDir, err := os.MkdirTemp("", "tdb-upgrade-")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	archivePath, err := downloadAsset(ctx, statusOut, asset.BrowserDownloadURL, filepath.Join(tmpDir, asset.Name))
	if err != nil {
		return fmt.Errorf("download asset: %w", err)
	}
	logSuccess(statusOut, "Download complete")
	logStep(statusOut, "Extracting archive")

	newBinary, err := extractBinary(archivePath, asset.Name, tmpDir)
	if err != nil {
		return fmt.Errorf("extract binary: %w", err)
	}
	logSuccess(statusOut, "Archive extracted")

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("determine executable path: %w", err)
	}
	logStep(statusOut, "Installing update")
	if err := installBinary(newBinary, exePath, cmd); err != nil {
		return err
	}

	logSuccess(stdout, fmt.Sprintf("Successfully updated tdb to %s.", latest))
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
			logInfo(out, fmt.Sprintf("Update available: tdb CLI %s (current %s). Run \"tdb upgrade\" to install.", latest, current))
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

func downloadAsset(ctx context.Context, out io.Writer, url, dest string) (string, error) {
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
	progressWriter := newProgressWriter(f, resp.ContentLength, out, style("[download]", colorInfo, colorBold))
	defer progressWriter.finish()
	if _, err := io.Copy(progressWriter, resp.Body); err != nil {
		return "", err
	}
	return dest, nil
}

type progressWriter struct {
	dest       io.Writer
	total      int64
	written    int64
	lastUpdate time.Time
	out        io.Writer
	barWidth   int
	label      string
	lastLen    int
	finished   bool
}

func newProgressWriter(dest io.Writer, total int64, out io.Writer, label string) *progressWriter {
	pw := &progressWriter{dest: dest, total: total, out: out, barWidth: 30, label: label}
	pw.render()
	return pw
}

func (w *progressWriter) Write(p []byte) (int, error) {
	n, err := w.dest.Write(p)
	w.written += int64(n)
	if time.Since(w.lastUpdate) >= 100*time.Millisecond || w.written == w.total {
		w.render()
	}
	return n, err
}

func (w *progressWriter) render() {
	if w.out == nil {
		w.lastUpdate = time.Now()
		return
	}
	var line string
	if w.total > 0 {
		percentage := (float64(w.written) / float64(w.total)) * 100
		filled := int(float64(w.barWidth) * (float64(w.written) / float64(w.total)))
		if filled > w.barWidth {
			filled = w.barWidth
		}
		bar := strings.Repeat("#", filled) + strings.Repeat("-", w.barWidth-filled)
		line = fmt.Sprintf("%s [%s] %6.2f%% (%s/%s)", w.label, bar, percentage, humanBytes(w.written), humanBytes(w.total))
	} else {
		line = fmt.Sprintf("%s %s downloaded", w.label, humanBytes(w.written))
	}
	if len(line) < w.lastLen {
		line += strings.Repeat(" ", w.lastLen-len(line))
	}
	fmt.Fprintf(w.out, "\r%s", line)
	w.lastLen = len(line)
	w.lastUpdate = time.Now()
}

func (w *progressWriter) finish() {
	if w.out == nil || w.finished {
		return
	}
	w.render()
	fmt.Fprint(w.out, "\n")
	w.finished = true
}

func humanBytes(n int64) string {
	if n < 0 {
		return "unknown"
	}
	units := []string{"B", "KB", "MB", "GB", "TB", "PB"}
	value := float64(n)
	idx := 0
	for value >= 1024 && idx < len(units)-1 {
		value /= 1024
		idx++
	}
	if idx == 0 {
		return fmt.Sprintf("%d %s", n, units[idx])
	}
	return fmt.Sprintf("%.1f %s", value, units[idx])
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
	psPath, err := resolvePowershellPath()
	if err != nil {
		_ = os.Remove(pending)
		return fmt.Errorf("locate powershell: %w", err)
	}
	script := fmt.Sprintf(`$ErrorActionPreference = 'Stop'
$target = '%s'
$source = '%s'
$pid = %d
while (Get-Process -Id $pid -ErrorAction SilentlyContinue) { Start-Sleep -Milliseconds 200 }
Move-Item -Force -Path $source -Destination $target
`, powershellEscape(exePath), powershellEscape(pending), pid)
	cmdPS := exec.Command(psPath, "-NoProfile", "-WindowStyle", "Hidden", "-Command", script)
	// No SysProcAttr for cross-platform compatibility
	if err := cmdPS.Start(); err != nil {
		_ = os.Remove(pending)
		return fmt.Errorf("launch replacement helper: %w", err)
	}
	logInfo(cmd.OutOrStdout(), "Update scheduled. The CLI will close to complete the upgrade.")
	return nil
}

func resolvePowershellPath() (string, error) {
	if runtime.GOOS != "windows" {
		return "", errors.New("powershell resolution is only supported on Windows hosts")
	}
	candidates := windowsPowershellCandidates()
	checked := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		clean := filepath.Clean(candidate)
		if clean == "" || !filepath.IsAbs(clean) {
			continue
		}
		if _, seen := checked[clean]; seen {
			continue
		}
		checked[clean] = struct{}{}
		info, err := os.Stat(clean)
		if err != nil {
			continue
		}
		if info.IsDir() {
			continue
		}
		return clean, nil
	}
	return "", fmt.Errorf("powershell executable not found in trusted locations")
}

func windowsPowershellCandidates() []string {
	var roots []string
	if systemRoot, ok := os.LookupEnv("SystemRoot"); ok {
		if trimmed := strings.TrimSpace(systemRoot); trimmed != "" {
			roots = append(roots, trimmed)
		}
	}
	if winDir, ok := os.LookupEnv("windir"); ok {
		if trimmed := strings.TrimSpace(winDir); trimmed != "" {
			roots = append(roots, trimmed)
		}
	}
	roots = append(roots, `C:\Windows`)
	var candidates []string
	for _, root := range roots {
		cleanRoot := filepath.Clean(root)
		if cleanRoot == "" || !filepath.IsAbs(cleanRoot) {
			continue
		}
		candidates = append(candidates,
			filepath.Join(cleanRoot, "System32", "WindowsPowerShell", "v1.0", "powershell.exe"),
			filepath.Join(cleanRoot, "SysWOW64", "WindowsPowerShell", "v1.0", "powershell.exe"),
		)
	}
	return candidates
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
