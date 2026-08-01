package downloader

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	neturl "net/url"
	"os"
	pathpkg "path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	_config "github.com/justjundana/govman/internal/config"
	_golang "github.com/justjundana/govman/internal/golang"
	_logger "github.com/justjundana/govman/internal/logger"
	_progress "github.com/justjundana/govman/internal/progress"
)

// maxExtractFileSize is the maximum size allowed per file during archive extraction (2 GB).
// This prevents zip bomb attacks from exhausting disk space.
const (
	maxExtractFileSize  = 2 << 30  // 2 GB
	maxExtractTotalSize = 10 << 30 // 10 GB
	maxArchiveEntries   = 100000
	legacyTarTypeReg    = byte(0)
)

type Downloader struct {
	config            *_config.Config
	client            *http.Client
	logger            *_logger.Logger
	maxFileSize       int64
	maxTotalSize      int64
	maxArchiveEntries int
}

// New creates a Downloader using the provided configuration.
// It initializes an HTTP client with the timeout from cfg.Download.Timeout and returns *Downloader.
func New(cfg *_config.Config) *Downloader {
	return NewWithLogger(cfg, _logger.Get())
}

// NewWithLogger creates a Downloader whose output is isolated to logger.
func NewWithLogger(cfg *_config.Config, logger *_logger.Logger) *Downloader {
	if logger == nil {
		logger = _logger.New()
	}
	return &Downloader{
		config: cfg,
		client: &http.Client{
			Timeout: cfg.Download.Timeout,
		},
		logger:            logger,
		maxFileSize:       maxExtractFileSize,
		maxTotalSize:      maxExtractTotalSize,
		maxArchiveEntries: maxArchiveEntries,
	}
}

// Download orchestrates fetching file metadata, downloading the archive, verifying its SHA-256 checksum,
// and extracting it into installDir for the specified version. Returns an error on any failure.
func (d *Downloader) Download(url, installDir, version string) error {
	return d.DownloadContext(context.Background(), url, installDir, version)
}

// DownloadContext is Download with caller-controlled cancellation.
func (d *Downloader) DownloadContext(ctx context.Context, url, installDir, version string) error {
	d.logger.InternalProgress("Retrieving file information")
	timer := d.logger.StartTimer("file info retrieval")
	fileInfo, err := _golang.GetFileInfoWithConfig(version,
		d.config.GoReleases.APIURL,
		d.config.GoReleases.CacheExpiry)
	if err != nil {
		d.logger.StopTimer(timer)
		return fmt.Errorf("failed to get file info: %w", err)
	}
	d.logger.StopTimer(timer)

	var archivePath string
	for attempt := 0; attempt < 2; attempt++ {
		d.logger.InternalProgress("Downloading file")
		archivePath, err = d.downloadFileContext(ctx, url, fileInfo)
		if err != nil {
			return fmt.Errorf("failed to download: %w", err)
		}

		d.logger.InternalProgress("Verifying checksum")
		timer = d.logger.StartTimer("checksum verification")
		err = d.verifyChecksum(archivePath, fileInfo.Sha256)
		d.logger.StopTimer(timer)
		if err == nil {
			break
		}
		if removeErr := os.Remove(archivePath); removeErr != nil && !os.IsNotExist(removeErr) {
			return fmt.Errorf("checksum verification failed: %w (also failed to remove corrupt cache: %v)", err, removeErr)
		}
		if attempt == 1 {
			return fmt.Errorf("checksum verification failed after a fresh download: %w", err)
		}
		d.logger.Warning("Cached archive checksum was invalid; retrying with a fresh download")
	}

	d.logger.InternalProgress("Extracting archive")
	timer = d.logger.StartTimer("archive extraction")
	if err := d.installArchive(archivePath, installDir, version); err != nil {
		d.logger.StopTimer(timer)
		return fmt.Errorf("failed to extract archive: %w", err)
	}
	d.logger.StopTimer(timer)

	return nil
}

