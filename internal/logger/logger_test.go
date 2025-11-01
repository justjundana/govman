package logger

import (
	"bytes"
	"strings"
	"sync"
	"testing"
	"time"

	viper "github.com/spf13/viper"
)

func TestNew(t *testing.T) {
	testCases := []struct {
		name          string
		quiet         bool
		verbose       bool
		expectedLevel LogLevel
	}{
		{
			name:          "Default normal level",
			quiet:         false,
			verbose:       false,
			expectedLevel: NormalLevel,
		},
		{
			name:          "Quiet level when quiet flag is true",
			quiet:         true,
			verbose:       false,
			expectedLevel: QuietLevel,
		},
		{
			name:          "Verbose level when verbose flag is true",
			quiet:         false,
			verbose:       true,
			expectedLevel: VerboseLevel,
		},
		{
			name:          "Quiet takes precedence over verbose",
			quiet:         true,
			verbose:       true,
			expectedLevel: QuietLevel,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset viper for each test
			viper.Reset()
			viper.Set("quiet", tc.quiet)
			viper.Set("verbose", tc.verbose)

			logger := New()

			if logger.Level() != tc.expectedLevel {
				t.Errorf("Expected level %v, got %v", tc.expectedLevel, logger.Level())
			}

			if logger.NormalWriter() == nil {
				t.Error("Normal writer should not be nil")
			}

			if logger.VerboseWriter() == nil {
				t.Error("Verbose writer should not be nil")
			}
		})
	}
}

func TestSetLevel(t *testing.T) {
	testCases := []struct {
		name     string
		newLevel LogLevel
	}{
		{
			name:     "Set to QuietLevel",
			newLevel: QuietLevel,
		},
		{
			name:     "Set to NormalLevel",
			newLevel: NormalLevel,
		},
		{
			name:     "Set to VerboseLevel",
			newLevel: VerboseLevel,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			viper.Reset()
			logger := New()
			logger.SetLevel(tc.newLevel)

			if logger.Level() != tc.newLevel {
				t.Errorf("Expected level %v, got %v", tc.newLevel, logger.Level())
			}
		})
	}
}

func TestSetWriters(t *testing.T) {
	testCases := []struct {
		name string
		test func(t *testing.T, logger *Logger)
	}{
		{
			name: "SetNormalWriter",
			test: func(t *testing.T, logger *Logger) {
				buf := &bytes.Buffer{}
				logger.SetNormalWriter(buf)

				if logger.NormalWriter() != buf {
					t.Error("Normal writer not set correctly")
				}
			},
		},
		{
			name: "SetVerboseWriter",
			test: func(t *testing.T, logger *Logger) {
				buf := &bytes.Buffer{}
				logger.SetVerboseWriter(buf)

				if logger.VerboseWriter() != buf {
					t.Error("Verbose writer not set correctly")
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			viper.Reset()
			logger := New()
			tc.test(t, logger)
		})
	}
}

func TestError(t *testing.T) {
	testCases := []struct {
		name           string
		level          LogLevel
		format         string
		args           []interface{}
		expectedOutput string
		shouldLog      bool
	}{
		{
			name:           "Error logs at QuietLevel",
			level:          QuietLevel,
			format:         "test error %s",
			args:           []interface{}{"message"},
			expectedOutput: "Error: test error message\n",
			shouldLog:      true,
		},
		{
			name:           "Error logs at NormalLevel",
			level:          NormalLevel,
			format:         "another error",
			args:           []interface{}{},
			expectedOutput: "Error: another error\n",
			shouldLog:      true,
		},
		{
			name:           "Error logs at VerboseLevel",
			level:          VerboseLevel,
			format:         "verbose error",
			args:           []interface{}{},
			expectedOutput: "Error: verbose error\n",
			shouldLog:      true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			viper.Reset()
			logger := New()
			buf := &bytes.Buffer{}
			logger.SetNormalWriter(buf)
			logger.SetLevel(tc.level)

			logger.Error(tc.format, tc.args...)

			output := buf.String()
			if tc.shouldLog && output != tc.expectedOutput {
				t.Errorf("Expected output %q, got %q", tc.expectedOutput, output)
			}
		})
	}
}

