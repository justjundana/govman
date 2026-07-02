package downloader

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	_config "github.com/justjundana/govman/internal/config"
	_golang "github.com/justjundana/govman/internal/golang"
	_logger "github.com/justjundana/govman/internal/logger"
)

// createTestConfig creates a test configuration with temporary directories
func createTestConfig(t *testing.T) *_config.Config {
	tempDir := t.TempDir()

	config := &_config.Config{
		InstallDir: filepath.Join(tempDir, "versions"),
		CacheDir:   filepath.Join(tempDir, "cache"),
		Download: _config.DownloadConfig{
			Timeout:    30 * time.Second,
			RetryCount: 3,
			RetryDelay: 1 * time.Second,
		},
		GoReleases: _config.GoReleasesConfig{
			APIURL:      "https://api.github.com/repos/golang/go/releases",
			CacheExpiry: time.Minute,
		},
	}

	// Create directories
	os.MkdirAll(config.InstallDir, 0755)
	os.MkdirAll(config.CacheDir, 0755)

	return config
}

// createTestDownloader creates a downloader instance for testing
func createTestDownloader(t *testing.T, config *_config.Config) *Downloader {
	return New(config)
}

// mockFileInfo creates a mock File struct for testing
func mockFileInfo() *_golang.File {
	return &_golang.File{
		Filename: "go1.20.0.darwin-amd64.tar.gz",
		OS:       "darwin",
		Arch:     "amd64",
		Version:  "go1.20.0",
		Sha256:   "1234567890abcdef",
		Size:     1024,
		Kind:     "archive",
	}
}

// TestDownloader_New tests the New constructor with various configs
func TestDownloader_New(t *testing.T) {
	testCases := []struct {
		name        string
		config      *_config.Config
		expectError bool
	}{
		{
			name: "Valid config",
			config: func() *_config.Config {
				config := createTestConfig(t)
				config.Download.Timeout = 60 * time.Second
				return config
			}(),
			expectError: false,
		},
		{
			name: "Config with zero timeout",
			config: func() *_config.Config {
				config := createTestConfig(t)
				config.Download.Timeout = 0
				return config
			}(),
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			downloader := New(tc.config)

			if tc.expectError {
				t.Skip("New() constructor doesn't fail in current implementation")
			}

			if downloader.config != tc.config {
				t.Error("Downloader config not set correctly")
			}
			if downloader.client == nil {
				t.Error("Downloader HTTP client not initialized")
			}
			if downloader.client.Timeout != tc.config.Download.Timeout {
				t.Errorf("Expected timeout %v, got %v", tc.config.Download.Timeout, downloader.client.Timeout)
			}
		})
	}
}

func TestDownloader_NewWithLogger(t *testing.T) {
	config := createTestConfig(t)
	injected := _logger.New()

	downloader := NewWithLogger(config, injected)
	if downloader.logger != injected {
		t.Fatal("NewWithLogger did not preserve the injected logger")
	}
	if fallback := NewWithLogger(config, nil); fallback.logger == nil {
		t.Fatal("NewWithLogger did not create a fallback logger")
	}
}

func TestRetryDelayVariants(t *testing.T) {
	now := time.Date(2026, time.July, 3, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name  string
		value string
		want  time.Duration
	}{
		{name: "empty", value: "", want: 0},
		{name: "seconds", value: "2", want: 2 * time.Second},
		{name: "negative", value: "-1", want: 0},
		{name: "seconds capped", value: "120", want: time.Minute},
		{name: "HTTP date", value: now.Add(5 * time.Second).Format(http.TimeFormat), want: 5 * time.Second},
		{name: "past HTTP date", value: now.Add(-time.Second).Format(http.TimeFormat), want: 0},
		{name: "invalid", value: "soon", want: 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := retryDelay(test.value, now); got != test.want {
				t.Fatalf("retryDelay(%q)=%v, want %v", test.value, got, test.want)
			}
		})
	}
	if err := waitForRetry(context.Background(), 0); err != nil {
		t.Fatalf("zero retry delay error=%v", err)
	}
	if err := waitForRetry(context.Background(), time.Nanosecond); err != nil {
		t.Fatalf("elapsed retry delay error=%v", err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := waitForRetry(cancelled, time.Second); err != context.Canceled {
		t.Fatalf("cancelled retry delay error=%v", err)
	}
}

func TestHandleResumeResponseBranches(t *testing.T) {
	config := createTestConfig(t)
	downloader := New(config)

	partial, err := os.CreateTemp(t.TempDir(), "partial-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = partial.Close() })
	if _, err := partial.WriteString("partial"); err != nil {
		t.Fatal(err)
	}
	if got, err := downloader.handleResumeResponse(partial, &http.Response{StatusCode: http.StatusOK}, 7, 10); err != nil || got != 0 {
		t.Fatalf("200 resume reset got=%d err=%v", got, err)
	}
	if info, err := partial.Stat(); err != nil || info.Size() != 0 {
		t.Fatalf("partial was not truncated: info=%v err=%v", info, err)
	}

	response := func(contentRange string, contentLength int64) *http.Response {
		return &http.Response{
			StatusCode:    http.StatusPartialContent,
			ContentLength: contentLength,
			Header:        http.Header{"Content-Range": []string{contentRange}},
		}
	}
	if got, err := downloader.handleResumeResponse(partial, response("bytes 5-9/10", 5), 5, 10); err != nil || got != 5 {
		t.Fatalf("valid resume got=%d err=%v", got, err)
	}
	for _, test := range []struct {
		name   string
		header string
		length int64
		start  int64
		total  int64
	}{
		{name: "missing header", header: "", length: 5, start: 5, total: 10},
		{name: "wrong start", header: "bytes 4-9/10", length: 6, start: 5, total: 10},
		{name: "wrong total", header: "bytes 5-10/11", length: 6, start: 5, total: 10},
		{name: "wrong end", header: "bytes 5-8/10", length: 4, start: 5, total: 10},
		{name: "wrong length", header: "bytes 5-9/10", length: 4, start: 5, total: 10},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := downloader.handleResumeResponse(partial, response(test.header, test.length), test.start, test.total); err == nil {
				t.Fatal("invalid resume response was accepted")
			}
		})
	}
}