// handleResumeResponse checks if the server supports resume and truncates the file if needed.
func (d *Downloader) handleResumeResponse(file *os.File, resp *http.Response, currentSize, expectedSize int64) (int64, error) {
	if currentSize > 0 && resp.StatusCode == http.StatusOK {
		d.logger.Verbose("Server does not support resume, restarting download from scratch")
		if err := file.Truncate(0); err != nil {
			return 0, fmt.Errorf("failed to truncate file for fresh download: %w", err)
		}
		if _, err := file.Seek(0, 0); err != nil {
			return 0, fmt.Errorf("failed to seek to beginning of file: %w", err)
		}
		return 0, nil
	}
	if resp.StatusCode == http.StatusPartialContent {
		start, end, total, err := parseContentRange(resp.Header.Get("Content-Range"))
		if err != nil {
			return 0, fmt.Errorf("invalid resume response: %w", err)
		}
		if start != currentSize || end < start {
			return 0, fmt.Errorf("invalid resume range: server returned bytes %d-%d, expected start %d", start, end, currentSize)
		}
		if expectedSize > 0 && total != expectedSize {
			return 0, fmt.Errorf("invalid resume total: server returned %d, expected %d", total, expectedSize)
		}
		if expectedSize > 0 && end != expectedSize-1 {
			return 0, fmt.Errorf("invalid resume end: server returned %d, expected %d", end, expectedSize-1)
		}
		if resp.ContentLength >= 0 && resp.ContentLength != end-start+1 {
			return 0, fmt.Errorf("invalid resume length: server declared %d bytes for range %d-%d", resp.ContentLength, start, end)
		}
	}
	return currentSize, nil
}

func parseContentRange(value string) (start, end, total int64, err error) {
	if !strings.HasPrefix(value, "bytes ") {
		return 0, 0, 0, fmt.Errorf("missing or unsupported Content-Range %q", value)
	}
	parts := strings.Split(strings.TrimPrefix(value, "bytes "), "/")
	if len(parts) != 2 || parts[1] == "*" {
		return 0, 0, 0, fmt.Errorf("malformed Content-Range %q", value)
	}
	rangeParts := strings.Split(parts[0], "-")
	if len(rangeParts) != 2 {
		return 0, 0, 0, fmt.Errorf("malformed Content-Range %q", value)
	}
	start, err = strconv.ParseInt(rangeParts[0], 10, 64)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid range start: %w", err)
	}
	end, err = strconv.ParseInt(rangeParts[1], 10, 64)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid range end: %w", err)
	}
	total, err = strconv.ParseInt(parts[1], 10, 64)
	if err != nil || total <= 0 || start < 0 || end < start || end >= total {
		return 0, 0, 0, fmt.Errorf("invalid range total in %q", value)
	}
	return start, end, total, nil
}

// downloadWithRetry performs the HTTP download with retry logic.
func (d *Downloader) downloadWithRetry(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error
	attempts := d.config.Download.RetryCount
	if attempts < 1 {
		attempts = 1
	}
	for attempt := 0; attempt < attempts; attempt++ {
		// #nosec G704 -- request URLs are parsed and restricted to HTTP(S) before this helper; custom release endpoints are an intentional configuration contract.
		resp, err = d.client.Do(req)
		if err == nil {
			retryable := resp.StatusCode == http.StatusRequestTimeout || resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500
			if !retryable || attempt == attempts-1 {
				return resp, nil
			}
			retryAfter := retryDelay(resp.Header.Get("Retry-After"), time.Now())
			_ = resp.Body.Close()
			if retryAfter > 0 {
				if err := waitForRetry(req.Context(), retryAfter); err != nil {
					return nil, err
				}
				continue
			}
		}
		if attempt < attempts-1 {
			delay := d.config.Download.RetryDelay
			if delay < 0 {
				delay = 0
			}
			d.logger.Warning("Download failed, retrying in %v... (%d/%d)",
				delay, attempt+1, attempts)
			if err := waitForRetry(req.Context(), delay); err != nil {
				return nil, err
			}
		}
	}
	return nil, fmt.Errorf("failed to download after %d attempts: %w", attempts, err)
}

func retryDelay(value string, now time.Time) time.Duration {
	if value == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(value); err == nil {
		if seconds < 0 {
			return 0
		}
		return min(time.Duration(seconds)*time.Second, time.Minute)
	}
	when, err := http.ParseTime(value)
	if err != nil || !when.After(now) {
		return 0
	}
	return min(when.Sub(now), time.Minute)
}

