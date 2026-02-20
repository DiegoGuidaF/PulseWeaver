package logging

import (
	"log/slog"
	"os"
	"strings"

	"github.com/lmittmann/tint"
)

// Tint 8-bit color codes (high intensity 8-15): 9 red, 12 blue, 13 magenta, 14 cyan.
const (
	tintColorOperation = 13 // bright magenta
	tintColorClientIP  = 14 // bright cyan
	tintColorRequestID = 12 // bright blue
	tintColorError     = 9  // bright red
)

// tintReplaceAttr customized selected attribute keys color (tint handler only).
func tintReplaceAttr(groups []string, a slog.Attr) slog.Attr {
	if len(groups) != 0 {
		return a
	}
	switch a.Key {
	case "operation":
		return tint.Attr(tintColorOperation, a)
	case "client_ip":
		return tint.Attr(tintColorClientIP, a)
	case "request_id":
		return tint.Attr(tintColorRequestID, a)
	case "error":
		return tint.Attr(tintColorError, a)
	}
	// Color any attribute whose value is an error (e.g. key might vary)
	if a.Value.Kind() == slog.KindAny {
		if _, ok := a.Value.Any().(error); ok {
			return tint.Attr(tintColorError, a)
		}
	}
	return a
}

// ParseLevel converts a string (e.g. "debug", "info", "warn", "error") to slog.Level.
// Invalid or empty values default to slog.LevelInfo.
func ParseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "info", "":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

type Format string

const JSONFormat Format = "json"

// Options configures logger creation.
// use Level (e.g. info) and Format (e.g. json).
type Options struct {
	Level  slog.Level
	Format Format
	Color  bool // Enable colored output for tint format (ignored for JSON format)
}

// New creates a slog.Logger from the given options.
// Format "json" uses slog.JSONHandler; otherwise tint is used with ReplaceAttr for colored keys.
// When using tint, color is controlled by the Color option (set via LOG_COLOR env var).
func New(opts Options) *slog.Logger {
	level := opts.Level
	if level == 0 {
		level = slog.LevelInfo
	}

	if opts.Format == JSONFormat {
		handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: level,
		})
		return slog.New(handler)
	}

	w := os.Stdout
	handler := tint.NewHandler(w, &tint.Options{
		Level:       level,
		NoColor:     !opts.Color,
		ReplaceAttr: tintReplaceAttr,
	})
	return slog.New(handler)
}