func TestPartialFileResetAndReplacementErrors(t *testing.T) {
	directory := t.TempDir()
	partial, err := os.CreateTemp(directory, "partial-*")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := partial.WriteString("data"); err != nil {
		t.Fatal(err)
	}
	if err := resetPartialFile(partial); err != nil {
		t.Fatal(err)
	}
	if offset, err := partial.Seek(0, io.SeekCurrent); err != nil || offset != 0 {
		t.Fatalf("reset offset=%d err=%v", offset, err)
	}
	if err := partial.Close(); err != nil {
		t.Fatal(err)
	}
	if err := resetPartialFile(partial); err == nil {
		t.Fatal("resetPartialFile accepted a closed file")
	}

	targetDirectory := filepath.Join(directory, "target-directory")
	if err := os.Mkdir(targetDirectory, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(targetDirectory, "child"), []byte("preserve"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := replaceFile(filepath.Join(directory, "missing"), targetDirectory); err == nil {
		t.Fatal("replaceFile accepted a missing source and non-empty target directory")
	}

	targetFile := filepath.Join(directory, "target-file")
	if err := os.WriteFile(targetFile, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := replaceFile(filepath.Join(directory, "still-missing"), targetFile); err == nil {
		t.Fatal("replaceFile accepted a missing source")
	}
	if _, err := os.Stat(targetFile); !os.IsNotExist(err) {
		t.Fatalf("fallback target removal did not occur: %v", err)
	}
}

// TestDownloader_downloadFile_Cached tests cached file handling
func TestDownloader_downloadFile_Cached(t *testing.T) {
	config := createTestConfig(t)
	downloader := createTestDownloader(t, config)

	testContent := "cached file content"
	cachePath := filepath.Join(config.CacheDir, "cached-file.tar.gz")
	err := os.WriteFile(cachePath, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create cached file: %v", err)
	}
	defer os.Remove(cachePath)

	fileInfo := mockFileInfo()
	fileInfo.Filename = "cached-file.tar.gz"
	fileInfo.Size = int64(len(testContent))

	resultPath, err := downloader.downloadFile("http://example.com/cached-file.tar.gz", fileInfo)
	if err != nil {
		t.Fatalf("downloadFile with cached file failed: %v", err)
	}

	if resultPath != cachePath {
		t.Errorf("Expected cached path %s, got %s", cachePath, resultPath)
	}
}

// TestDownloader_downloadFile_Timeout tests timeout handling
func TestDownloader_downloadFile_Timeout(t *testing.T) {
	config := createTestConfig(t)
	config.Download.Timeout = 1 * time.Millisecond
	downloader := createTestDownloader(t, config)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		w.Write([]byte("delayed response"))
	}))
	defer server.Close()

	fileInfo := mockFileInfo()
	fileInfo.Size = 17

	_, err := downloader.downloadFile(server.URL+"/"+fileInfo.Filename, fileInfo)
	if err == nil {
		t.Error("Expected timeout error but got none")
	}
}

// TestDownloader_verifyChecksum_EmptyFile tests checksum verification for empty files
func TestDownloader_verifyChecksum_EmptyFile(t *testing.T) {
	config := createTestConfig(t)
	downloader := createTestDownloader(t, config)

	testFile := filepath.Join(config.CacheDir, "empty.txt")
	err := os.WriteFile(testFile, []byte(""), 0644)
	if err != nil {
		t.Fatalf("Failed to create empty test file: %v", err)
	}
	defer os.Remove(testFile)

	emptySHA256 := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

	err = downloader.verifyChecksum(testFile, emptySHA256)
	if err != nil {
		t.Errorf("Empty file checksum verification failed: %v", err)
	}
}

// TestNew tests the New function
func TestNew(t *testing.T) {
	testCases := []struct {
		name  string
		setup func() *_config.Config
		check func(*testing.T, *Downloader)
	}{
		{
			name: "Valid configuration",
			setup: func() *_config.Config {
				return createTestConfig(t)
			},
			check: func(t *testing.T, d *Downloader) {
				if d == nil {
					t.Fatal("New() returned nil")
				}
				if d.client == nil {
					t.Error("Downloader HTTP client not initialized")
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := tc.setup()
			downloader := New(config)
			tc.check(t, downloader)
		})
	}
}

// TestDownloader_Download tests the Download method with a mock server
func TestDownloader_Download(t *testing.T) {
	testCases := []struct {
		name          string
		version       string
		mockResponse  string
		expectedError string
		setupDownload func(t *testing.T, config *_config.Config) (string, func())
	}{
		{
			name:          "Download with valid file info",
			version:       "1.20.0",
			mockResponse:  `[{"version":"go1.20.0","stable":true,"files":[{"filename":"go1.20.0.darwin-arm64.tar.gz","os":"darwin","arch":"arm64","version":"go1.20.0","sha256":"1234567890abcdef","size":1024,"kind":"archive"}]}]`,
			expectedError: "failed to download",
		},
		{
			name:          "Download with no matching files",
			version:       "1.19.0",
			mockResponse:  `[{"version":"go1.20.0","stable":true,"files":[{"filename":"go1.20.0.darwin-amd64.tar.gz","os":"darwin","arch":"amd64","version":"go1.20.0","sha256":"1234567890abcdef","size":1024,"kind":"archive"}]}]`,
			expectedError: "no file info available",
		},
		{
			name:          "Successful download",
			version:       "1.21.0",
			mockResponse:  "",
			expectedError: "",
			setupDownload: func(t *testing.T, config *_config.Config) (string, func()) {
				// Clear the golang package cache before setting up
				_golang.ClearReleasesCache()

				// Create a valid tar.gz file
				var buf bytes.Buffer
				gzWriter := gzip.NewWriter(&buf)
				tarWriter := tar.NewWriter(gzWriter)

				content := "test file content"
				header := &tar.Header{
					Name: "test.txt",
					Size: int64(len(content)),
					Mode: 0644,
				}
				tarWriter.WriteHeader(header)
				tarWriter.Write([]byte(content))
				for _, executable := range []string{"bin/go", "bin/go.exe"} {
					executableContent := "test go executable"
					executableHeader := &tar.Header{
						Name: executable,
						Size: int64(len(executableContent)),
						Mode: 0755,
					}
					tarWriter.WriteHeader(executableHeader)
					tarWriter.Write([]byte(executableContent))
				}
				tarWriter.Close()
				gzWriter.Close()

				archiveData := buf.Bytes()
				expectedSHA256 := fmt.Sprintf("%x", sha256.Sum256(archiveData))

				// Create API server first
				apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.Write([]byte(fmt.Sprintf(`[{"version":"go1.21.0","stable":true,"files":[{"filename":"go1.21.0.darwin-arm64.tar.gz","os":"darwin","arch":"arm64","version":"go1.21.0","sha256":"%s","size":%d,"kind":"archive"}]}]`, expectedSHA256, len(archiveData))))
				}))

				// Update config to use mock API server BEFORE creating download server
				config.GoReleases.APIURL = apiServer.URL

				// Create mock download server
				downloadServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Write(archiveData)
				}))

				cleanup := func() {
					apiServer.Close()
					downloadServer.Close()
					_golang.ClearReleasesCache()
				}

				return downloadServer.URL + "/go1.21.0.darwin-arm64.tar.gz", cleanup
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := createTestConfig(t)
			downloader := createTestDownloader(t, config)

			var cleanup func()
			downloadURL := "http://example.com/test.tar.gz"

			if tc.setupDownload != nil {
				downloadURL, cleanup = tc.setupDownload(t, config)
				defer cleanup()
			} else {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.Write([]byte(tc.mockResponse))
				}))
				cleanup = func() { server.Close() }
				defer cleanup()

				config.GoReleases.APIURL = server.URL
			}

			installDir := filepath.Join(config.InstallDir, "test")
			err := downloader.Download(downloadURL, installDir, tc.version)

			if tc.expectedError != "" {
				if err == nil {
					t.Error("Expected error but got none")
				} else if !strings.Contains(err.Error(), tc.expectedError) {
					t.Errorf("Expected error containing %q, got: %v", tc.expectedError, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				if tc.name == "Successful download" {
					extractedFile := filepath.Join(installDir, "test.txt")
					if _, err := os.Stat(extractedFile); os.IsNotExist(err) {
						t.Error("Extracted file does not exist")
					} else {
						content, err := os.ReadFile(extractedFile)
						if err != nil {
							t.Fatalf("Failed to read extracted file: %v", err)
						}
						if string(content) != "test file content" {
							t.Errorf("Expected extracted content %q, got %q", "test file content", string(content))
						}
					}
				}
			}
		})
	}
}