func waitForRetry(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// setupProgressReader wraps the response body with a progress bar if available.
func setupProgressReader(body io.Reader, totalSize, currentSize int64, filename string, enabled bool) (io.Reader, *_progress.ProgressBar) {
	progressBar := _progress.NewWithWriter(totalSize, fmt.Sprintf("Downloading %s", filename), os.Stderr, enabled)
	if progressBar != nil {
		progressBar.Set(currentSize)
		return io.TeeReader(body, progressBar), progressBar
	}
	return body, nil
}

// downloadFile downloads (or resumes) the archive to the cache directory with retries and a progress bar.
// Parameters: url (download URL), fileInfo (expected file metadata). Returns the cached file path or an error.
func (d *Downloader) downloadFile(url string, fileInfo *_golang.File) (string, error) {
	return d.downloadFileContext(context.Background(), url, fileInfo)
}

func (d *Downloader) downloadFileContext(ctx context.Context, url string, fileInfo *_golang.File) (resultPath string, resultErr error) {
	parsedURL, err := neturl.Parse(url)
	if err != nil {
		return "", fmt.Errorf("invalid download URL: %w", err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return "", fmt.Errorf("unsupported download URL scheme: %s", parsedURL.Scheme)
	}
	filename := pathpkg.Base(parsedURL.Path)
	if filename == "." || filename == "/" || filename == "" {
		return "", fmt.Errorf("download URL does not contain a filename")
	}
	if fileInfo == nil || fileInfo.Size <= 0 || fileInfo.Filename == "" {
		return "", fmt.Errorf("invalid download metadata")
	}
	if strings.ContainsAny(fileInfo.Filename, `/\\`) || strings.ContainsRune(fileInfo.Filename, '\x00') || fileInfo.Filename == "." || fileInfo.Filename == ".." {
		return "", fmt.Errorf("unsafe download filename in release metadata: %q", fileInfo.Filename)
	}
	if filename != fileInfo.Filename {
		return "", fmt.Errorf("download filename %q does not match release metadata %q", filename, fileInfo.Filename)
	}
	if err := os.MkdirAll(d.config.CacheDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create cache directory: %w", err)
	}
	cachePath := filepath.Join(d.config.CacheDir, filename)
	resumePath := cachePath + ".partial"
	lockPath := cachePath + ".lock"
	lockWait := d.config.Download.Timeout*time.Duration(max(d.config.Download.RetryCount, 1)) +
		d.config.Download.RetryDelay*time.Duration(max(d.config.Download.RetryCount-1, 0))
	if lockWait <= 0 {
		lockWait = 2 * time.Minute
	}
	lockWait = min(lockWait, 15*time.Minute)
	unlock, err := acquireCacheLock(ctx, lockPath, lockWait)
	if err != nil {
		return "", err
	}
	defer func() {
		if unlockErr := unlock(); unlockErr != nil {
			resultErr = errors.Join(resultErr, unlockErr)
		}
	}()

	if stat, err := os.Lstat(cachePath); err == nil && stat.Mode().IsRegular() && stat.Size() == fileInfo.Size {
		d.logger.Success("Using cached file: %s", filename)
		return cachePath, nil
	} else if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("failed to inspect cached file: %w", err)
	}

	file, err := os.CreateTemp(d.config.CacheDir, "."+filename+".partial-*")
	if err != nil {
		return "", fmt.Errorf("failed to create partial cache file: %w", err)
	}
	partialPath := file.Name()
	keepForResume := true
	fileClosed := false
	defer func() {
		if !fileClosed {
			if closeErr := file.Close(); resultErr == nil && closeErr != nil {
				resultErr = fmt.Errorf("failed to close partial cache file: %w", closeErr)
			}
		}
		if keepForResume {
			if stat, statErr := os.Stat(partialPath); statErr == nil && stat.Size() > 0 && stat.Size() < fileInfo.Size {
				if renameErr := replaceFile(partialPath, resumePath); renameErr == nil {
					return
				}
			}
		}
		if removeErr := os.Remove(partialPath); removeErr != nil && !os.IsNotExist(removeErr) {
			resultErr = errors.Join(resultErr, fmt.Errorf("failed to clean partial cache file: %w", removeErr))
		}
	}()

	currentSize, err := seedPartialDownload(file, resumePath, cachePath, fileInfo.Size)
	if err != nil {
		return "", err
	}
	if currentSize > 0 {
		d.logger.Download("Resuming download: %s", filename)
	} else {
		d.logger.Download("Downloading: %s", filename)
	}

	for requestAttempt := 0; requestAttempt < 2; requestAttempt++ {
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if reqErr != nil {
			return "", fmt.Errorf("failed to create request: %w", reqErr)
		}
		if currentSize > 0 {
			req.Header.Set("Range", fmt.Sprintf("bytes=%d-", currentSize))
		}

		resp, requestErr := d.downloadWithRetry(req)
		if requestErr != nil {
			return "", requestErr
		}
		if resp.StatusCode == http.StatusRequestedRangeNotSatisfiable && currentSize > 0 {
			_ = resp.Body.Close()
			if err := resetPartialFile(file); err != nil {
				return "", err
			}
			currentSize = 0
			continue
		}
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
			_ = resp.Body.Close()
			return "", fmt.Errorf("download failed with status %d: %s", resp.StatusCode, resp.Status)
		}

		validatedSize, resumeErr := d.handleResumeResponse(file, resp, currentSize, fileInfo.Size)
		if resumeErr != nil && currentSize > 0 && requestAttempt == 0 {
			_ = resp.Body.Close()
			d.logger.Warning("Server returned an invalid resume response; restarting download")
			if err := resetPartialFile(file); err != nil {
				return "", err
			}
			currentSize = 0
			continue
		}
		if resumeErr != nil {
			_ = resp.Body.Close()
			return "", resumeErr
		}
		currentSize = validatedSize
		progressEnabled := !d.config.Quiet && _progress.IsTerminalWriter(os.Stderr)
		reader, progressBar := setupProgressReader(resp.Body, fileInfo.Size, currentSize, filename, progressEnabled)
		remaining := fileInfo.Size - currentSize
		written, copyErr := io.Copy(file, io.LimitReader(reader, remaining+1))
		closeErr := resp.Body.Close()
		if progressBar != nil {
			progressBar.Finish()
		}
		if copyErr != nil {
			return "", fmt.Errorf("failed to write file: %w", copyErr)
		}
		if closeErr != nil {
			return "", fmt.Errorf("failed to close download response: %w", closeErr)
		}
		if written != remaining {
			if written > remaining {
				keepForResume = false
				return "", fmt.Errorf("download exceeded expected size %d", fileInfo.Size)
			}
			return "", fmt.Errorf("download truncated: received %d of %d remaining bytes", written, remaining)
		}
		break
	}

	if err := file.Sync(); err != nil {
		return "", fmt.Errorf("failed to sync cache file: %w", err)
	}
	if err := file.Close(); err != nil {
		return "", fmt.Errorf("failed to close partial cache file: %w", err)
	}
	fileClosed = true
	keepForResume = false
	if err := replaceFile(partialPath, cachePath); err != nil {
		return "", fmt.Errorf("failed to commit downloaded file to cache: %w", err)
	}
	if err := os.Remove(resumePath); err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("download committed but stale resume file could not be removed: %w", err)
	}
	return cachePath, nil
}