func TestInfo(t *testing.T) {
	testCases := []struct {
		name           string
		level          LogLevel
		format         string
		args           []interface{}
		expectedOutput string
		shouldLog      bool
	}{
		{
			name:           "Info does not log at QuietLevel",
			level:          QuietLevel,
			format:         "test info",
			args:           []interface{}{},
			expectedOutput: "",
			shouldLog:      false,
		},
		{
			name:           "Info logs at NormalLevel",
			level:          NormalLevel,
			format:         "processing %d items",
			args:           []interface{}{5},
			expectedOutput: "processing 5 items\n",
			shouldLog:      true,
		},
		{
			name:           "Info logs at VerboseLevel",
			level:          VerboseLevel,
			format:         "verbose info",
			args:           []interface{}{},
			expectedOutput: "verbose info\n",
			shouldLog:      true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			viper.Reset()
			logger := New()
			buf := &bytes.Buffer{}
			logger.SetNormalWriter(buf)
			logger.SetLevel(tc.level)

			logger.Info(tc.format, tc.args...)

			output := buf.String()
			if output != tc.expectedOutput {
				t.Errorf("Expected output %q, got %q", tc.expectedOutput, output)
			}
		})
	}
}

func TestSuccess(t *testing.T) {
	testCases := []struct {
		name           string
		level          LogLevel
		format         string
		args           []interface{}
		expectedOutput string
		shouldLog      bool
	}{
		{
			name:           "Success does not log at QuietLevel",
			level:          QuietLevel,
			format:         "completed",
			args:           []interface{}{},
			expectedOutput: "",
			shouldLog:      false,
		},
		{
			name:           "Success logs at NormalLevel",
			level:          NormalLevel,
			format:         "operation %s completed",
			args:           []interface{}{"backup"},
			expectedOutput: "Success: operation backup completed\n",
			shouldLog:      true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			viper.Reset()
			logger := New()
			buf := &bytes.Buffer{}
			logger.SetNormalWriter(buf)
			logger.SetLevel(tc.level)

			logger.Success(tc.format, tc.args...)

			output := buf.String()
			if output != tc.expectedOutput {
				t.Errorf("Expected output %q, got %q", tc.expectedOutput, output)
			}
		})
	}
}

func TestWarning(t *testing.T) {
	testCases := []struct {
		name           string
		level          LogLevel
		format         string
		args           []interface{}
		expectedOutput string
		shouldLog      bool
	}{
		{
			name:           "Warning does not log at QuietLevel",
			level:          QuietLevel,
			format:         "deprecated",
			args:           []interface{}{},
			expectedOutput: "",
			shouldLog:      false,
		},
		{
			name:           "Warning logs at NormalLevel",
			level:          NormalLevel,
			format:         "low disk space: %d%%",
			args:           []interface{}{10},
			expectedOutput: "Warning: low disk space: 10%\n",
			shouldLog:      true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			viper.Reset()
			logger := New()
			buf := &bytes.Buffer{}
			logger.SetNormalWriter(buf)
			logger.SetLevel(tc.level)

			logger.Warning(tc.format, tc.args...)

			output := buf.String()
			if output != tc.expectedOutput {
				t.Errorf("Expected output %q, got %q", tc.expectedOutput, output)
			}
		})
	}
}

func TestVerbose(t *testing.T) {
	testCases := []struct {
		name           string
		level          LogLevel
		format         string
		args           []interface{}
		expectedOutput string
		shouldLog      bool
	}{
		{
			name:           "Verbose does not log at QuietLevel",
			level:          QuietLevel,
			format:         "verbose message",
			args:           []interface{}{},
			expectedOutput: "",
			shouldLog:      false,
		},
		{
			name:           "Verbose does not log at NormalLevel",
			level:          NormalLevel,
			format:         "verbose message",
			args:           []interface{}{},
			expectedOutput: "",
			shouldLog:      false,
		},
		{
			name:           "Verbose logs at VerboseLevel",
			level:          VerboseLevel,
			format:         "detailed info: %s",
			args:           []interface{}{"data"},
			expectedOutput: "[VERBOSE] detailed info: data\n",
			shouldLog:      true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			viper.Reset()
			logger := New()
			buf := &bytes.Buffer{}
			logger.SetVerboseWriter(buf)
			logger.SetLevel(tc.level)

			logger.Verbose(tc.format, tc.args...)

			output := buf.String()
			if output != tc.expectedOutput {
				t.Errorf("Expected output %q, got %q", tc.expectedOutput, output)
			}
		})
	}
}