// TestDownloader_downloadFile tests the downloadFile method
func TestDownloader_downloadFile(t *testing.T) {
	testCases := []struct {
		name          string
		fileContent   string
		statusCode    int
		expectError   bool
		errorContains string
	}{
		{
			name:          "Successful download",
			fileContent:   "test file content",
			statusCode:    200,
			expectError:   false,
			errorContains: "",
		},
		{
			name:          "Server error",
			fileContent:   "",
			statusCode:    500,
			expectError:   true,
			errorContains: "download failed with status",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := createTestConfig(t)
			downloader := createTestDownloader(t, config)

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tc.statusCode != 200 {
					w.WriteHeader(tc.statusCode)
					return
				}
				w.Write([]byte(tc.fileContent))
			}))
			defer server.Close()

			fileInfo := mockFileInfo()
			fileInfo.Size = int64(len(tc.fileContent))
			if fileInfo.Size == 0 {
				fileInfo.Size = 1
			}

			cachePath, err := downloader.downloadFile(server.URL+"/"+fileInfo.Filename, fileInfo)

			if tc.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if !strings.Contains(err.Error(), tc.errorContains) {
					t.Errorf("Expected error containing %q, got: %v", tc.errorContains, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("downloadFile failed: %v", err)
			}

			if _, err := os.Stat(cachePath); os.IsNotExist(err) {
				t.Error("Downloaded file does not exist")
			}

			content, err := os.ReadFile(cachePath)
			if err != nil {
				t.Fatalf("Failed to read downloaded file: %v", err)
			}
			if string(content) != tc.fileContent {
				t.Errorf("Expected content %q, got %q", tc.fileContent, string(content))
			}

			os.Remove(cachePath)
		})
	}
}

// TestDownloader_downloadFile_Resume tests the resume functionality
func TestDownloader_downloadFile_Resume(t *testing.T) {
	testCases := []struct {
		name        string
		testContent string
		partialSize int
		expectError bool
	}{
		{
			name:        "Resume partial download",
			testContent: "test file content for resume",
			partialSize: 10,
			expectError: false,
		},
		{
			name:        "Resume with complete file",
			testContent: "complete file content",
			partialSize: 21,
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := createTestConfig(t)
			downloader := createTestDownloader(t, config)

			partialContent := tc.testContent[:tc.partialSize]

			cachePath := filepath.Join(config.CacheDir, "test-resume.txt")
			err := os.WriteFile(cachePath, []byte(partialContent), 0644)
			if err != nil {
				t.Fatalf("Failed to create partial file: %v", err)
			}

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Range") == fmt.Sprintf("bytes=%d-", tc.partialSize) {
					w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", tc.partialSize, len(tc.testContent)-1, len(tc.testContent)))
					w.WriteHeader(http.StatusPartialContent)
					w.Write([]byte(tc.testContent[tc.partialSize:]))
				} else {
					w.Write([]byte(tc.testContent))
				}
			}))
			defer server.Close()

			fileInfo := mockFileInfo()
			fileInfo.Size = int64(len(tc.testContent))
			fileInfo.Filename = "test-resume.txt"

			downloadedPath, err := downloader.downloadFile(server.URL+"/"+fileInfo.Filename, fileInfo)

			if tc.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("downloadFile resume failed: %v", err)
			}

			content, err := os.ReadFile(downloadedPath)
			if err != nil {
				t.Fatalf("Failed to read resumed file: %v", err)
			}
			if string(content) != tc.testContent {
				t.Errorf("Expected resumed content %q, got %q", tc.testContent, string(content))
			}

			os.Remove(downloadedPath)
		})
	}
}

// TestDownloader_verifyChecksum tests checksum verification
func TestDownloader_verifyChecksum(t *testing.T) {
	testCases := []struct {
		name           string
		fileContent    string
		expectedSHA256 string
		expectError    bool
	}{
		{
			name:           "Valid checksum",
			fileContent:    "test content for checksum",
			expectedSHA256: "",
			expectError:    false,
		},
		{
			name:           "Invalid checksum",
			fileContent:    "test content for checksum",
			expectedSHA256: "invalid-checksum",
			expectError:    true,
		},
		{
			name:           "Non-existent file",
			fileContent:    "",
			expectedSHA256: "1234567890abcdef",
			expectError:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := createTestConfig(t)
			downloader := createTestDownloader(t, config)

			var testFile string
			if tc.name != "Non-existent file" {
				if tc.expectedSHA256 == "" {
					tc.expectedSHA256 = fmt.Sprintf("%x", sha256.Sum256([]byte(tc.fileContent)))
				}

				testFile = filepath.Join(config.CacheDir, "test-checksum.txt")
				err := os.WriteFile(testFile, []byte(tc.fileContent), 0644)
				if err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
				defer os.Remove(testFile)
			} else {
				testFile = filepath.Join(config.CacheDir, "non-existent.txt")
			}

			err := downloader.verifyChecksum(testFile, tc.expectedSHA256)

			if tc.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Valid checksum verification failed: %v", err)
				}
			}
		})
	}
}

// TestDownloader_extractArchive tests archive extraction
func TestDownloader_extractArchive(t *testing.T) {
	testCases := []struct {
		name          string
		archiveName   string
		expectError   bool
		errorContains string
	}{
		{
			name:          "Unsupported format",
			archiveName:   "test.unsupported",
			expectError:   true,
			errorContains: "unsupported archive format",
		},
		{
			name:          "Tar.gz format",
			archiveName:   "test.tar.gz",
			expectError:   false,
			errorContains: "",
		},
		{
			name:          "Zip format",
			archiveName:   "test.zip",
			expectError:   false,
			errorContains: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := createTestConfig(t)
			downloader := createTestDownloader(t, config)

			installDir := filepath.Join(config.InstallDir, "test-extract")
			archiveFile := filepath.Join(config.CacheDir, tc.archiveName)

			err := os.WriteFile(archiveFile, []byte("test"), 0644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}
			defer os.Remove(archiveFile)

			err = downloader.extractArchive(archiveFile, installDir)

			if tc.expectError {
				if err == nil || !strings.Contains(err.Error(), tc.errorContains) {
					t.Errorf("Expected error containing %q, got: %v", tc.errorContains, err)
				}
			} else {
				if err == nil {
					t.Error("Expected error for invalid archive file")
				}
			}
		})
	}
}

// createTestTarGz creates a tar.gz archive with the given file entry.
func createTestTarGz(t *testing.T, fileName, fileContent string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	tarWriter := tar.NewWriter(gzWriter)

	if fileName != "" {
		header := &tar.Header{
			Name: fileName,
			Size: int64(len(fileContent)),
			Mode: 0644,
		}
		if strings.HasSuffix(fileName, "/") {
			header.Typeflag = tar.TypeDir
			header.Mode = 0755
			header.Size = 0
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			t.Fatalf("Failed to write tar header: %v", err)
		}
		if !strings.HasSuffix(fileName, "/") {
			if _, err := tarWriter.Write([]byte(fileContent)); err != nil {
				t.Fatalf("Failed to write tar content: %v", err)
			}
		}
	}

	tarWriter.Close()
	gzWriter.Close()
	return buf.Bytes()
}

