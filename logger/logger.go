package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
)

var (
	Logger       *slog.Logger
	StdoutLogger *slog.Logger
	// logLevel allows dynamic log level changes for the stderr logger
	logLevel slog.LevelVar
)

func init() {
	logLevel.Set(slog.LevelWarn)

	Logger = slog.New(NewCustomTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: &logLevel,
	}))

	StdoutLogger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	slog.SetDefault(Logger)
}

// Init sets the log level (called from main to override default)
func Init(level slog.Level) {
	logLevel.Set(level)
}

// CustomTextHandler formats logs in our preferred format: HH:MM:SS - LEVEL - message key=value
type CustomTextHandler struct {
	opts   slog.HandlerOptions
	writer io.Writer
	mu     sync.Mutex // protects writer
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
			value = fmt.Sprintf("%q", v)
		case []string:
			value = fmt.Sprintf("[%s]", strings.Join(v, " "))
		default:
			value = fmt.Sprintf("%v", v)
		}
		sb.WriteString(fmt.Sprintf(" %s=%s", a.Key, value))
		return true
	})

	sb.WriteString("\n")

	h.mu.Lock()
	_, err := io.WriteString(h.writer, sb.String())
	h.mu.Unlock()
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

// SetLogLevel changes the log level dynamically at runtime
func SetLogLevel(level slog.Level) {
	logLevel.Set(level)
}

// ToggleDebug toggles between debug and info log levels
func ToggleDebug() {
	if logLevel.Level() == slog.LevelDebug {
		logLevel.Set(slog.LevelInfo)
	} else {
		logLevel.Set(slog.LevelDebug)
	}
}

// IsDebugEnabled returns true if debug level logging is enabled
func IsDebugEnabled() bool {
	return logLevel.Level() == slog.LevelDebug
}

// IsVerboseEnabled returns true if verbose logging is enabled (info level or lower)
func IsVerboseEnabled() bool {
	return logLevel.Level() <= slog.LevelInfo
}