func seedPartialDownload(destination *os.File, resumePath, cachePath string, expectedSize int64) (int64, error) {
	for _, sourcePath := range []string{resumePath, cachePath} {
		stat, err := os.Lstat(sourcePath)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return 0, fmt.Errorf("failed to inspect partial download: %w", err)
		}
		if !stat.Mode().IsRegular() || stat.Size() <= 0 || stat.Size() >= expectedSize {
			continue
		}
		source, err := os.Open(sourcePath)
		if err != nil {
			return 0, fmt.Errorf("failed to open partial download: %w", err)
		}
		written, copyErr := io.CopyN(destination, source, stat.Size())
		closeErr := source.Close()
		if copyErr != nil {
			return 0, fmt.Errorf("failed to copy partial download: %w", copyErr)
		}
		if closeErr != nil {
			return 0, fmt.Errorf("failed to close partial download: %w", closeErr)
		}
		return written, nil
	}
	return 0, nil
}

func resetPartialFile(file *os.File) error {
	if err := file.Truncate(0); err != nil {
		return fmt.Errorf("failed to reset partial download: %w", err)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek partial download: %w", err)
	}
	return nil
}

func replaceFile(sourcePath, targetPath string) error {
	if err := os.Rename(sourcePath, targetPath); err == nil {
		return nil
	}
	if err := os.Remove(targetPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.Rename(sourcePath, targetPath)
}

func acquireCacheLock(ctx context.Context, lockPath string, maxWait time.Duration) (func() error, error) {
	waitCtx, cancel := context.WithTimeout(ctx, maxWait)
	defer cancel()
	for {
		file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err == nil {
			if closeErr := file.Close(); closeErr != nil {
				_ = os.Remove(lockPath)
				return nil, fmt.Errorf("failed to close cache lock: %w", closeErr)
			}
			return func() error {
				if err := os.Remove(lockPath); err != nil && !os.IsNotExist(err) {
					return fmt.Errorf("failed to release cache lock: %w", err)
				}
				return nil
			}, nil
		}
		if !os.IsExist(err) {
			return nil, fmt.Errorf("failed to create cache lock: %w", err)
		}
		if stat, statErr := os.Stat(lockPath); statErr == nil && time.Since(stat.ModTime()) > 2*maxWait {
			if removeErr := os.Remove(lockPath); removeErr == nil || os.IsNotExist(removeErr) {
				continue
			}
		}
		if err := waitForRetry(waitCtx, 25*time.Millisecond); err != nil {
			return nil, fmt.Errorf("waiting for concurrent download: %w", err)
		}
	}
}

// verifyChecksum computes the SHA-256 of filePath and compares it to expectedSHA256.
// Returns an error on mismatch or I/O failure; nil when the checksum matches.
func (d *Downloader) verifyChecksum(filePath, expectedSHA256 string) error {
	d.logger.Verify("Verifying checksum...")

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer func() { _ = file.Close() }()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}

	actualSHA256 := fmt.Sprintf("%x", hasher.Sum(nil))
	if actualSHA256 != expectedSHA256 {
		return fmt.Errorf("checksum mismatch: expected %s, got %s",
			expectedSHA256, actualSHA256)
	}

	d.logger.Success("Checksum verified")
	return nil
}