func TestDebug(t *testing.T) {
	testCases := []struct {
		name           string
		level          LogLevel
		format         string
		args           []interface{}
		expectedOutput string
		shouldLog      bool
	}{
		{
			name:           "Debug does not log at NormalLevel",
			level:          NormalLevel,
			format:         "debug info",
			args:           []interface{}{},
			expectedOutput: "",
			shouldLog:      false,
		},
		{
			name:           "Debug logs at VerboseLevel",
			level:          VerboseLevel,
			format:         "variable value: %v",
			args:           []interface{}{42},
			expectedOutput: "[DEBUG] variable value: 42\n",
			shouldLog:      true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			viper.Reset()
			logger := New()
			buf := &bytes.Buffer{}
			logger.SetVerboseWriter(buf)
			logger.SetLevel(tc.level)

			logger.Debug(tc.format, tc.args...)

			output := buf.String()
			if output != tc.expectedOutput {
				t.Errorf("Expected output %q, got %q", tc.expectedOutput, output)
			}
		})
	}
}

func TestProgress(t *testing.T) {
	testCases := []struct {
		name           string
		level          LogLevel
		format         string
		args           []interface{}
		expectedOutput string
		shouldLog      bool
	}{
		{
			name:           "Progress does not log at QuietLevel",
			level:          QuietLevel,
			format:         "50%% complete",
			args:           []interface{}{},
			expectedOutput: "",
			shouldLog:      false,
		},
		{
			name:           "Progress logs at NormalLevel",
			level:          NormalLevel,
			format:         "%d/%d files processed",
			args:           []interface{}{5, 10},
			expectedOutput: "Progress: 5/10 files processed\n",
			shouldLog:      true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			viper.Reset()
			logger := New()
			buf := &bytes.Buffer{}
			logger.SetNormalWriter(buf)
			logger.SetLevel(tc.level)

			logger.Progress(tc.format, tc.args...)

			output := buf.String()
			if output != tc.expectedOutput {
				t.Errorf("Expected output %q, got %q", tc.expectedOutput, output)
			}
		})
	}
}

func TestStep(t *testing.T) {
	testCases := []struct {
		name           string
		level          LogLevel
		format         string
		args           []interface{}
		expectedOutput string
		shouldLog      bool
	}{
		{
			name:           "Step does not log at NormalLevel",
			level:          NormalLevel,
			format:         "initializing",
			args:           []interface{}{},
			expectedOutput: "",
			shouldLog:      false,
		},
		{
			name:           "Step logs at VerboseLevel",
			level:          VerboseLevel,
			format:         "step %d: %s",
			args:           []interface{}{1, "setup"},
			expectedOutput: "Step: step 1: setup\n",
			shouldLog:      true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			viper.Reset()
			logger := New()
			buf := &bytes.Buffer{}
			logger.SetVerboseWriter(buf)
			logger.SetLevel(tc.level)

			logger.Step(tc.format, tc.args...)

			output := buf.String()
			if output != tc.expectedOutput {
				t.Errorf("Expected output %q, got %q", tc.expectedOutput, output)
			}
		})
	}
}

func TestDownload(t *testing.T) {
	testCases := []struct {
		name           string
		level          LogLevel
		format         string
		args           []interface{}
		expectedOutput string
		shouldLog      bool
	}{
		{
			name:           "Download does not log at QuietLevel",
			level:          QuietLevel,
			format:         "downloading file",
			args:           []interface{}{},
			expectedOutput: "",
			shouldLog:      false,
		},
		{
			name:           "Download logs at NormalLevel",
			level:          NormalLevel,
			format:         "downloading %s",
			args:           []interface{}{"package.tar.gz"},
			expectedOutput: "Download: downloading package.tar.gz\n",
			shouldLog:      true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			viper.Reset()
			logger := New()
			buf := &bytes.Buffer{}
			logger.SetNormalWriter(buf)
			logger.SetLevel(tc.level)

			logger.Download(tc.format, tc.args...)

			output := buf.String()
			if output != tc.expectedOutput {
				t.Errorf("Expected output %q, got %q", tc.expectedOutput, output)
			}
		})
	}
}

