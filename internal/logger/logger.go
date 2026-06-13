// Package logger provides structured logging configuration using slog.
package logger

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// Options configures a logger instance.
type Options struct {
	// Level is the minimum log level: debug, info, warn, or error. Defaults to info.
	Level string
	// Format is the output format: text or json. Defaults to text.
	Format string
	// Output is the write destination. Defaults to os.Stdout.
	Output io.Writer
	// AddSource includes the caller's file name and line number in each entry.
	AddSource bool
	// SetDefault replaces the global slog default logger with the new instance.
	SetDefault bool
	// Attrs are additional key-value pairs added to every log entry.
	Attrs []any
}

// New creates a slog.Logger from the provided options.
func New(o Options) *slog.Logger {
	out := o.Output
	if out == nil {
		out = os.Stdout
	}

	handler := buildHandler(out, o.Level, o.Format, o.AddSource)

	log := slog.New(handler)
	if len(o.Attrs) > 0 {
		log = log.With(o.Attrs...)
	}

	if o.SetDefault {
		slog.SetDefault(log)
	}

	return log
}

// ParseLevel converts a string level to the corresponding slog.Level.
// Unrecognised values default to slog.LevelInfo.
func ParseLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// buildHandler constructs a slog.Handler for the given format.
// Extracted so it can be tested independently of New.
func buildHandler(out io.Writer, level, format string, addSource bool) slog.Handler {
	opts := &slog.HandlerOptions{
		Level:     ParseLevel(level),
		AddSource: addSource,
	}
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "json":
		return slog.NewJSONHandler(out, opts)
	default:
		return slog.NewTextHandler(out, opts)
	}
}