func (d *Downloader) installArchive(archivePath, installDir, version string) (resultErr error) {
	parentDir := filepath.Dir(installDir)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create installation parent directory: %w", err)
	}
	stagingDir, err := os.MkdirTemp(parentDir, "."+filepath.Base(installDir)+".partial-")
	if err != nil {
		return fmt.Errorf("failed to create installation staging directory: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			if cleanupErr := os.RemoveAll(stagingDir); cleanupErr != nil {
				resultErr = errors.Join(resultErr, fmt.Errorf("failed to clean installation staging directory: %w", cleanupErr))
			}
		}
	}()

	if err := d.extractArchive(archivePath, stagingDir); err != nil {
		return err
	}
	if err := validateExtractedInstallation(stagingDir, version); err != nil {
		return err
	}
	if _, err := os.Lstat(installDir); err == nil {
		return fmt.Errorf("installation target already exists: %s", installDir)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to inspect installation target: %w", err)
	}
	if err := os.Rename(stagingDir, installDir); err != nil {
		return fmt.Errorf("failed to commit installation: %w", err)
	}
	committed = true
	return nil
}

func validateExtractedInstallation(installDir, version string) error {
	goExecutable := filepath.Join(installDir, "bin", "go")
	if runtime.GOOS == "windows" {
		goExecutable += ".exe"
	}
	info, err := os.Lstat(goExecutable)
	if err != nil {
		return fmt.Errorf("downloaded Go %s archive is incomplete: executable not found: %w", version, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("downloaded Go %s executable is not a regular file", version)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0111 == 0 {
		return fmt.Errorf("downloaded Go %s executable is not executable", version)
	}
	return nil
}

// extractArchive ensures installDir exists and extracts archivePath based on its extension (.tar.gz or .zip).
// Returns an error for unsupported formats or extraction failures.
func (d *Downloader) extractArchive(archivePath, installDir string) error {
	d.logger.Extract("Extracting archive...")

	if !strings.HasSuffix(archivePath, ".tar.gz") && !strings.HasSuffix(archivePath, ".zip") {
		return fmt.Errorf("unsupported archive format")
	}
	if err := os.MkdirAll(installDir, 0755); err != nil {
		return fmt.Errorf("failed to create install directory: %w", err)
	}

	if strings.HasSuffix(archivePath, ".tar.gz") {
		return d.extractTarGz(archivePath, installDir)
	}
	return d.extractZip(archivePath, installDir)
}

// validateArchivePath checks if an archive path is safe (no traversal) and returns the target path.
func validateArchivePath(path, installDir, originalName string) (string, error) {
	normalized := strings.ReplaceAll(path, "\\", "/")
	if normalized == "" || strings.ContainsRune(normalized, '\x00') || strings.HasPrefix(normalized, "/") {
		return "", fmt.Errorf("unsafe path in archive: %s", originalName)
	}
	if len(normalized) >= 2 && normalized[1] == ':' {
		return "", fmt.Errorf("unsafe volume path in archive: %s", originalName)
	}
	cleaned := pathpkg.Clean(normalized)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", fmt.Errorf("unsafe path in archive: %s", originalName)
	}
	root, err := filepath.Abs(installDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve extraction root: %w", err)
	}
	targetPath := filepath.Join(root, filepath.FromSlash(cleaned))
	rel, err := filepath.Rel(root, targetPath)
	if err != nil || rel == ".." || filepath.IsAbs(rel) || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("path traversal attempt detected in archive: %s", originalName)
	}
	return targetPath, nil
}