func TestExtract(t *testing.T) {
	testCases := []struct {
		name           string
		level          LogLevel
		format         string
		args           []interface{}
		expectedOutput string
		shouldLog      bool
	}{
		{
			name:           "Extract does not log at QuietLevel",
			level:          QuietLevel,
			format:         "extracting archive",
			args:           []interface{}{},
			expectedOutput: "",
			shouldLog:      false,
		},
		{
			name:           "Extract logs at NormalLevel",
			level:          NormalLevel,
			format:         "extracting to %s",
			args:           []interface{}{"/tmp/extract"},
			expectedOutput: "Extract: extracting to /tmp/extract\n",
			shouldLog:      true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			viper.Reset()
			logger := New()
			buf := &bytes.Buffer{}
			logger.SetNormalWriter(buf)
			logger.SetLevel(tc.level)

			logger.Extract(tc.format, tc.args...)

			output := buf.String()
			if output != tc.expectedOutput {
				t.Errorf("Expected output %q, got %q", tc.expectedOutput, output)
			}
		})
	}
}

func TestVerify(t *testing.T) {
	testCases := []struct {
		name           string
		level          LogLevel
		format         string
		args           []interface{}
		expectedOutput string
		shouldLog      bool
	}{
		{
			name:           "Verify does not log at QuietLevel",
			level:          QuietLevel,
			format:         "verifying checksum",
			args:           []interface{}{},
			expectedOutput: "",
			shouldLog:      false,
		},
		{
			name:           "Verify logs at NormalLevel",
			level:          NormalLevel,
			format:         "verifying %s",
			args:           []interface{}{"signature"},
			expectedOutput: "Verify: verifying signature\n",
			shouldLog:      true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			viper.Reset()
			logger := New()
			buf := &bytes.Buffer{}
			logger.SetNormalWriter(buf)
			logger.SetLevel(tc.level)

			logger.Verify(tc.format, tc.args...)

			output := buf.String()
			if output != tc.expectedOutput {
				t.Errorf("Expected output %q, got %q", tc.expectedOutput, output)
			}
		})
	}
}

func TestInternalProgress(t *testing.T) {
	testCases := []struct {
		name           string
		level          LogLevel
		format         string
		args           []interface{}
		expectedOutput string
		shouldLog      bool
	}{
		{
			name:           "InternalProgress does not log at NormalLevel",
			level:          NormalLevel,
			format:         "internal step",
			args:           []interface{}{},
			expectedOutput: "",
			shouldLog:      false,
		},
		{
			name:           "InternalProgress logs at VerboseLevel",
			level:          VerboseLevel,
			format:         "processing item %d",
			args:           []interface{}{3},
			expectedOutput: "[INTERNAL] processing item 3\n",
			shouldLog:      true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			viper.Reset()
			logger := New()
			buf := &bytes.Buffer{}
			logger.SetVerboseWriter(buf)
			logger.SetLevel(tc.level)

			logger.InternalProgress(tc.format, tc.args...)

			output := buf.String()
			if output != tc.expectedOutput {
				t.Errorf("Expected output %q, got %q", tc.expectedOutput, output)
			}
		})
	}
}

