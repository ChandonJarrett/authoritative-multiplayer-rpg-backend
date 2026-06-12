// Package logger provides structured logging configuration using slog.
package logger

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// Options defines logger configuration.
type Options struct {
	Level      string
	Format     string
	Output     io.Writer
	AddSource  bool
	SetDefault bool
	Attrs      []any
}

// New creates a slog.Logger from options.
func New(o Options) *slog.Logger {
	out := o.Output
	if out == nil {
		out = os.Stdout
	}

	opts := &slog.HandlerOptions{
		Level:     ParseLevel(o.Level),
		AddSource: o.AddSource,
	}

	var handler slog.Handler
	switch strings.ToLower(strings.TrimSpace(o.Format)) {
	case "json":
		handler = slog.NewJSONHandler(out, opts)
	default:
		handler = slog.NewTextHandler(out, opts)
	}

	log := slog.New(handler)

	if len(o.Attrs) > 0 {
		log = log.With(o.Attrs...)
	}

	if o.SetDefault {
		slog.SetDefault(log)
	}

	return log
}

// ParseLevel converts a string log level into a slog level.
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
