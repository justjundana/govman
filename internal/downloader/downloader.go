package downloader

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	_config "github.com/justjundana/govman/internal/config"
	_golang "github.com/justjundana/govman/internal/golang"
	_logger "github.com/justjundana/govman/internal/logger"
	_progress "github.com/justjundana/govman/internal/progress"
)

type Downloader struct {
	config *_config.Config
	client *http.Client
}

// New creates a Downloader using the provided configuration.
// It initializes an HTTP client with the timeout from cfg.Download.Timeout and returns *Downloader.
func New(cfg *_config.Config) *Downloader {
	return &Downloader{
		config: cfg,
		client: &http.Client{
			Timeout: cfg.Download.Timeout,
		},
	}
}

// Download orchestrates fetching file metadata, downloading the archive, verifying its SHA-256 checksum,
// and extracting it into installDir for the specified version. Returns an error on any failure.
func (d *Downloader) Download(url, installDir, version string) error {
	_logger.InternalProgress("Retrieving file information")
	timer := _logger.StartTimer("file info retrieval")
	fileInfo, err := _golang.GetFileInfoWithConfig(version,
		d.config.GoReleases.APIURL,
		d.config.GoReleases.CacheExpiry)
	if err != nil {
		_logger.StopTimer(timer)
		return fmt.Errorf("failed to get file info: %w", err)
	}
	_logger.StopTimer(timer)

	_logger.InternalProgress("Downloading file")
	archivePath, err := d.downloadFile(url, fileInfo)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}

	// Note: We intentionally don't delete the archive here to preserve the cache.
	// Users can run 'govman clean' to manage cache when needed.

	_logger.InternalProgress("Verifying checksum")
	timer = _logger.StartTimer("checksum verification")
	if err := d.verifyChecksum(archivePath, fileInfo.Sha256); err != nil {
		_logger.StopTimer(timer)
		// Remove corrupted file from cache
		os.Remove(archivePath)
		return fmt.Errorf("checksum verification failed: %w", err)
	}
	_logger.StopTimer(timer)

	_logger.InternalProgress("Extracting archive")
	timer = _logger.StartTimer("archive extraction")
	if err := d.extractArchive(archivePath, installDir); err != nil {
		_logger.StopTimer(timer)
		return fmt.Errorf("failed to extract archive: %w", err)
	}
	_logger.StopTimer(timer)

	return nil
}

// downloadFile downloads (or resumes) the archive to the cache directory with retries and a progress bar.
// Parameters: url (download URL), fileInfo (expected file metadata). Returns the cached file path or an error.
func (d *Downloader) downloadFile(url string, fileInfo *_golang.File) (string, error) {
	filename := filepath.Base(url)
	cachePath := filepath.Join(d.config.CacheDir, filename)

	if stat, err := os.Stat(cachePath); err == nil {
		if stat.Size() == fileInfo.Size {
			_logger.Success("Using cached file: %s", filename)
			return cachePath, nil
		}
		_logger.Download("Resuming download: %s", filename)
	} else {
		_logger.Download("Downloading: %s", filename)
	}

	file, err := os.OpenFile(cachePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to create cache file: %w", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("failed to stat cache file: %w", err)
	}
	currentSize := stat.Size()

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	if currentSize > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", currentSize))
	}

	var resp *http.Response
	for attempt := 0; attempt < d.config.Download.RetryCount; attempt++ {
		resp, err = d.client.Do(req)
		if err != nil {
			if attempt < d.config.Download.RetryCount-1 {
				_logger.Warning("Download failed, retrying in %v... (%d/%d)",
					d.config.Download.RetryDelay, attempt+1, d.config.Download.RetryCount)
				time.Sleep(d.config.Download.RetryDelay)
				continue
			}
			return "", fmt.Errorf("failed to download after %d attempts: %w",
				attempt+1, err)
		}
		break
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return "", fmt.Errorf("download failed with status %d: %s", resp.StatusCode, resp.Status)
	}

	totalSize := fileInfo.Size
	if resp.StatusCode == http.StatusPartialContent {
		totalSize = currentSize + resp.ContentLength
	}

	progressBar := _progress.New(totalSize, fmt.Sprintf("Downloading %s", filename))
	if progressBar != nil {
		progressBar.Set(currentSize)
	}

	var reader io.Reader
	if progressBar != nil {
		reader = io.TeeReader(resp.Body, progressBar)
	} else {
		reader = resp.Body
	}

	if _, err := io.Copy(file, reader); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	if progressBar != nil {
		progressBar.Finish()
	}
	return cachePath, nil
}

// verifyChecksum computes the SHA-256 of filePath and compares it to expectedSHA256.
// Returns an error on mismatch or I/O failure; nil when the checksum matches.
func (d *Downloader) verifyChecksum(filePath, expectedSHA256 string) error {
	_logger.Verify("Verifying checksum...")

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}

	actualSHA256 := fmt.Sprintf("%x", hasher.Sum(nil))
	if actualSHA256 != expectedSHA256 {
		return fmt.Errorf("checksum mismatch: expected %s, got %s",
			expectedSHA256, actualSHA256)
	}

	_logger.Success("Checksum verified")
	return nil
}