func TestTimer(t *testing.T) {
	testCases := []struct {
		name             string
		level            LogLevel
		timerName        string
		shouldLogStart   bool
		shouldLogStop    bool
		expectedInOutput []string
	}{
		{
			name:             "Timer does not log at NormalLevel",
			level:            NormalLevel,
			timerName:        "operation",
			shouldLogStart:   false,
			shouldLogStop:    false,
			expectedInOutput: []string{},
		},
		{
			name:           "Timer logs at VerboseLevel",
			level:          VerboseLevel,
			timerName:      "database migration",
			shouldLogStart: true,
			shouldLogStop:  true,
			expectedInOutput: []string{
				"[VERBOSE] Starting database migration...",
				"[VERBOSE] Completed database migration in",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			viper.Reset()
			logger := New()
			buf := &bytes.Buffer{}
			logger.SetVerboseWriter(buf)
			logger.SetLevel(tc.level)

			timer := logger.StartTimer(tc.timerName)
			time.Sleep(10 * time.Millisecond)
			logger.StopTimer(timer)

			output := buf.String()

			for _, expected := range tc.expectedInOutput {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain %q, got %q", expected, output)
				}
			}

			if !tc.shouldLogStart && !tc.shouldLogStop && output != "" {
				t.Errorf("Expected no output, got %q", output)
			}
		})
	}
}

func TestTimerNil(t *testing.T) {
	viper.Reset()
	logger := New()
	buf := &bytes.Buffer{}
	logger.SetVerboseWriter(buf)
	logger.SetLevel(VerboseLevel)

	// StopTimer with nil timer should not panic
	logger.StopTimer(nil)

	output := buf.String()
	if output != "" {
		t.Errorf("Expected no output for nil timer, got %q", output)
	}
}

