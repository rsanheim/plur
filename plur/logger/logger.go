package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

var (
	// Logger is the global slog logger instance
	Logger *slog.Logger

	// VerboseMode indicates if verbose logging is enabled
	VerboseMode bool
)

// CustomTextHandler formats logs in our preferred format: HH:MM:SS - LEVEL - message key=value
type CustomTextHandler struct {
	opts   slog.HandlerOptions
	writer io.Writer
}

func NewCustomTextHandler(w io.Writer, opts *slog.HandlerOptions) *CustomTextHandler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}
	return &CustomTextHandler{
		opts:   *opts,
		writer: w,
	}
}

func (h *CustomTextHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.opts.Level.Level()
}

func (h *CustomTextHandler) Handle(_ context.Context, r slog.Record) error {
	timestamp := r.Time.Format("15:04:05")

	level := strings.ToUpper(r.Level.String())
	level = fmt.Sprintf("%-5s", level)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s - %s - %s", timestamp, level, r.Message))

	// Add attributes
	r.Attrs(func(a slog.Attr) bool {
		// Format value based on type
		var value string
		switch v := a.Value.Any().(type) {
		case string:
			value = v
		case []string:
			value = fmt.Sprintf("[%s]", strings.Join(v, " "))
		default:
			value = fmt.Sprintf("%v", v)
		}
		sb.WriteString(fmt.Sprintf(" %s=%s", a.Key, value))
		return true
	})

	sb.WriteString("\n")

	_, err := io.WriteString(h.writer, sb.String())
	return err
}

func (h *CustomTextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// For simplicity, we don't implement attribute inheritance
	return h
}

func (h *CustomTextHandler) WithGroup(name string) slog.Handler {
	// For simplicity, we don't implement groups
	return h
}

// InitLogger initializes the slog logger based on the verbose flag and debug mode
func InitLogger(verbose bool, debug bool) {
	VerboseMode = verbose

	var level slog.Level
	if debug {
		level = slog.LevelDebug
	} else if verbose {
		level = slog.LevelInfo
	} else {
		level = slog.LevelInfo
	}

	// Create custom text handler that writes to stderr
	opts := &slog.HandlerOptions{
		Level: level,
	}

	handler := NewCustomTextHandler(os.Stderr, opts)
	Logger = slog.New(handler)

	// Set as default logger
	slog.SetDefault(Logger)
}

// LogVerbose logs a message only if verbose mode is enabled
func LogVerbose(msg string, args ...any) {
	if VerboseMode {
		Logger.Info(msg, args...)
	}
}

// LogDebug logs a debug message (only shown with PLUR_DEBUG=1)
func LogDebug(msg string, args ...any) {
	Logger.Debug(msg, args...)
}

// LogError logs an error message (always shown)
func LogError(msg string, err error, args ...any) {
	allArgs := append([]any{"error", err}, args...)
	Logger.Error(msg, allArgs...)
}

// LogWarn logs a warning message (always shown)
func LogWarn(msg string, args ...any) {
	Logger.Warn(msg, args...)
}

// WithContext returns a logger with context values
func WithContext(ctx context.Context) *slog.Logger {
	return Logger
}

// WithWorker returns a logger with worker ID context
func WithWorker(workerID int) *slog.Logger {
	return Logger.With("worker", workerID)
}

// WithFile returns a logger with file context
func WithFile(file string) *slog.Logger {
	return Logger.With("file", file)
}
