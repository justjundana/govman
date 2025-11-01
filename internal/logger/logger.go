package logger

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	viper "github.com/spf13/viper"
)

type LogLevel int

const (
	QuietLevel LogLevel = iota
	NormalLevel
	VerboseLevel
)

type Logger struct {
	level         LogLevel
	normalWriter  io.Writer
	verboseWriter io.Writer
	mutex         sync.Mutex
}

type Timer struct {
	start time.Time
	name  string
}

// New constructs a Logger and sets its initial level based on viper flags (quiet/verbose).
func New() *Logger {
	l := &Logger{
		normalWriter:  os.Stderr,
		verboseWriter: os.Stderr,
	}

	if viper.GetBool("quiet") {
		l.level = QuietLevel
	} else if viper.GetBool("verbose") {
		l.level = VerboseLevel
	} else {
		l.level = NormalLevel
	}

	return l
}

// SetLevel updates the logger's verbosity level in a thread-safe manner.
func (l *Logger) SetLevel(level LogLevel) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.level = level
}

// SetNormalWriter sets the destination writer for normal-level logs.
func (l *Logger) SetNormalWriter(writer io.Writer) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.normalWriter = writer
}

// SetVerboseWriter sets the destination writer for verbose-level logs.
func (l *Logger) SetVerboseWriter(writer io.Writer) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.verboseWriter = writer
}

// Level returns the current log level.
func (l *Logger) Level() LogLevel {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	return l.level
}

// NormalWriter returns the current writer used for normal-level logs.
func (l *Logger) NormalWriter() io.Writer {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	return l.normalWriter
}

// VerboseWriter returns the current writer used for verbose-level logs.
func (l *Logger) VerboseWriter() io.Writer {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	return l.verboseWriter
}

// Error logs an error message to the normal writer (shown unless fully quiet).
func (l *Logger) Error(format string, args ...interface{}) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if l.level >= QuietLevel {
		fmt.Fprintf(l.normalWriter, "Error: "+format+"\n", args...)
	}
}

// ErrorWithHelp logs an error message and an optional help hint.
func (l *Logger) ErrorWithHelp(errorMsg, helpMsg string, args ...interface{}) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if l.level >= QuietLevel {
		fmt.Fprintf(l.normalWriter, "Error: "+errorMsg+"\n", args...)
		if helpMsg != "" {
			fmt.Fprintf(l.normalWriter, "Help: %s\n", helpMsg)
		}
	}
}

// StartTimer begins a named timer; in verbose mode it logs the start.
func (l *Logger) StartTimer(name string) *Timer {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if l.level >= VerboseLevel {
		fmt.Fprintf(l.verboseWriter, "[VERBOSE] Starting %s...\n", name)
	}
	return &Timer{
		start: time.Now(),
		name:  name,
	}
}

// StopTimer stops a timer and logs the elapsed duration in verbose mode.
func (l *Logger) StopTimer(t *Timer) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if l.level >= VerboseLevel && t != nil {
		duration := time.Since(t.start)
		fmt.Fprintf(l.verboseWriter, "[VERBOSE] Completed %s in %v\n", t.name, duration)
	}
}

// Info logs an informational message at normal level.
func (l *Logger) Info(format string, args ...interface{}) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if l.level >= NormalLevel {
		fmt.Fprintf(l.normalWriter, format+"\n", args...)
	}
}

// Success logs a success message at normal level.
func (l *Logger) Success(format string, args ...interface{}) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if l.level >= NormalLevel {
		fmt.Fprintf(l.normalWriter, "Success: "+format+"\n", args...)
	}
}

// Warning logs a warning message at normal level.
func (l *Logger) Warning(format string, args ...interface{}) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if l.level >= NormalLevel {
		fmt.Fprintf(l.normalWriter, "Warning: "+format+"\n", args...)
	}
}

// Verbose logs a detailed message at verbose level.
func (l *Logger) Verbose(format string, args ...interface{}) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if l.level >= VerboseLevel {
		fmt.Fprintf(l.verboseWriter, "[VERBOSE] "+format+"\n", args...)
	}
}