// verifyExtractedFile checks that the extracted file exists and has the expected content.
func verifyExtractedFile(t *testing.T, installDir, fileName, expectedContent string) {
	t.Helper()
	extractedPath := filepath.Join(installDir, fileName)
	if strings.HasSuffix(fileName, "/") {
		if _, err := os.Stat(extractedPath); os.IsNotExist(err) {
			t.Error("Extracted directory does not exist")
		}
		return
	}

	if _, err := os.Stat(extractedPath); os.IsNotExist(err) {
		t.Error("Extracted file does not exist")
	}

	content, err := os.ReadFile(extractedPath)
	if err != nil {
		t.Fatalf("Failed to read extracted file: %v", err)
	}
	if string(content) != expectedContent {
		t.Errorf("Expected extracted content %q, got %q", expectedContent, string(content))
	}
}

// TestDownloader_extractTarGz tests tar.gz extraction
func TestDownloader_extractTarGz(t *testing.T) {
	testCases := []struct {
		name        string
		fileName    string
		fileContent string
		expectError bool
	}{
		{
			name:        "Valid tar.gz extraction",
			fileName:    "test.txt",
			fileContent: "test file content",
			expectError: false,
		},
		{
			name:        "Empty file name",
			fileName:    "",
			fileContent: "content",
			expectError: false,
		},
		{
			name:        "Directory entry",
			fileName:    "testdir/",
			fileContent: "",
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := createTestConfig(t)
			downloader := createTestDownloader(t, config)

			installDir := filepath.Join(config.InstallDir, "test-tar")

			tarData := createTestTarGz(t, tc.fileName, tc.fileContent)

			tarFile := filepath.Join(config.CacheDir, "test.tar.gz")
			if err := os.WriteFile(tarFile, tarData, 0644); err != nil {
				t.Fatalf("Failed to write tar.gz file: %v", err)
			}
			defer os.Remove(tarFile)

			err := downloader.extractTarGz(tarFile, installDir)

			if tc.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("extractTarGz failed: %v", err)
			}

			if tc.fileName != "" {
				verifyExtractedFile(t, installDir, tc.fileName, tc.fileContent)
			}

			os.RemoveAll(installDir)
		})
	}
}

// TestDownloader_extractZip tests zip extraction
func TestDownloader_extractZip(t *testing.T) {
	testCases := []struct {
		name        string
		fileName    string
		fileContent string
		expectError bool
	}{
		{
			name:        "Valid zip extraction",
			fileName:    "test.txt",
			fileContent: "test zip content",
			expectError: false,
		},
		{
			name:        "Go-prefixed file name",
			fileName:    "go/test.txt",
			fileContent: "content",
			expectError: false,
		},
		{
			name:        "Nested directory structure",
			fileName:    "go/bin/go",
			fileContent: "binary content",
			expectError: false,
		},
		{
			name:        "Empty file",
			fileName:    "empty.txt",
			fileContent: "",
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := createTestConfig(t)
			downloader := createTestDownloader(t, config)

			installDir := filepath.Join(config.InstallDir, "test-zip")

			var buf bytes.Buffer
			zipWriter := zip.NewWriter(&buf)

			fileWriter, err := zipWriter.Create(tc.fileName)
			if err != nil {
				t.Fatalf("Failed to create zip file: %v", err)
			}
			_, err = fileWriter.Write([]byte(tc.fileContent))
			if err != nil {
				t.Fatalf("Failed to write zip content: %v", err)
			}

			zipWriter.Close()

			zipFile := filepath.Join(config.CacheDir, "test.zip")
			err = os.WriteFile(zipFile, buf.Bytes(), 0644)
			if err != nil {
				t.Fatalf("Failed to write zip file: %v", err)
			}
			defer os.Remove(zipFile)

			err = downloader.extractZip(zipFile, installDir)

			if tc.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("extractZip failed: %v", err)
			}

			expectedName := tc.fileName
			if strings.HasPrefix(tc.fileName, "go/") || strings.HasPrefix(tc.fileName, "go\\") {
				expectedName = tc.fileName[3:]
			}

			extractedFile := filepath.Join(installDir, expectedName)
			if _, err := os.Stat(extractedFile); os.IsNotExist(err) {
				t.Error("Extracted file does not exist")
			}

			content, err := os.ReadFile(extractedFile)
			if err != nil {
				t.Fatalf("Failed to read extracted file: %v", err)
			}
			if string(content) != tc.fileContent {
				t.Errorf("Expected extracted content %q, got %q", tc.fileContent, string(content))
			}

			os.RemoveAll(installDir)
		})
	}
}

// TestDownloader_extractTarGz_PathTraversal tests path traversal protection in tar extraction
func TestDownloader_extractTarGz_PathTraversal(t *testing.T) {
	testCases := []struct {
		name     string
		fileName string
		expected string
	}{
		{
			name:     "Path traversal with ..",
			fileName: "../../../etc/passwd",
			expected: "unsafe path in archive",
		},
		{
			name:     "Absolute path",
			fileName: "/etc/passwd",
			expected: "unsafe path in archive",
		},
		{
			name:     "Path with backslash traversal",
			fileName: "..\\..\\etc\\passwd",
			expected: "unsafe path in archive",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := createTestConfig(t)
			downloader := createTestDownloader(t, config)

			installDir := filepath.Join(config.InstallDir, "test-path-traversal")

			var buf bytes.Buffer
			gzWriter := gzip.NewWriter(&buf)
			tarWriter := tar.NewWriter(gzWriter)

			header := &tar.Header{
				Name: tc.fileName,
				Size: 4,
				Mode: 0644,
			}
			err := tarWriter.WriteHeader(header)
			if err != nil {
				t.Fatalf("Failed to write tar header: %v", err)
			}
			_, err = tarWriter.Write([]byte("test"))
			if err != nil {
				t.Fatalf("Failed to write tar content: %v", err)
			}

			tarWriter.Close()
			gzWriter.Close()

			tarFile := filepath.Join(config.CacheDir, "malicious.tar.gz")
			err = os.WriteFile(tarFile, buf.Bytes(), 0644)
			if err != nil {
				t.Fatalf("Failed to write malicious tar.gz file: %v", err)
			}
			defer os.Remove(tarFile)

			err = downloader.extractTarGz(tarFile, installDir)
			if err == nil || !strings.Contains(err.Error(), tc.expected) {
				t.Errorf("Expected error containing %q, got: %v", tc.expected, err)
			}
		})
	}
}

// TestDownloader_extractZip_PathTraversal tests path traversal protection in zip extraction
func TestDownloader_extractZip_PathTraversal(t *testing.T) {
	testCases := []struct {
		name     string
		fileName string
		expected string
	}{
		{
			name:     "Path traversal with ..",
			fileName: "../../../etc/passwd",
			expected: "unsafe path in archive",
		},
		{
			name:     "Absolute path",
			fileName: "/etc/passwd",
			expected: "unsafe path in archive",
		},
		{
			name:     "Path with backslash traversal",
			fileName: "..\\..\\etc\\passwd",
			expected: "unsafe path in archive",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := createTestConfig(t)
			downloader := createTestDownloader(t, config)

			installDir := filepath.Join(config.InstallDir, "test-zip-traversal")

			var buf bytes.Buffer
			zipWriter := zip.NewWriter(&buf)

			_, err := zipWriter.Create(tc.fileName)
			if err != nil {
				t.Fatalf("Failed to create zip file: %v", err)
			}

			zipWriter.Close()

			zipFile := filepath.Join(config.CacheDir, "malicious.zip")
			err = os.WriteFile(zipFile, buf.Bytes(), 0644)
			if err != nil {
				t.Fatalf("Failed to write malicious zip file: %v", err)
			}
			defer os.Remove(zipFile)

			err = downloader.extractZip(zipFile, installDir)
			if err == nil || !strings.Contains(err.Error(), tc.expected) {
				t.Errorf("Expected error containing %q, got: %v", tc.expected, err)
			}
		})
	}
}

