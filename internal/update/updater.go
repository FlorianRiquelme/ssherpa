package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
)

type installMethod int

const (
	installBinary   installMethod = iota
	installHomebrew
)

// detectInstallMethod determines how ssherpa was installed.
func detectInstallMethod(binaryPath string) installMethod {
	lower := strings.ToLower(binaryPath)
	if strings.Contains(lower, "/cellar/") || strings.Contains(lower, "/homebrew/") {
		return installHomebrew
	}
	return installBinary
}

// archiveURL returns the download URL for a release archive.
func archiveURL(version, goos, goarch string) string {
	return fmt.Sprintf(
		"https://github.com/%s/%s/releases/download/v%s/ssherpa_%s_%s_%s.tar.gz",
		repoOwner, repoName, version, version, goos, goarch,
	)
}

// checksumsURL returns the checksums.txt URL for a release.
func checksumsURL(version string) string {
	return fmt.Sprintf(
		"https://github.com/%s/%s/releases/download/v%s/checksums.txt",
		repoOwner, repoName, version,
	)
}

// verifyChecksum checks SHA256 of data against an expected hex string.
func verifyChecksum(data []byte, expectedHex string) bool {
	sum := sha256.Sum256(data)
	actual := fmt.Sprintf("%x", sum)
	return actual == strings.ToLower(expectedHex)
}

// findChecksumForFile extracts the hash for a specific filename from checksums.txt content.
func findChecksumForFile(checksumContent, filename string) (string, error) {
	for _, line := range strings.Split(checksumContent, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Format: "hash  filename" (two spaces)
		parts := strings.Fields(line)
		if len(parts) == 2 && parts[1] == filename {
			return parts[0], nil
		}
	}
	return "", fmt.Errorf("checksum not found for %s", filename)
}

// PerformUpdate downloads and installs the new version, then restarts.
// This function does not return on success (exec replaces the process).
// Returns an error if the update fails at any step.
func PerformUpdate(version string) error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot determine binary path: %w", err)
	}
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return fmt.Errorf("cannot resolve binary path: %w", err)
	}

	method := detectInstallMethod(exePath)

	switch method {
	case installHomebrew:
		return updateViaHomebrew()
	default:
		return updateViaBinary(version, exePath)
	}
}

// updateViaHomebrew runs brew upgrade and restarts.
func updateViaHomebrew() error {
	cmd := exec.Command("brew", "upgrade", "ssherpa")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("brew upgrade failed: %w", err)
	}

	// Find the new binary path
	newPath, err := exec.LookPath("ssherpa")
	if err != nil {
		return fmt.Errorf("cannot find ssherpa after brew upgrade: %w", err)
	}

	return restartBinary(newPath)
}

// updateViaBinary downloads, verifies, and replaces the binary.
func updateViaBinary(version, currentPath string) error {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	archiveName := fmt.Sprintf("ssherpa_%s_%s_%s.tar.gz", version, goos, goarch)

	// Download checksums
	checksumsBody, err := fetchBody(checksumsURL(version))
	if err != nil {
		return fmt.Errorf("failed to download checksums: %w", err)
	}

	expectedHash, err := findChecksumForFile(checksumsBody, archiveName)
	if err != nil {
		return fmt.Errorf("checksum lookup failed: %w", err)
	}

	// Download archive
	archiveData, err := downloadBytes(archiveURL(version, goos, goarch))
	if err != nil {
		return fmt.Errorf("failed to download archive: %w", err)
	}

	// Verify checksum
	if !verifyChecksum(archiveData, expectedHash) {
		return fmt.Errorf("checksum verification failed for %s", archiveName)
	}

	// Extract binary from tar.gz
	binaryData, err := extractBinaryFromTarGz(archiveData, "ssherpa")
	if err != nil {
		return fmt.Errorf("failed to extract binary: %w", err)
	}

	// Write to temp file in same directory (for atomic rename)
	dir := filepath.Dir(currentPath)
	tmpFile, err := os.CreateTemp(dir, "ssherpa-update-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	if _, err := tmpFile.Write(binaryData); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("failed to write temp binary: %w", err)
	}
	tmpFile.Close()

	// Make executable
	if err := os.Chmod(tmpPath, 0o755); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to chmod: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, currentPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to replace binary (permission denied? try: sudo mv %s %s): %w", tmpPath, currentPath, err)
	}

	return restartBinary(currentPath)
}

// restartBinary replaces the current process with the new binary.
func restartBinary(path string) error {
	return syscall.Exec(path, os.Args, os.Environ())
}

// downloadBytes fetches a URL and returns the raw bytes.
func downloadBytes(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "ssherpa-update-checker")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// extractBinaryFromTarGz extracts a named file from a tar.gz archive in memory.
func extractBinaryFromTarGz(data []byte, binaryName string) ([]byte, error) {
	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		// Match the binary name (may be in a subdirectory)
		name := filepath.Base(header.Name)
		if name == binaryName && header.Typeflag == tar.TypeReg {
			return io.ReadAll(tr)
		}
	}

	return nil, fmt.Errorf("binary %q not found in archive", binaryName)
}