// extractTarGz extracts a .tar.gz archive into installDir with path safety checks and file permissions preserved.
// Returns an error on I/O issues or unsafe paths.
func (d *Downloader) extractTarGz(archivePath, installDir string) error {
	if err := os.MkdirAll(installDir, 0755); err != nil {
		return fmt.Errorf("failed to create extraction root: %w", err)
	}
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer func() { _ = file.Close() }()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer func() { _ = gzReader.Close() }()

	tarReader := tar.NewReader(gzReader)
	root, err := os.OpenRoot(installDir)
	if err != nil {
		return fmt.Errorf("failed to open extraction root: %w", err)
	}
	defer func() { _ = root.Close() }()
	entryCount := 0
	var totalSize int64

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
		entryCount++
		if entryCount > d.maxArchiveEntries {
			return fmt.Errorf("archive contains too many entries: limit is %d", d.maxArchiveEntries)
		}
		if header.Size < 0 || header.Size > d.maxFileSize {
			return fmt.Errorf("archive entry %s exceeds size limit", header.Name)
		}
		if header.Typeflag == tar.TypeReg || header.Typeflag == legacyTarTypeReg {
			totalSize += header.Size
			if totalSize > d.maxTotalSize {
				return fmt.Errorf("archive extracted size exceeds limit")
			}
		}

		targetPath, err := validateArchivePath(path, installDir, header.Name)
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(installDir, targetPath)
		if err != nil {
			return fmt.Errorf("failed to resolve archive entry path: %w", err)
		}
		if err := d.extractTarEntry(header, tarReader, root, relPath); err != nil {
			return err
		}
	}

	return nil
}

// extractTarEntry processes a single tar entry (directory, symlink, or regular file).
func (d *Downloader) extractTarEntry(header *tar.Header, tarReader *tar.Reader, root *os.Root, targetPath string) error {
	parentDir := filepath.Dir(targetPath)
	if parentDir != "." {
		if err := root.MkdirAll(parentDir, 0755); err != nil {
			return fmt.Errorf("failed to create safe parent directory: %w", err)
		}
	}
	switch header.Typeflag {
	case tar.TypeDir:
		mode := os.FileMode(header.Mode & 0755)
		if mode == 0 {
			mode = 0755
		}
		if err := root.MkdirAll(targetPath, mode); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", targetPath, err)
		}
	case tar.TypeReg, legacyTarTypeReg:
		mode := os.FileMode(header.Mode & 0755)
		if mode == 0 {
			mode = 0644
		}
		outFile, err := root.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %w", targetPath, err)
		}
		written, copyErr := io.Copy(outFile, io.LimitReader(tarReader, header.Size+1))
		closeErr := outFile.Close()
		if copyErr != nil {
			return fmt.Errorf("failed to write file %s: %w", targetPath, copyErr)
		}
		if closeErr != nil {
			return fmt.Errorf("failed to close file %s: %w", targetPath, closeErr)
		}
		if written != header.Size {
			return fmt.Errorf("archive entry %s size mismatch: wrote %d, expected %d", header.Name, written, header.Size)
		}
	case tar.TypeSymlink, tar.TypeLink:
		return fmt.Errorf("archive links are not allowed: %s", header.Name)
	default:
		return fmt.Errorf("unsupported tar entry type %d for %s", header.Typeflag, header.Name)
	}
	return nil
}