// TestDownloader_Download_ErrorPaths tests error handling in the Download method
func TestDownloader_Download_ErrorPaths(t *testing.T) {
	extension := ".tar.gz"
	if runtime.GOOS == "windows" {
		extension = ".zip"
	}
	filename := fmt.Sprintf("go1.20.0.%s-%s%s", runtime.GOOS, runtime.GOARCH, extension)
	metadata := fmt.Sprintf(`[{"version":"go1.20.0","stable":true,"files":[{"filename":%q,"os":%q,"arch":%q,"version":"go1.20.0","sha256":"1234567890abcdef","size":1024,"kind":"archive"}]}]`, filename, runtime.GOOS, runtime.GOARCH)

	t.Run("Invalid version - no file info", func(t *testing.T) {
		config := createTestConfig(t)
		downloader := createTestDownloader(t, config)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(metadata))
		}))
		defer server.Close()
		config.GoReleases.APIURL = server.URL

		err := downloader.Download(server.URL+"/"+filename, filepath.Join(config.InstallDir, "invalid-version"), "invalid-version")
		if err == nil || !strings.Contains(err.Error(), "no file info available") {
			t.Fatalf("expected no file info error, got: %v", err)
		}
	})

	t.Run("Network error during download", func(t *testing.T) {
		config := createTestConfig(t)
		config.Download.RetryCount = 1
		downloader := createTestDownloader(t, config)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/metadata" {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(metadata))
				return
			}
			http.Error(w, "download unavailable", http.StatusInternalServerError)
		}))
		defer server.Close()
		config.GoReleases.APIURL = server.URL + "/metadata"

		err := downloader.Download(server.URL+"/"+filename, filepath.Join(config.InstallDir, "network-error"), "1.20.0")
		if err == nil || !strings.Contains(err.Error(), "failed to download") {
			t.Fatalf("expected download error, got: %v", err)
		}
	})
}

// TestDownloader_extractTarGz_ErrorHandling tests error handling in tar.gz extraction
func TestDownloader_extractTarGz_ErrorHandling(t *testing.T) {
	testCases := []struct {
		name          string
		setupArchive  func() ([]byte, error)
		expectError   bool
		errorContains string
	}{
		{
			name: "Invalid gzip data",
			setupArchive: func() ([]byte, error) {
				var buf bytes.Buffer
				tarWriter := tar.NewWriter(&buf)
				header := &tar.Header{
					Name: "test.txt",
					Size: 4,
					Mode: 0644,
				}
				tarWriter.WriteHeader(header)
				tarWriter.Write([]byte("test"))
				tarWriter.Close()
				return buf.Bytes(), nil
			},
			expectError:   true,
			errorContains: "failed to create gzip reader",
		},
		{
			name: "Corrupted tar data",
			setupArchive: func() ([]byte, error) {
				var buf bytes.Buffer
				gzWriter := gzip.NewWriter(&buf)
				gzWriter.Write([]byte("not a tar file"))
				gzWriter.Close()
				return buf.Bytes(), nil
			},
			expectError:   true,
			errorContains: "failed to read tar header",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := createTestConfig(t)
			downloader := createTestDownloader(t, config)

			installDir := filepath.Join(config.InstallDir, "test-tar-error")

			archiveData, err := tc.setupArchive()
			if err != nil {
				t.Fatalf("Failed to setup archive: %v", err)
			}

			tarFile := filepath.Join(config.CacheDir, "corrupted.tar.gz")
			err = os.WriteFile(tarFile, archiveData, 0644)
			if err != nil {
				t.Fatalf("Failed to write corrupted tar.gz file: %v", err)
			}
			defer os.Remove(tarFile)

			err = downloader.extractTarGz(tarFile, installDir)

			if tc.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if !strings.Contains(err.Error(), tc.errorContains) {
					t.Errorf("Expected error containing %q, got: %v", tc.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

// TestDownloader_extractZip_ErrorHandling tests error handling in zip extraction
func TestDownloader_extractZip_ErrorHandling(t *testing.T) {
	testCases := []struct {
		name          string
		setupArchive  func() ([]byte, error)
		expectError   bool
		errorContains string
	}{
		{
			name: "Invalid zip data",
			setupArchive: func() ([]byte, error) {
				return []byte("not a zip file"), nil
			},
			expectError:   true,
			errorContains: "failed to open zip archive",
		},
		{
			name: "Corrupted zip data",
			setupArchive: func() ([]byte, error) {
				var buf bytes.Buffer
				buf.WriteString("PK\x03\x04")
				buf.Write(make([]byte, 20))
				return buf.Bytes(), nil
			},
			expectError:   true,
			errorContains: "failed to open zip archive",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := createTestConfig(t)
			downloader := createTestDownloader(t, config)

			installDir := filepath.Join(config.InstallDir, "test-zip-error")

			archiveData, err := tc.setupArchive()
			if err != nil {
				t.Fatalf("Failed to setup archive: %v", err)
			}

			zipFile := filepath.Join(config.CacheDir, "corrupted.zip")
			err = os.WriteFile(zipFile, archiveData, 0644)
			if err != nil {
				t.Fatalf("Failed to write corrupted zip file: %v", err)
			}
			defer os.Remove(zipFile)

			err = downloader.extractZip(zipFile, installDir)

			if tc.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if !strings.Contains(err.Error(), tc.errorContains) {
					t.Errorf("Expected error containing %q, got: %v", tc.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

// TestDownloader_extractZip_Symlinks tests symlink handling in zip extraction
func TestDownloader_extractZip_Symlinks(t *testing.T) {
	config := createTestConfig(t)
	downloader := createTestDownloader(t, config)

	installDir := filepath.Join(config.InstallDir, "test-zip-symlinks")

	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	fileWriter, err := zipWriter.Create("target.txt")
	if err != nil {
		t.Fatalf("Failed to create zip file: %v", err)
	}
	_, err = fileWriter.Write([]byte("target content"))
	if err != nil {
		t.Fatalf("Failed to write zip content: %v", err)
	}

	linkWriter, err := zipWriter.Create("link.txt")
	if err != nil {
		t.Fatalf("Failed to create zip symlink: %v", err)
	}
	_, err = linkWriter.Write([]byte("link content"))
	if err != nil {
		t.Fatalf("Failed to write zip link content: %v", err)
	}

	zipWriter.Close()

	zipFile := filepath.Join(config.CacheDir, "symlink.zip")
	err = os.WriteFile(zipFile, buf.Bytes(), 0644)
	if err != nil {
		t.Fatalf("Failed to write zip file: %v", err)
	}
	defer os.Remove(zipFile)

	err = downloader.extractZip(zipFile, installDir)
	if err != nil {
		t.Fatalf("extractZip failed: %v", err)
	}

	targetFile := filepath.Join(installDir, "target.txt")
	if _, err := os.Stat(targetFile); os.IsNotExist(err) {
		t.Error("Target file does not exist")
	}

	linkFile := filepath.Join(installDir, "link.txt")
	if _, err := os.Stat(linkFile); os.IsNotExist(err) {
		t.Error("Link file does not exist")
	}

	os.RemoveAll(installDir)
}

// TestDownloader_extractZip_DirectoryCreation tests directory creation in zip extraction
func TestDownloader_extractZip_DirectoryCreation(t *testing.T) {
	testCases := []struct {
		name        string
		dirName     string
		expectError bool
	}{
		{
			name:        "Create simple directory",
			dirName:     "testdir/",
			expectError: false,
		},
		{
			name:        "Create nested directory",
			dirName:     "parent/child/",
			expectError: false,
		},
		{
			name:        "Create directory with go prefix",
			dirName:     "go/testdir/",
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := createTestConfig(t)
			downloader := createTestDownloader(t, config)

			installDir := filepath.Join(config.InstallDir, "test-zip-dirs")

			err := os.MkdirAll(installDir, 0755)
			if err != nil {
				t.Fatalf("Failed to create install directory: %v", err)
			}

			var buf bytes.Buffer
			zipWriter := zip.NewWriter(&buf)

			_, err = zipWriter.Create(tc.dirName)
			if err != nil {
				t.Fatalf("Failed to create zip directory: %v", err)
			}

			fileName := strings.TrimSuffix(tc.dirName, "/") + "/file.txt"
			fileWriter, err := zipWriter.Create(fileName)
			if err != nil {
				t.Fatalf("Failed to create zip file: %v", err)
			}
			_, err = fileWriter.Write([]byte("file content"))
			if err != nil {
				t.Fatalf("Failed to write zip content: %v", err)
			}

			zipWriter.Close()

			zipFile := filepath.Join(config.CacheDir, "dir.zip")
			err = os.WriteFile(zipFile, buf.Bytes(), 0644)
			if err != nil {
				t.Fatalf("Failed to write zip file: %v", err)
			}
			defer os.Remove(zipFile)

			err = downloader.extractZip(zipFile, installDir)

			if tc.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("extractZip failed: %v", err)
			}

			expectedDir := filepath.Join(installDir, strings.TrimPrefix(strings.TrimSuffix(tc.dirName, "/"), "go/"))
			if _, err := os.Stat(expectedDir); os.IsNotExist(err) {
				t.Errorf("Directory %s does not exist", expectedDir)
			}

			expectedFile := filepath.Join(installDir, strings.TrimPrefix(fileName, "go/"))
			if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
				t.Errorf("File %s does not exist", expectedFile)
			}

			os.RemoveAll(installDir)
		})
	}
}

// TestDownloader_verifyChecksum_WrongHash tests checksum verification with wrong hash
func TestDownloader_verifyChecksum_WrongHash(t *testing.T) {
	config := createTestConfig(t)
	downloader := createTestDownloader(t, config)

	testFile := filepath.Join(config.CacheDir, "wrong-hash.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile)

	err = downloader.verifyChecksum(testFile, "wrong-hash-value")
	if err == nil {
		t.Error("Expected checksum verification error but got none")
	}
	if !strings.Contains(err.Error(), "checksum mismatch") {
		t.Errorf("Expected checksum mismatch error, got: %v", err)
	}
}

// TestDownloader_GetFileInfoFailure tests file info retrieval failure
func TestDownloader_GetFileInfoFailure(t *testing.T) {
	_golang.ClearReleasesCache()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}))
	defer server.Close()

	_, err := _golang.GetFileInfoWithConfig("1.20.0", server.URL, time.Minute)
	if err == nil {
		t.Fatal("Expected file info retrieval error but got none")
	}
	t.Logf("Got expected error: %v", err)
}

// TestDownloader_downloadFile_ServerError tests server error handling
func TestDownloader_downloadFile_ServerError(t *testing.T) {
	config := createTestConfig(t)
	downloader := createTestDownloader(t, config)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}))
	defer server.Close()

	fileInfo := mockFileInfo()

	_, err := downloader.downloadFile(server.URL+"/"+fileInfo.Filename, fileInfo)
	if err == nil {
		t.Error("Expected server error but got none")
	}
	if !strings.Contains(err.Error(), "download failed with status 500") {
		t.Errorf("Expected server error, got: %v", err)
	}
}

// TestDownloader_downloadFile_NetworkTimeout tests network timeout handling
func TestDownloader_downloadFile_NetworkTimeout(t *testing.T) {
	config := createTestConfig(t)
	config.Download.Timeout = 1 * time.Nanosecond
	downloader := createTestDownloader(t, config)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		w.Write([]byte("delayed response"))
	}))
	defer server.Close()

	fileInfo := mockFileInfo()

	_, err := downloader.downloadFile(server.URL+"/"+fileInfo.Filename, fileInfo)
	if err == nil {
		t.Error("Expected timeout error but got none")
	}
}