func TestGlobalLogger(t *testing.T) {
	testCases := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "Get returns singleton",
			test: func(t *testing.T) {
				logger1 := Get()
				logger2 := Get()
				if logger1 != logger2 {
					t.Error("Get() should return the same instance")
				}
			},
		},
		{
			name: "Global Error function",
			test: func(t *testing.T) {
				viper.Reset()
				globalLogger = nil
				once = sync.Once{}

				buf := &bytes.Buffer{}
				logger := Get()
				logger.SetNormalWriter(buf)
				logger.SetLevel(NormalLevel)

				Error("global error %s", "test")

				output := buf.String()
				expected := "Error: global error test\n"
				if output != expected {
					t.Errorf("Expected %q, got %q", expected, output)
				}
			},
		},
		{
			name: "Global Info function",
			test: func(t *testing.T) {
				viper.Reset()
				globalLogger = nil
				once = sync.Once{}

				buf := &bytes.Buffer{}
				logger := Get()
				logger.SetNormalWriter(buf)
				logger.SetLevel(NormalLevel)

				Info("global info")

				output := buf.String()
				expected := "global info\n"
				if output != expected {
					t.Errorf("Expected %q, got %q", expected, output)
				}
			},
		},
		{
			name: "Global Success function",
			test: func(t *testing.T) {
				viper.Reset()
				globalLogger = nil
				once = sync.Once{}

				buf := &bytes.Buffer{}
				logger := Get()
				logger.SetNormalWriter(buf)
				logger.SetLevel(NormalLevel)

				Success("task completed")

				output := buf.String()
				expected := "Success: task completed\n"
				if output != expected {
					t.Errorf("Expected %q, got %q", expected, output)
				}
			},
		},
		{
			name: "Global Warning function",
			test: func(t *testing.T) {
				viper.Reset()
				globalLogger = nil
				once = sync.Once{}

				buf := &bytes.Buffer{}
				logger := Get()
				logger.SetNormalWriter(buf)
				logger.SetLevel(NormalLevel)

				Warning("warning message")

				output := buf.String()
				expected := "Warning: warning message\n"
				if output != expected {
					t.Errorf("Expected %q, got %q", expected, output)
				}
			},
		},
		{
			name: "Global Verbose function",
			test: func(t *testing.T) {
				viper.Reset()
				globalLogger = nil
				once = sync.Once{}

				buf := &bytes.Buffer{}
				logger := Get()
				logger.SetVerboseWriter(buf)
				logger.SetLevel(VerboseLevel)

				Verbose("verbose details")

				output := buf.String()
				expected := "[VERBOSE] verbose details\n"
				if output != expected {
					t.Errorf("Expected %q, got %q", expected, output)
				}
			},
		},
		{
			name: "Global Debug function",
			test: func(t *testing.T) {
				viper.Reset()
				globalLogger = nil
				once = sync.Once{}

				buf := &bytes.Buffer{}
				logger := Get()
				logger.SetVerboseWriter(buf)
				logger.SetLevel(VerboseLevel)

				Debug("debug info")

				output := buf.String()
				expected := "[DEBUG] debug info\n"
				if output != expected {
					t.Errorf("Expected %q, got %q", expected, output)
				}
			},
		},
		{
			name: "Global Progress function",
			test: func(t *testing.T) {
				viper.Reset()
				globalLogger = nil
				once = sync.Once{}

				buf := &bytes.Buffer{}
				logger := Get()
				logger.SetNormalWriter(buf)
				logger.SetLevel(NormalLevel)

				Progress("50%% done")

				output := buf.String()
				expected := "Progress: 50% done\n"
				if output != expected {
					t.Errorf("Expected %q, got %q", expected, output)
				}
			},
		},
		{
			name: "Global Download function",
			test: func(t *testing.T) {
				viper.Reset()
				globalLogger = nil
				once = sync.Once{}

				buf := &bytes.Buffer{}
				logger := Get()
				logger.SetNormalWriter(buf)
				logger.SetLevel(NormalLevel)

				Download("file.zip")

				output := buf.String()
				expected := "Download: file.zip\n"
				if output != expected {
					t.Errorf("Expected %q, got %q", expected, output)
				}
			},
		},
		{
			name: "Global Extract function",
			test: func(t *testing.T) {
				viper.Reset()
				globalLogger = nil
				once = sync.Once{}

				buf := &bytes.Buffer{}
				logger := Get()
				logger.SetNormalWriter(buf)
				logger.SetLevel(NormalLevel)

				Extract("extracting")

				output := buf.String()
				expected := "Extract: extracting\n"
				if output != expected {
					t.Errorf("Expected %q, got %q", expected, output)
				}
			},
		},
		{
			name: "Global Verify function",
			test: func(t *testing.T) {
				viper.Reset()
				globalLogger = nil
				once = sync.Once{}

				buf := &bytes.Buffer{}
				logger := Get()
				logger.SetNormalWriter(buf)
				logger.SetLevel(NormalLevel)

				Verify("checksum")

				output := buf.String()
				expected := "Verify: checksum\n"
				if output != expected {
					t.Errorf("Expected %q, got %q", expected, output)
				}
			},
		},
		{
			name: "Global StartTimer function",
			test: func(t *testing.T) {
				viper.Reset()
				globalLogger = nil
				once = sync.Once{}

				buf := &bytes.Buffer{}
				logger := Get()
				logger.SetVerboseWriter(buf)
				logger.SetLevel(VerboseLevel)

				timer := StartTimer("test timer")

				if timer == nil {
					t.Error("Timer should not be nil")
				}

				output := buf.String()
				expected := "[VERBOSE] Starting test timer...\n"
				if output != expected {
					t.Errorf("Expected %q, got %q", expected, output)
				}
			},
		},
		{
			name: "Global StopTimer function",
			test: func(t *testing.T) {
				viper.Reset()
				globalLogger = nil
				once = sync.Once{}

				buf := &bytes.Buffer{}
				logger := Get()
				logger.SetVerboseWriter(buf)
				logger.SetLevel(VerboseLevel)

				timer := StartTimer("test operation")
				buf.Reset() // Clear the start message
				time.Sleep(10 * time.Millisecond)
				StopTimer(timer)

				output := buf.String()
				if !strings.Contains(output, "[VERBOSE] Completed test operation in") {
					t.Errorf("Expected completion message, got %q", output)
				}
			},
		},
		{
			name: "Global ErrorWithHelp function",
			test: func(t *testing.T) {
				viper.Reset()
				globalLogger = nil
				once = sync.Once{}

				buf := &bytes.Buffer{}
				logger := Get()
				logger.SetNormalWriter(buf)
				logger.SetLevel(NormalLevel)

				ErrorWithHelp("connection failed", "Check network settings")

				output := buf.String()
				expected := "Error: connection failed\nHelp: Check network settings\n"
				if output != expected {
					t.Errorf("Expected %q, got %q", expected, output)
				}
			},
		},
		{
			name: "Global Step function",
			test: func(t *testing.T) {
				viper.Reset()
				globalLogger = nil
				once = sync.Once{}

				buf := &bytes.Buffer{}
				logger := Get()
				logger.SetVerboseWriter(buf)
				logger.SetLevel(VerboseLevel)

				Step("initialization")

				output := buf.String()
				expected := "Step: initialization\n"
				if output != expected {
					t.Errorf("Expected %q, got %q", expected, output)
				}
			},
		},
		{
			name: "Global InternalProgress function",
			test: func(t *testing.T) {
				viper.Reset()
				globalLogger = nil
				once = sync.Once{}

				buf := &bytes.Buffer{}
				logger := Get()
				logger.SetVerboseWriter(buf)
				logger.SetLevel(VerboseLevel)

				InternalProgress("processing")

				output := buf.String()
				expected := "[INTERNAL] processing\n"
				if output != expected {
					t.Errorf("Expected %q, got %q", expected, output)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.test(t)
		})
	}
}