// extractArchive ensures installDir exists and extracts archivePath based on its extension (.tar.gz or .zip).
// Returns an error for unsupported formats or extraction failures.
func (d *Downloader) extractArchive(archivePath, installDir string) error {
	_logger.Extract("Extracting archive...")

	if err := os.MkdirAll(installDir, 0755); err != nil {
		return fmt.Errorf("failed to create install directory: %w", err)
	}

	if strings.HasSuffix(archivePath, ".tar.gz") {
		return d.extractTarGz(archivePath, installDir)
	} else if strings.HasSuffix(archivePath, ".zip") {
		return d.extractZip(archivePath, installDir)
	}

	return fmt.Errorf("unsupported archive format")
}

// extractTarGz extracts a .tar.gz archive into installDir with path safety checks and file permissions preserved.
// Returns an error on I/O issues or unsafe paths.
func (d *Downloader) extractTarGz(archivePath, installDir string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		path := strings.TrimPrefix(header.Name, "go/")
		if path == "" {
			continue
		}

		// Path traversal protection
		if strings.Contains(path, "..") || filepath.IsAbs(path) || strings.Contains(path, "\\..") {
			return fmt.Errorf("unsafe path in archive: %s", header.Name)
		}

		// Additional check: ensure the resolved target path is within installDir
		targetPath := filepath.Join(installDir, path)
		if !strings.HasPrefix(filepath.Clean(targetPath), filepath.Clean(installDir)) {
			return fmt.Errorf("path traversal attempt detected in archive: %s", header.Name)
		}

		parentDir := filepath.Dir(targetPath)
		if _, err := os.Stat(parentDir); os.IsNotExist(err) {
			if err := os.MkdirAll(parentDir, 0755); err != nil {
				return fmt.Errorf("failed to create parent directory: %w", err)
			}
		}

		if header.Typeflag == tar.TypeDir {
			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", targetPath, err)
			}
			continue
		}

		if header.Typeflag == tar.TypeSymlink {
			// Validate symlink target for security
			linkTarget := header.Linkname
			if strings.Contains(linkTarget, "..") {
				return fmt.Errorf("unsafe symlink target in archive: %s -> %s", header.Name, linkTarget)
			}
			// Remove existing symlink if any
			os.Remove(targetPath)
			if err := os.Symlink(linkTarget, targetPath); err != nil {
				return fmt.Errorf("failed to create symlink %s -> %s: %w", targetPath, linkTarget, err)
			}
			continue
		}

		if header.Typeflag == tar.TypeReg {
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory: %w", err)
			}

			outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", targetPath, err)
			}

			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return fmt.Errorf("failed to write file %s: %w", targetPath, err)
			}
			outFile.Close()
		}
	}

	return nil
}

// extractZip extracts a .zip archive into installDir with path safety checks and directory creation as needed.
// Returns an error on I/O issues or unsafe paths.
func (d *Downloader) extractZip(archivePath, installDir string) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open zip archive: %w", err)
	}
	defer reader.Close()

	for _, file := range reader.File {
		path := file.Name
		if strings.HasPrefix(path, "go/") || strings.HasPrefix(path, "go\\") {
			path = path[3:]
		}

		if path == "" {
			continue
		}

		// Path traversal protection
		if strings.Contains(path, "..") || filepath.IsAbs(path) || strings.Contains(path, "\\..") {
			return fmt.Errorf("unsafe path in archive: %s", file.Name)
		}

		// Additional check: ensure the resolved target path is within installDir
		targetPath := filepath.Join(installDir, path)
		if !strings.HasPrefix(filepath.Clean(targetPath), filepath.Clean(installDir)) {
			return fmt.Errorf("path traversal attempt detected in archive: %s", file.Name)
		}

		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", targetPath, err)
			}
			continue
		}

		parentDir := filepath.Dir(targetPath)
		if _, err := os.Stat(parentDir); os.IsNotExist(err) {
			if err := os.MkdirAll(parentDir, 0755); err != nil {
				return fmt.Errorf("failed to create parent directory: %w", err)
			}
		}

		srcFile, err := file.Open()
		if err != nil {
			return fmt.Errorf("failed to open file in archive: %w", err)
		}

		dstFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			srcFile.Close()
			return fmt.Errorf("failed to create file %s: %w", targetPath, err)
		}

		if _, err := io.Copy(dstFile, srcFile); err != nil {
			srcFile.Close()
			dstFile.Close()
			return fmt.Errorf("failed to write file %s: %w", targetPath, err)
		}

		srcFile.Close()
		dstFile.Close()
	}

	return nil
}