// TestDownloader_extractArchive_UnsupportedFormatDirect tests unsupported archive formats directly
func TestDownloader_extractArchive_UnsupportedFormatDirect(t *testing.T) {
	config := createTestConfig(t)
	downloader := createTestDownloader(t, config)

	installDir := filepath.Join(config.InstallDir, "test-unsupported")

	archiveFile := filepath.Join(config.CacheDir, "test.rar")
	err := os.WriteFile(archiveFile, []byte("dummy"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(archiveFile)

	err = downloader.extractArchive(archiveFile, installDir)
	if err == nil {
		t.Error("Expected unsupported format error but got none")
	}
	if !strings.Contains(err.Error(), "unsupported archive format") {
		t.Errorf("Expected unsupported format error, got: %v", err)
	}
}

// TestDownloader_downloadFile_RetryExhaustion tests when all retry attempts are exhausted
func TestDownloader_downloadFile_RetryExhaustion(t *testing.T) {
	config := createTestConfig(t)
	config.Download.RetryCount = 1
	downloader := createTestDownloader(t, config)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server error", http.StatusInternalServerError)
	}))
	defer server.Close()

	fileInfo := mockFileInfo()
	fileInfo.Size = 1024

	_, err := downloader.downloadFile(server.URL+"/"+fileInfo.Filename, fileInfo)
	if err == nil {
		t.Error("Expected error but got none")
	}
	if !strings.Contains(err.Error(), "download failed with status 500") {
		t.Errorf("Expected download error, got: %v", err)
	}
}

// TestDownloader_downloadFile_PartialResume tests partial download resume functionality more thoroughly
func TestDownloader_downloadFile_PartialResume(t *testing.T) {
	testCases := []struct {
		name        string
		initialData string
		finalData   string
		expectError bool
	}{
		{
			name:        "Resume with matching partial content",
			initialData: "partial",
			finalData:   "partial-complete",
			expectError: false,
		},
		{
			name:        "Resume with full content already",
			initialData: "full-content",
			finalData:   "full-content",
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := createTestConfig(t)
			downloader := createTestDownloader(t, config)

			cachePath := filepath.Join(config.CacheDir, "resume-test.txt")
			err := os.WriteFile(cachePath, []byte(tc.initialData), 0644)
			if err != nil {
				t.Fatalf("Failed to create partial file: %v", err)
			}

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				rangeHeader := r.Header.Get("Range")
				if rangeHeader == fmt.Sprintf("bytes=%d-", len(tc.initialData)) {
					w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", len(tc.initialData), len(tc.finalData)-1, len(tc.finalData)))
					w.WriteHeader(http.StatusPartialContent)
					w.Write([]byte(tc.finalData[len(tc.initialData):]))
				} else {
					w.Write([]byte(tc.finalData))
				}
			}))
			defer server.Close()

			fileInfo := mockFileInfo()
			fileInfo.Size = int64(len(tc.finalData))
			fileInfo.Filename = "resume-test.txt"

			resultPath, err := downloader.downloadFile(server.URL+"/"+fileInfo.Filename, fileInfo)

			if tc.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("downloadFile failed: %v", err)
			}

			content, err := os.ReadFile(resultPath)
			if err != nil {
				t.Fatalf("Failed to read result file: %v", err)
			}
			if string(content) != tc.finalData {
				t.Errorf("Expected content %q, got %q", tc.finalData, string(content))
			}

			os.Remove(resultPath)
		})
	}
}