func TestConcurrency(t *testing.T) {
	testCases := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "Concurrent SetLevel calls",
			test: func(t *testing.T) {
				viper.Reset()
				logger := New()
				var wg sync.WaitGroup

				for i := 0; i < 100; i++ {
					wg.Add(1)
					go func(level LogLevel) {
						defer wg.Done()
						logger.SetLevel(level)
					}(LogLevel(i % 3))
				}

				wg.Wait()

				// Should not panic and should have a valid level
				level := logger.Level()
				if level < QuietLevel || level > VerboseLevel {
					t.Errorf("Invalid level after concurrent updates: %v", level)
				}
			},
		},
		{
			name: "Concurrent writer updates",
			test: func(t *testing.T) {
				viper.Reset()
				logger := New()
				var wg sync.WaitGroup

				for i := 0; i < 50; i++ {
					wg.Add(2)
					go func() {
						defer wg.Done()
						buf := &bytes.Buffer{}
						logger.SetNormalWriter(buf)
					}()
					go func() {
						defer wg.Done()
						buf := &bytes.Buffer{}
						logger.SetVerboseWriter(buf)
					}()
				}

				wg.Wait()

				// Should not panic
				if logger.NormalWriter() == nil {
					t.Error("Normal writer should not be nil")
				}
				if logger.VerboseWriter() == nil {
					t.Error("Verbose writer should not be nil")
				}
			},
		},
		{
			name: "Concurrent log writes",
			test: func(t *testing.T) {
				viper.Reset()
				logger := New()
				buf := &bytes.Buffer{}
				logger.SetNormalWriter(buf)
				logger.SetVerboseWriter(buf)
				logger.SetLevel(VerboseLevel)

				var wg sync.WaitGroup

				for i := 0; i < 100; i++ {
					wg.Add(1)
					go func(idx int) {
						defer wg.Done()
						logger.Info("message %d", idx)
						logger.Verbose("verbose %d", idx)
						logger.Error("error %d", idx)
					}(i)
				}

				wg.Wait()

				// Should not panic and should have some output
				output := buf.String()
				if output == "" {
					t.Error("Expected some output from concurrent writes")
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.test(t)
		})
	}
}

func TestLogLevelConstants(t *testing.T) {
	testCases := []struct {
		name  string
		level LogLevel
		value int
	}{
		{
			name:  "QuietLevel is 0",
			level: QuietLevel,
			value: 0,
		},
		{
			name:  "NormalLevel is 1",
			level: NormalLevel,
			value: 1,
		},
		{
			name:  "VerboseLevel is 2",
			level: VerboseLevel,
			value: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if int(tc.level) != tc.value {
				t.Errorf("Expected %s to be %d, got %d", tc.name, tc.value, int(tc.level))
			}
		})
	}
}

func TestTimerFields(t *testing.T) {
	testCases := []struct {
		name      string
		timerName string
	}{
		{
			name:      "Timer has correct name",
			timerName: "test operation",
		},
		{
			name:      "Timer with empty name",
			timerName: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			viper.Reset()
			logger := New()
			logger.SetLevel(VerboseLevel)

			timer := logger.StartTimer(tc.timerName)

			if timer.name != tc.timerName {
				t.Errorf("Expected timer name %q, got %q", tc.timerName, timer.name)
			}

			if timer.start.IsZero() {
				t.Error("Timer start time should not be zero")
			}
		})
	}
}
