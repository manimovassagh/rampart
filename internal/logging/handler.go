package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"
)

// ANSI color codes.
const (
	reset   = "\033[0m"
	red     = "\033[31m"
	green   = "\033[32m"
	yellow  = "\033[33m"
	blue    = "\033[34m"
	magenta = "\033[35m"
	cyan    = "\033[36m"
	gray    = "\033[90m"
	white   = "\033[97m"
)

// PrettyHandler is a colorized, human-friendly slog.Handler.
type PrettyHandler struct {
	opts slog.HandlerOptions
	mu   sync.Mutex
	w    io.Writer
}

// NewPrettyHandler creates a colorized log handler writing to w.
func NewPrettyHandler(w io.Writer, opts *slog.HandlerOptions) *PrettyHandler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}
	return &PrettyHandler{opts: *opts, w: w}
}

// Enabled reports whether the handler handles records at the given level.
func (h *PrettyHandler) Enabled(_ context.Context, level slog.Level) bool {
	minLevel := slog.LevelInfo
	if h.opts.Level != nil {
		minLevel = h.opts.Level.Level()
	}
	return level >= minLevel
}

// Handle formats and writes a log record with ANSI colors.
func (h *PrettyHandler) Handle(_ context.Context, r slog.Record) error { //nolint:gocritic // slog.Handler interface requires value receiver
	timeStr := r.Time.Format(time.DateTime)
	levelStr, levelColor := formatLevel(r.Level)

	h.mu.Lock()
	defer h.mu.Unlock()

	// Time | Level | Message
	_, _ = fmt.Fprintf(h.w, "%s%s%s %s%-5s%s %s%s%s",
		gray, timeStr, reset,
		levelColor, levelStr, reset,
		white, r.Message, reset,
	)

	// Attributes
	r.Attrs(func(a slog.Attr) bool {
		_, _ = fmt.Fprintf(h.w, " %s%s%s=%s%s%s",
			cyan, a.Key, reset,
			colorForValue(a.Key, a.Value), a.Value.String(), reset,
		)
		return true
	})

	_, _ = fmt.Fprintln(h.w)
	return nil
}

// WithAttrs returns the handler unchanged (attributes are not accumulated).
func (h *PrettyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

// WithGroup returns the handler unchanged (groups are not supported).
func (h *PrettyHandler) WithGroup(name string) slog.Handler {
	return h
}

func formatLevel(level slog.Level) (label, color string) {
	switch {
	case level >= slog.LevelError:
		return "ERROR", red
	case level >= slog.LevelWarn:
		return "WARN", yellow
	case level >= slog.LevelInfo:
		return "INFO", green
	default:
		return "DEBUG", blue
	}
}

func colorForValue(key string, _ slog.Value) string {
	switch key {
	case "status":
		return yellow
	case "method":
		return magenta
	case "path":
		return cyan
	case "duration_ms":
		return yellow
	case "error":
		return red
	default:
		return gray
	}
}