// extractZip extracts a .zip archive into installDir with path safety checks and directory creation as needed.
// Returns an error on I/O issues or unsafe paths.
func (d *Downloader) extractZip(archivePath, installDir string) error {
	if err := os.MkdirAll(installDir, 0755); err != nil {
		return fmt.Errorf("failed to create extraction root: %w", err)
	}
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open zip archive: %w", err)
	}
	defer func() { _ = reader.Close() }()
	root, err := os.OpenRoot(installDir)
	if err != nil {
		return fmt.Errorf("failed to open extraction root: %w", err)
	}
	defer func() { _ = root.Close() }()
	entryCount := 0
	var totalSize int64

	for _, file := range reader.File {
		entryCount++
		if entryCount > d.maxArchiveEntries {
			return fmt.Errorf("archive contains too many entries: limit is %d", d.maxArchiveEntries)
		}
		if file.UncompressedSize64 > math.MaxInt64 {
			return fmt.Errorf("archive entry %s exceeds supported size", file.Name)
		}
		fileSize := int64(file.UncompressedSize64)
		if d.maxFileSize < 0 || fileSize > d.maxFileSize {
			return fmt.Errorf("archive entry %s exceeds size limit", file.Name)
		}
		if d.maxTotalSize < 0 || totalSize > d.maxTotalSize || fileSize > d.maxTotalSize-totalSize {
			return fmt.Errorf("archive extracted size exceeds limit")
		}
		totalSize += fileSize
		if file.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("archive links are not allowed: %s", file.Name)
		}
		path := file.Name
		if strings.HasPrefix(path, "go/") || strings.HasPrefix(path, "go\\") {
			path = path[3:]
		}

		if path == "" {
			continue
		}

		targetPath, err := validateArchivePath(path, installDir, file.Name)
		if err != nil {
			return err
		}

		if file.FileInfo().IsDir() {
			relPath, relErr := filepath.Rel(installDir, targetPath)
			if relErr != nil {
				return fmt.Errorf("failed to resolve directory path: %w", relErr)
			}
			if err := root.MkdirAll(relPath, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", targetPath, err)
			}
			continue
		}

		relPath, relErr := filepath.Rel(installDir, targetPath)
		if relErr != nil {
			return fmt.Errorf("failed to resolve file path: %w", relErr)
		}
		if err := d.extractZipFile(file, root, relPath, fileSize); err != nil {
			return err
		}
	}

	return nil
}

// extractZipFile extracts a single file from a zip archive.
func (d *Downloader) extractZipFile(file *zip.File, root *os.Root, targetPath string, expectedSize int64) error {
	parentDir := filepath.Dir(targetPath)
	if parentDir != "." {
		if err := root.MkdirAll(parentDir, 0755); err != nil {
			return fmt.Errorf("failed to create safe parent directory: %w", err)
		}
	}

	srcFile, err := file.Open()
	if err != nil {
		return fmt.Errorf("failed to open file in archive: %w", err)
	}
	defer func() { _ = srcFile.Close() }()

	mode := file.Mode().Perm() & 0755
	if mode == 0 {
		mode = 0644
	}
	dstFile, err := root.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", targetPath, err)
	}

	written, copyErr := io.Copy(dstFile, io.LimitReader(srcFile, expectedSize+1))
	closeErr := dstFile.Close()
	if copyErr != nil {
		return fmt.Errorf("failed to write file %s: %w", targetPath, copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("failed to close file %s: %w", targetPath, closeErr)
	}
	if written != expectedSize {
		return fmt.Errorf("archive entry %s size mismatch: wrote %d, expected %d", file.Name, written, expectedSize)
	}

	return nil
}