// TestDownloader_verifyChecksum_FileNotFound tests checksum verification with non-existent file
func TestDownloader_verifyChecksum_FileNotFound(t *testing.T) {
	config := createTestConfig(t)
	downloader := createTestDownloader(t, config)

	err := downloader.verifyChecksum("/non/existent/file.txt", "dummy")
	if err == nil {
		t.Error("Expected error for non-existent file but got none")
	}
}

// TestDownloader_extractArchive_UnsupportedFormat tests unsupported archive formats
func TestDownloader_extractArchive_UnsupportedFormat(t *testing.T) {
	testCases := []struct {
		name         string
		archiveName  string
		expectError  bool
		errorMessage string
	}{
		{
			name:         "RAR format",
			archiveName:  "test.rar",
			expectError:  true,
			errorMessage: "unsupported archive format",
		},
		{
			name:         "7Z format",
			archiveName:  "test.7z",
			expectError:  true,
			errorMessage: "unsupported archive format",
		},
		{
			name:         "TAR without GZ",
			archiveName:  "test.tar",
			expectError:  true,
			errorMessage: "unsupported archive format",
		},
		{
			name:         "EXE file",
			archiveName:  "test.exe",
			expectError:  true,
			errorMessage: "unsupported archive format",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := createTestConfig(t)
			downloader := createTestDownloader(t, config)

			installDir := filepath.Join(config.InstallDir, "test-unsupported")

			archiveFile := filepath.Join(config.CacheDir, tc.archiveName)
			err := os.WriteFile(archiveFile, []byte("dummy"), 0644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}
			defer os.Remove(archiveFile)

			err = downloader.extractArchive(archiveFile, installDir)

			if tc.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if !strings.Contains(err.Error(), tc.errorMessage) {
					t.Errorf("Expected error containing %q, got: %v", tc.errorMessage, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

type testTarEntry struct {
	header tar.Header
	body   string
}

func createTarGzEntries(t *testing.T, entries ...testTarEntry) []byte {
	t.Helper()
	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)
	tarWriter := tar.NewWriter(gzipWriter)
	for _, entry := range entries {
		header := entry.header
		if header.Typeflag == 0 {
			header.Typeflag = tar.TypeReg
		}
		if header.Typeflag == tar.TypeReg {
			header.Size = int64(len(entry.body))
		}
		if err := tarWriter.WriteHeader(&header); err != nil {
			t.Fatal(err)
		}
		if entry.body != "" {
			if _, err := tarWriter.Write([]byte(entry.body)); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func validGoArchive(t *testing.T, marker string) []byte {
	t.Helper()
	return createTarGzEntries(t,
		testTarEntry{header: tar.Header{Name: "go/bin/go", Mode: 0755}, body: "go executable"},
		testTarEntry{header: tar.Header{Name: "go/bin/go.exe", Mode: 0755}, body: "go executable"},
		testTarEntry{header: tar.Header{Name: "go/marker.txt", Mode: 0644}, body: marker},
	)
}

func writeTestArchive(t *testing.T, directory, name string, data []byte) string {
	t.Helper()
	path := filepath.Join(directory, name)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestDownloader_installArchiveTransactional(t *testing.T) {
	config := createTestConfig(t)
	downloader := createTestDownloader(t, config)
	installDir := filepath.Join(config.InstallDir, "go1.25.1")

	incomplete := writeTestArchive(t, config.CacheDir, "incomplete.tar.gz", createTarGzEntries(t,
		testTarEntry{header: tar.Header{Name: "go/README", Mode: 0644}, body: "missing executable"},
	))
	if err := downloader.installArchive(incomplete, installDir, "1.25.1"); err == nil {
		t.Fatal("incomplete archive was installed")
	}
	if _, err := os.Lstat(installDir); !os.IsNotExist(err) {
		t.Fatalf("failed install left final directory: %v", err)
	}
	partials, err := filepath.Glob(filepath.Join(config.InstallDir, ".go1.25.1.partial-*"))
	if err != nil || len(partials) != 0 {
		t.Fatalf("failed install left staging directories: %v (glob error: %v)", partials, err)
	}

	valid := writeTestArchive(t, config.CacheDir, "valid.tar.gz", validGoArchive(t, "retry succeeded"))
	if err := downloader.installArchive(valid, installDir, "1.25.1"); err != nil {
		t.Fatalf("retry after failed install did not succeed: %v", err)
	}
	marker, err := os.ReadFile(filepath.Join(installDir, "marker.txt"))
	if err != nil || string(marker) != "retry succeeded" {
		t.Fatalf("committed installation is invalid: marker=%q err=%v", marker, err)
	}
}

func TestDownloader_installArchiveConcurrentWinner(t *testing.T) {
	config := createTestConfig(t)
	downloader := createTestDownloader(t, config)
	installDir := filepath.Join(config.InstallDir, "go1.25.1")
	archives := []string{
		writeTestArchive(t, config.CacheDir, "a.tar.gz", validGoArchive(t, "a")),
		writeTestArchive(t, config.CacheDir, "b.tar.gz", validGoArchive(t, "b")),
	}

	start := make(chan struct{})
	errors := make(chan error, len(archives))
	for _, archive := range archives {
		archive := archive
		go func() {
			<-start
			errors <- downloader.installArchive(archive, installDir, "1.25.1")
		}()
	}
	close(start)
	successes := 0
	for range archives {
		if err := <-errors; err == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("concurrent installation winners = %d, want 1", successes)
	}
	marker, err := os.ReadFile(filepath.Join(installDir, "marker.txt"))
	if err != nil || (string(marker) != "a" && string(marker) != "b") {
		t.Fatalf("final installation combined or corrupt: marker=%q err=%v", marker, err)
	}
}

func TestDownloader_rejectsArchiveLinksAndUnsafePaths(t *testing.T) {
	config := createTestConfig(t)
	downloader := createTestDownloader(t, config)

	for _, entry := range []testTarEntry{
		{header: tar.Header{Name: "go/link", Typeflag: tar.TypeSymlink, Linkname: "/tmp/outside", Mode: 0777}},
		{header: tar.Header{Name: "go/link", Typeflag: tar.TypeLink, Linkname: "../../outside", Mode: 0777}},
	} {
		archive := writeTestArchive(t, config.CacheDir, fmt.Sprintf("link-%d.tar.gz", entry.header.Typeflag), createTarGzEntries(t, entry))
		if err := downloader.extractTarGz(archive, filepath.Join(config.InstallDir, fmt.Sprintf("link-%d", entry.header.Typeflag))); err == nil || !strings.Contains(err.Error(), "links are not allowed") {
			t.Fatalf("archive link type %d was not rejected: %v", entry.header.Typeflag, err)
		}
	}

	unsafeNames := []string{"../outside", "/absolute", `..\\outside`, `\\\\server\\share`, `C:\\outside`, "\x00bad"}
	for _, name := range unsafeNames {
		if _, err := validateArchivePath(name, config.InstallDir, name); err == nil {
			t.Errorf("unsafe archive path %q was accepted", name)
		}
	}
}

func TestDownloader_rejectsZipSymlink(t *testing.T) {
	config := createTestConfig(t)
	downloader := createTestDownloader(t, config)
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	header := &zip.FileHeader{Name: "go/link"}
	header.SetMode(os.ModeSymlink | 0777)
	entry, err := writer.CreateHeader(header)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte("../../outside")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	archive := writeTestArchive(t, config.CacheDir, "symlink.zip", buffer.Bytes())
	if err := downloader.extractZip(archive, filepath.Join(config.InstallDir, "zip-link")); err == nil || !strings.Contains(err.Error(), "links are not allowed") {
		t.Fatalf("zip symlink was not rejected: %v", err)
	}
}

func TestDownloader_enforcesArchiveLimits(t *testing.T) {
	config := createTestConfig(t)
	tests := []struct {
		name       string
		entries    []testTarEntry
		fileLimit  int64
		totalLimit int64
		entryLimit int
		want       string
	}{
		{
			name:       "file size",
			entries:    []testTarEntry{{header: tar.Header{Name: "go/large", Mode: 0644}, body: "12345"}},
			fileLimit:  4,
			totalLimit: 100,
			entryLimit: 10,
			want:       "exceeds size limit",
		},
		{
			name: "total size",
			entries: []testTarEntry{
				{header: tar.Header{Name: "go/a", Mode: 0644}, body: "1234"},
				{header: tar.Header{Name: "go/b", Mode: 0644}, body: "5678"},
			},
			fileLimit:  10,
			totalLimit: 7,
			entryLimit: 10,
			want:       "extracted size exceeds limit",
		},
		{
			name: "entry count",
			entries: []testTarEntry{
				{header: tar.Header{Name: "go/a", Mode: 0644}, body: "a"},
				{header: tar.Header{Name: "go/b", Mode: 0644}, body: "b"},
			},
			fileLimit:  10,
			totalLimit: 10,
			entryLimit: 1,
			want:       "too many entries",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			downloader := createTestDownloader(t, config)
			downloader.maxFileSize = test.fileLimit
			downloader.maxTotalSize = test.totalLimit
			downloader.maxArchiveEntries = test.entryLimit
			archive := writeTestArchive(t, config.CacheDir, strings.ReplaceAll(test.name, " ", "-")+".tar.gz", createTarGzEntries(t, test.entries...))
			err := downloader.extractTarGz(archive, filepath.Join(config.InstallDir, strings.ReplaceAll(test.name, " ", "-")))
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("limit error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestDownloader_invalidResumeRestartsFromZero(t *testing.T) {
	config := createTestConfig(t)
	config.Download.RetryDelay = 0
	downloader := createTestDownloader(t, config)
	content := []byte("complete download")
	fileInfo := mockFileInfo()
	fileInfo.Filename = "resume.tar.gz"
	fileInfo.Size = int64(len(content))
	if err := os.WriteFile(filepath.Join(config.CacheDir, fileInfo.Filename), content[:4], 0600); err != nil {
		t.Fatal(err)
	}
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		attempt := requests.Add(1)
		if attempt == 1 {
			w.WriteHeader(http.StatusPartialContent)
			w.Write(content[4:])
			return
		}
		w.Write(content)
	}))
	defer server.Close()

	path, err := downloader.downloadFile(server.URL+"/"+fileInfo.Filename, fileInfo)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(data, content) {
		t.Fatalf("fresh restart result = %q, err=%v", data, err)
	}
	if requests.Load() != 2 {
		t.Fatalf("requests = %d, want 2", requests.Load())
	}
}

func TestDownloader_cacheCommitIsCoordinated(t *testing.T) {
	config := createTestConfig(t)
	config.Download.RetryDelay = 0
	downloader := createTestDownloader(t, config)
	content := []byte("one complete cache value")
	fileInfo := mockFileInfo()
	fileInfo.Filename = "concurrent.tar.gz"
	fileInfo.Size = int64(len(content))
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		requests.Add(1)
		time.Sleep(25 * time.Millisecond)
		w.Write(content)
	}))
	defer server.Close()

	start := make(chan struct{})
	results := make(chan error, 2)
	var wait sync.WaitGroup
	for range 2 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			_, err := downloader.downloadFile(server.URL+"/"+fileInfo.Filename, fileInfo)
			results <- err
		}()
	}
	close(start)
	wait.Wait()
	close(results)
	for err := range results {
		if err != nil {
			t.Fatal(err)
		}
	}
	if requests.Load() != 1 {
		t.Fatalf("network requests = %d, want 1", requests.Load())
	}
	data, err := os.ReadFile(filepath.Join(config.CacheDir, fileInfo.Filename))
	if err != nil || !bytes.Equal(data, content) {
		t.Fatalf("cache value = %q, err=%v", data, err)
	}
}

func TestDownloader_rejectsTruncatedAndOversizedResponses(t *testing.T) {
	for _, test := range []struct {
		name string
		body []byte
		size int64
		want string
	}{
		{name: "truncated", body: []byte("short"), size: 10, want: "download truncated"},
		{name: "oversized", body: []byte("too-long"), size: 3, want: "exceeded expected size"},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := createTestConfig(t)
			config.Download.RetryDelay = 0
			downloader := createTestDownloader(t, config)
			fileInfo := mockFileInfo()
			fileInfo.Filename = test.name + ".tar.gz"
			fileInfo.Size = test.size
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
				w.Write(test.body)
			}))
			defer server.Close()
			_, err := downloader.downloadFile(server.URL+"/"+fileInfo.Filename, fileInfo)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("download error = %v, want %q", err, test.want)
			}
			if _, err := os.Lstat(filepath.Join(config.CacheDir, fileInfo.Filename)); !os.IsNotExist(err) {
				t.Fatalf("invalid response was committed to final cache: %v", err)
			}
		})
	}
}