// Debug logs a debug message at verbose level.
func (l *Logger) Debug(format string, args ...interface{}) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if l.level >= VerboseLevel {
		fmt.Fprintf(l.verboseWriter, "[DEBUG] "+format+"\n", args...)
	}
}

// Progress logs a progress update at normal level.
func (l *Logger) Progress(format string, args ...interface{}) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if l.level >= NormalLevel {
		fmt.Fprintf(l.normalWriter, "Progress: "+format+"\n", args...)
	}
}

// Step logs a step-level message (verbose flow guidance).
func (l *Logger) Step(format string, args ...interface{}) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if l.level >= VerboseLevel {
		fmt.Fprintf(l.verboseWriter, "Step: "+format+"\n", args...)
	}
}

// Download logs a download-related message at normal level.
func (l *Logger) Download(format string, args ...interface{}) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if l.level >= NormalLevel {
		fmt.Fprintf(l.normalWriter, "Download: "+format+"\n", args...)
	}
}

// Extract logs an extraction-related message at normal level.
func (l *Logger) Extract(format string, args ...interface{}) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if l.level >= NormalLevel {
		fmt.Fprintf(l.normalWriter, "Extract: "+format+"\n", args...)
	}
}

// Verify logs a verification-related message at normal level.
func (l *Logger) Verify(format string, args ...interface{}) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if l.level >= NormalLevel {
		fmt.Fprintf(l.normalWriter, "Verify: "+format+"\n", args...)
	}
}

// InternalProgress logs internal progress details at verbose level.
func (l *Logger) InternalProgress(format string, args ...interface{}) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if l.level >= VerboseLevel {
		fmt.Fprintf(l.verboseWriter, "[INTERNAL] "+format+"\n", args...)
	}
}

var globalLogger *Logger
var once sync.Once

// Get returns the singleton global logger instance (initialized on first use).
func Get() *Logger {
	once.Do(func() {
		globalLogger = New()
	})
	return globalLogger
}

// Error is a package-level proxy to Logger.Error.
func Error(format string, args ...interface{}) {
	Get().Error(format, args...)
}

// Info is a package-level proxy to Logger.Info.
func Info(format string, args ...interface{}) {
	Get().Info(format, args...)
}

// Success is a package-level proxy to Logger.Success.
func Success(format string, args ...interface{}) {
	Get().Success(format, args...)
}

// Warning is a package-level proxy to Logger.Warning.
func Warning(format string, args ...interface{}) {
	Get().Warning(format, args...)
}

// Verbose is a package-level proxy to Logger.Verbose.
func Verbose(format string, args ...interface{}) {
	Get().Verbose(format, args...)
}

// Debug is a package-level proxy to Logger.Debug.
func Debug(format string, args ...interface{}) {
	Get().Debug(format, args...)
}

// Progress is a package-level proxy to Logger.Progress.
func Progress(format string, args ...interface{}) {
	Get().Progress(format, args...)
}

// Download is a package-level proxy to Logger.Download.
func Download(format string, args ...interface{}) {
	Get().Download(format, args...)
}

// Extract is a package-level proxy to Logger.Extract.
func Extract(format string, args ...interface{}) {
	Get().Extract(format, args...)
}

// Verify is a package-level proxy to Logger.Verify.
func Verify(format string, args ...interface{}) {
	Get().Verify(format, args...)
}

// StartTimer is a package-level proxy to Logger.StartTimer.
func StartTimer(name string) *Timer {
	return Get().StartTimer(name)
}

// StopTimer is a package-level proxy to Logger.StopTimer.
func StopTimer(t *Timer) {
	Get().StopTimer(t)
}

// ErrorWithHelp is a package-level proxy to Logger.ErrorWithHelp.
func ErrorWithHelp(errorMsg, helpMsg string, args ...interface{}) {
	Get().ErrorWithHelp(errorMsg, helpMsg, args...)
}

// Step is a package-level proxy to Logger.Step.
func Step(format string, args ...interface{}) {
	Get().Step(format, args...)
}

// InternalProgress is a package-level proxy to Logger.InternalProgress.
func InternalProgress(format string, args ...interface{}) {
	Get().InternalProgress(format, args...)
}