func TestDownloader_downloadURLValidation(t *testing.T) {
	config := createTestConfig(t)
	downloader := createTestDownloader(t, config)
	content := []byte("content")
	fileInfo := mockFileInfo()
	fileInfo.Filename = "archive.tar.gz"
	fileInfo.Size = int64(len(content))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		w.Write(content)
	}))
	defer server.Close()
	if _, err := downloader.downloadFile(server.URL+"/archive.tar.gz?token=value", fileInfo); err != nil {
		t.Fatalf("query string changed cache filename: %v", err)
	}
	fileInfo.Filename = "different.tar.gz"
	if _, err := downloader.downloadFile(server.URL+"/archive.tar.gz", fileInfo); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("metadata filename mismatch was not rejected: %v", err)
	}
}

func TestParseContentRange(t *testing.T) {
	for _, test := range []struct {
		value string
		ok    bool
	}{
		{value: "bytes 5-9/10", ok: true},
		{value: "bytes 5-9/*"},
		{value: "bytes 10-9/10"},
		{value: "items 5-9/10"},
		{value: "bytes nope"},
	} {
		_, _, _, err := parseContentRange(test.value)
		if (err == nil) != test.ok {
			t.Errorf("parseContentRange(%q) error = %v, ok=%v", test.value, err, test.ok)
		}
	}
}
